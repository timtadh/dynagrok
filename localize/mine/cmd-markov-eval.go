package mine

import (
	"fmt"
	"sort"
	"strings"
)

import (
	"github.com/timtadh/getopt"
	"github.com/timtadh/matrix"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/lattice"
)

func NewMarkovEvalParser(c *cmd.Config, o *Options) cmd.Runnable {
	return cmd.Cmd(
		"markov-eval",
		`[options]`,
		`
Evaluate a fault localization method from ground truth

Option Flags
    -h,--help                         Show this message
    -f,--faults=<path>                Path to a fault file.
`,
		"f:",
		[]string{
			"faults=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			faultsPath := ""
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
				}
			}
			if faultsPath == "" {
				return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
			}
			faults, err := LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			for _, f := range faults {
				fmt.Println(f)
			}
			if o.Score == nil {
				for name, score := range Scores {
					m := NewMiner(o.Miner, o.Lattice, score, o.Opts...)
					colors, P := DsgMarkovChain(m)
					MarkovEval(faults, o.Lattice, "mine-dsg + "+name, colors, P)
					colors, P = RankListMarkovChain(m)
					MarkovEval(faults, o.Lattice, name, colors, P)
					colors, P = RankListWithJumpsMarkovChain(m)
					MarkovEval(faults, o.Lattice, "jumps + " + name, colors, P)
				}
			} else {
				m := NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
				colors, P := DsgMarkovChain(m)
				MarkovEval(faults, o.Lattice, "mine-dsg + "+o.ScoreName, colors, P)
				colors, P = RankListMarkovChain(m)
				MarkovEval(faults, o.Lattice, o.ScoreName, colors, P)
				colors, P = RankListWithJumpsMarkovChain(m)
				MarkovEval(faults, o.Lattice, "jumps + " + o.ScoreName, colors, P)
			}
			return nil, nil
		})
}

func MarkovEval(faults []*Fault, lat *lattice.Lattice, name string, colorStates map[int][]int, P [][]float64) {
	group := func(order []int, scores map[int]float64) [][]int {
		sort.Slice(order, func(i, j int) bool {
			return scores[order[i]] < scores[order[j]]
		})
		groups := make([][]int, 0, 10)
		for _, n := range order {
			lg := len(groups)
			if lg > 0 && scores[n] == scores[groups[lg-1][0]] {
				groups[lg-1] = append(groups[lg-1], n)
			} else {
				groups = append(groups, make([]int, 0, 10))
				groups[lg] = append(groups[lg], n)
			}
		}
		return groups
	}
	scores := make(map[int]float64)
	order := make([]int, 0, len(colorStates))
	for color, states := range colorStates {
		order = append(order, color)
		for _, state := range states {
			hit := ExpectedHittingTime(0, state, P)
			if min, has := scores[color]; !has || hit < min {
				scores[color] = hit
			}
		}
	}
	grouped := group(order, scores)
	ranks := make(map[int]float64)
	total := 0
	for gid, group := range grouped {
		count := 0
		for _, color := range group {
			score := scores[color]
			count++
			ranks[color] = float64(total) + float64(len(group))/2
			b, fn, pos := lat.Info.Get(color)
			if false {
				fmt.Printf(
					"    {\n\tgroup: %v, size: %d,\n\trank: %v, hitting time: %v,\n\tfn: %v (%d),\n\tpos: %v\n    }\n",
					gid, len(group),
					ranks[color],
					score,
					fn, b, pos,
				)
			}
		}
		total += len(group)
	}
	for _, f := range faults {
		for color, score := range scores {
			b, fn, pos := lat.Info.Get(color)
			if fn == f.FnName && b == f.BasicBlockId {
				fmt.Printf(
					"    %v {\n\trank: %v,\n\thitting time: %v,\n\tfn: %v (%d),\n\tpos: %v\n    }\n",
					name,
					ranks[color],
					score,
					fn, b, pos,
				)
				break
			}
		}
	}
}

func RankListMarkovChain(m *Miner) (blockStates map[int][]int, P [][]float64) {
	groups := LocalizeNodes(m.Score).Group()
	groupStates := make(map[int]int)
	blockStates = make(map[int][]int)
	blocks := 0
	states := 0
	for gid, group := range groups {
		groupStates[gid] = states
		states++
		for _, n := range group {
			blockStates[n.Color] = append(blockStates[n.Color], states)
			states++
			blocks++
		}
	}
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	for gid, group := range groups {
		groupState := groupStates[gid]
		for _, n := range group {
			for _, blockState := range blockStates[n.Color] {
				P[groupState][blockState] = (1./2.) * 1/float64(len(group))
				P[blockState][groupState] = 1
			}
		}
		if gid > 0 {
			prev := groupStates[gid - 1]
			P[groupState][prev] = (1./2.) * float64(blocks - 1)/float64(blocks)
			P[prev][groupState] = (1./2.) * 1/float64(blocks)
		}
	}
	first := groupStates[0]
	last := groupStates[len(groups)-1]
	P[first][first] = (1./2.) * float64(blocks - 1)/float64(blocks)
	P[last][last]   = (1./2.) * 1/float64(blocks)
	if false {
		for _, row := range P {
			cols := make([]string, 0, len(row))
			for _, col := range row {
				cols = append(cols, fmt.Sprintf("%.3g", col))
			}
			fmt.Println(">>", strings.Join(cols, ", "))
		}
	}
	return blockStates, P
}

func RankListWithJumpsMarkovChain(m *Miner) (blockStates map[int][]int, P [][]float64) {
	jumpPr := 1./10.
	groups := LocalizeNodes(m.Score).Group()
	groupStates := make(map[int]int)
	blockStates = make(map[int][]int)
	blocks := 0
	states := 0
	for gid, group := range groups {
		groupStates[gid] = states
		states++
		for _, n := range group {
			blockStates[n.Color] = append(blockStates[n.Color], states)
			states++
			blocks++
		}
	}
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	for gid, group := range groups {
		groupState := groupStates[gid]
		for _, n := range group {
			for _, blockState := range blockStates[n.Color] {
				P[groupState][blockState] = (1./2.) * 1/float64(len(group))
				P[blockState][groupState] = 1. - jumpPr
				edgesFrom := m.Lattice.Fail.EdgesFromColor[n.Color]
				edgesTo := m.Lattice.Fail.EdgesToColor[n.Color]
				degree := float64(len(edgesFrom) + len(edgesTo))
				for _, e := range edgesFrom {
					for _, targState := range blockStates[e.TargColor] {
						P[blockState][targState] = jumpPr * (1./degree)
					}
				}
				for _, e := range edgesTo {
					for _, srcState := range blockStates[e.SrcColor] {
						P[blockState][srcState] = jumpPr * (1./degree)
					}
				}
			}
		}
		if gid > 0 {
			prev := groupStates[gid - 1]
			P[groupState][prev] = (1./2.) * float64(blocks - 1)/float64(blocks)
			P[prev][groupState] = (1./2.) * 1/float64(blocks)
		}
	}
	first := groupStates[0]
	last := groupStates[len(groups)-1]
	P[first][first] = (1./2.) * float64(blocks - 1)/float64(blocks)
	P[last][last]   = (1./2.) * 1/float64(blocks)
	return blockStates, P
}

func DsgMarkovChain(m *Miner) (blockStates map[int][]int, P [][]float64) {
	groups := m.Mine().group()
	type graph struct {
		gid int
		nid int
	}
	groupStates := make(map[int]int)
	graphStates := make(map[graph]int)
	blockStates = make(map[int][]int)
	graphsPerColor := make(map[int]int)
	graphs := 0
	states := 0
	for gid, group := range groups {
		groupStates[gid] = states
		states++
		for nid, n := range group {
			graphStates[graph{gid,nid}] = states
			states++
			graphs++
			unique := make(map[int]bool)
			for _, v := range n.Node.SubGraph.V {
				unique[v.Color] = true
			}
			for color := range unique {
				if _, has := blockStates[color]; !has {
					blockStates[color] = append(blockStates[color], states)
					states++
				}
				graphsPerColor[color]++
			}
		}
	}
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	for gid, group := range groups {
		groupState := groupStates[gid]
		for nid, n := range group {
			graphState := graphStates[graph{gid,nid}]
			P[groupState][graphState] = (1./2.) * 1/float64(len(group))
			P[graphState][groupState] = (1./2.)
			unique := make(map[int]bool)
			for _, v := range n.Node.SubGraph.V {
				unique[v.Color] = true
			}
			for color := range unique {
				for _, blockState := range blockStates[color] {
					P[graphState][blockState] = (1./2.) * 1/float64(len(unique))
					P[blockState][graphState] = 1/float64(graphsPerColor[color])
				}
			}
		}
		if gid > 0 {
			prev := groupStates[gid - 1]
			P[groupState][prev] = (1./2.) * float64(graphs - 1)/float64(graphs)
			P[prev][groupState] = (1./2.) * 1/float64(graphs)
		}
	}
	first := groupStates[0]
	last := groupStates[len(groups)-1]
	P[first][first] = (1./2.) * float64(graphs - 1)/float64(graphs)
	P[last][last]   = (1./2.) * 1/float64(graphs)
	if false {
		fmt.Println("group", groupStates)
		fmt.Println("graph", graphStates)
		fmt.Println("block", blockStates)
		headers := make([]string, 0, len(P))
		for i := range P {
			headers = append(headers, fmt.Sprintf("%5d", i))
		}
		fmt.Printf("%3v -- %v\n", "", strings.Join(headers, ", "))
		for i, row := range P {
			cols := make([]string, 0, len(row))
			for _, col := range row {
				if col == 0 {
					cols = append(cols, fmt.Sprintf("%5v", ""))
				} else {
					cols = append(cols, fmt.Sprintf("%5.2g", col))
				}
			}
			fmt.Printf("%3d >> %v\n", i, strings.Join(cols, ", "))
		}
	}
	return blockStates, P
}

func ExpectedHittingTimes(transitions [][]float64) [][]float64 {
	P := matrix.MakeDenseMatrixStacked(transitions)
	M := matrix.Zeros(P.Rows(), P.Cols())
	for i := 0; i < P.Rows()*P.Rows(); i++ {
		prevM := M.Copy()
		for t := 0; t < M.Rows(); t++ {
			for s := 0; s < M.Cols(); s++ {
				if t == s {
					M.Set(t, s, 0.0)
				} else {
					sum := 0.0
					for k := 0; k < P.Rows(); k++ {
						if k != s {
							sum += P.Get(t, k) * (M.Get(k, s) + 1)
						}
					}
					M.Set(t, s, P.Get(t, s) + sum)
				}
			}
		}
		diff, err := M.Minus(prevM)
		if err != nil {
			panic(err)
		}
		if diff.DenseMatrix().TwoNorm() < .01 {
			break
		}
	}
	return M.Arrays()
}

func ExpectedHittingTime(x, y int, transitions [][]float64) float64 {
	P := matrix.MakeDenseMatrixStacked(transitions)
	for s := 0; s < P.Cols(); s++ {
		P.Set(y, s, 0)
	}
	P.Set(y, y, 1)
	last := P.Rows()
	P.SwapRows(y, last-1)
	P = P.Transpose()
	P.SwapRows(y, last-1)
	P = P.Transpose()
	Q := P.GetMatrix(0, 0, last-1, last-1)
	I := matrix.Eye(Q.Rows())
	c := matrix.Ones(Q.Rows(), 1)
	IQ, err := I.Minus(Q)
	if err != nil {
		panic(err)
	}
	N, err := IQ.DenseMatrix().Inverse()
	if err != nil {
		panic(err)
	}
	Nc, err := N.Times(c)
	if err != nil {
		panic(err)
	}
	// fmt.Println(x, y, Nc.Get(x,0))
	return Nc.Get(x, 0)
}
