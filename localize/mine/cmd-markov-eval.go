package mine

import (
	"fmt"
	"sort"
	"math/rand"
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
	sum := func(slice []float64) float64 {
		sum := 0.0
		for _, x := range slice {
			sum += x
		}
		return sum
	}
	groups := LocalizeNodes(m.Score).Group()
	groupStates := make(map[int]int)
	blockStates = make(map[int][]int)
	states := 0
	for gid, group := range groups {
		groupStates[gid] = states
		states++
		for _, n := range group {
			blockStates[n.Color] = append(blockStates[n.Color], states)
			states++
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
				P[groupState][blockState] = 1
				P[blockState][groupState] = 1
			}
		}
		if gid > 0 {
			prev := groupStates[gid - 1]
			P[prev][groupState] = 1
			P[groupState][prev] = 1 + rand.Float64()
			P[groupState][prev] = 1
		}
	}
	for _, row := range P {
		total := sum(row)
		if total == 0 {
			continue
		}
		for state := range row {
			row[state] = row[state]/total
		}
	}
	return blockStates, P
}

func RankListWithJumpsMarkovChain(m *Miner) (blockStates map[int][]int, P [][]float64) {
	sum := func(slice []float64) float64 {
		sum := 0.0
		for _, x := range slice {
			sum += x
		}
		return sum
	}
	groups := LocalizeNodes(m.Score).Group()
	groupStates := make(map[int]int)
	blockStates = make(map[int][]int)
	states := 0
	for gid, group := range groups {
		groupStates[gid] = states
		states++
		for _, n := range group {
			blockStates[n.Color] = append(blockStates[n.Color], states)
			states++
		}
	}
	type pair struct { i, j int}
	T := make(map[pair]float64)
	for gid, group := range groups {
		groupState := groupStates[gid]
		for _, n := range group {
			for _, blockState := range blockStates[n.Color] {
				T[pair{groupState, blockState}] = 1
				T[pair{blockState, groupState}] = 1
			}
		}
		if gid > 0 {
			prev := groupStates[gid - 1]
			T[pair{prev, groupState}] = 1
			T[pair{groupState, prev}] = 1 + rand.Float64()
			T[pair{groupState, prev}] = 1
		}
	}
	for color, colorStates := range blockStates {
		for _, state := range colorStates {
			edgesFrom := m.Lattice.Fail.EdgesFromColor[color]
			degree := float64(len(edgesFrom))
			for _, e := range edgesFrom {
				for _, targState := range blockStates[e.TargColor] {
					T[pair{state, targState}] = 1/degree
				}
			}
		}
	}
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	for key, entry := range T {
		P[key.i][key.j] = entry
	}
	for _, row := range P {
		total := sum(row)
		if total == 0 {
			continue
		}
		for state := range row {
			row[state] = row[state]/total
		}
	}
	return blockStates, P
}

func DsgMarkovChain(m *Miner) (blockStates map[int][]int, P [][]float64) {
	sum := func(slice []float64) float64 {
		sum := 0.0
		for _, x := range slice {
			sum += x
		}
		return sum
	}
	groups := m.Mine().group()
	groupStates := make(map[int]int)
	graphStates := make(map[int]int)
	blockStates = make(map[int][]int)
	states := 0
	for gid, group := range groups {
		groupStates[gid] = states
		states++
		for nid, n := range group {
			graphStates[nid] = states
			states++
			for _, v := range n.Node.SubGraph.V {
				if _, has := blockStates[v.Color]; !has {
					blockStates[v.Color] = append(blockStates[v.Color], states)
					states++
				}
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
			graphState := graphStates[nid]
			P[groupState][graphState] = 1
			P[graphState][groupState] = 1
			for _, v := range n.Node.SubGraph.V {
				for _, blockState := range blockStates[v.Color] {
					P[graphState][blockState] = 1
					P[blockState][graphState] = 1
				}
			}
		}
		if gid > 0 {
			prev := groupStates[gid - 1]
			P[prev][groupState] = 1
			P[groupState][prev] = 1 + rand.Float64()
			P[groupState][prev] = 1
		}
	}
	for _, row := range P {
		total := sum(row)
		if total == 0 {
			continue
		}
		for state := range row {
			row[state] = row[state]/total
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
