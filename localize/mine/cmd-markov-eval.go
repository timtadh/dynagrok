package mine

import (
	"math"
	"bytes"
	"fmt"
	"sort"
	"strings"
	"strconv"
	"encoding/json"
	"os/exec"
	"runtime"
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
    -m,--max=<int>                    Maximum number of states in the chain
`,
		"f:m:",
		[]string{
			"faults=",
			"max=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			max := 1000000
			faultsPath := ""
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
				case "-m", "--max":
					var err error
					max, err = strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "For flag %v expected an int got %v. err: %v", oa.Opt, oa.Arg(), err)
					}
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
					colors, P := DsgMarkovChain(max, m)
					MarkovEval(faults, o.Lattice, "mine-dsg + "+name, colors, P)
					colors, P = RankListMarkovChain(max, m)
					MarkovEval(faults, o.Lattice, name, colors, P)
					colors, P = BehaviorJumps(max, m)
					MarkovEval(faults, o.Lattice, "behavioral jumps + " + name, colors, P)
					colors, P = SpacialJumps(max, m)
					MarkovEval(faults, o.Lattice, "spacial jumps + " + name, colors, P)
				}
			} else {
				m := NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
				colors, P := DsgMarkovChain(max, m)
				MarkovEval(faults, o.Lattice, "mine-dsg + "+o.ScoreName, colors, P)
				colors, P = RankListMarkovChain(max, m)
				MarkovEval(faults, o.Lattice, o.ScoreName, colors, P)
				colors, P = BehaviorJumps(max, m)
				MarkovEval(faults, o.Lattice, "behavioral jumps + " + o.ScoreName, colors, P)
				colors, P = SpacialJumps(max, m)
				MarkovEval(faults, o.Lattice, "spacial jumps + " + o.ScoreName, colors, P)
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
	order := make([]int, 0, len(colorStates))
	states := make([]int, 0, len(colorStates))
	for color, group := range colorStates {
		order = append(order, color)
		for _, state := range group {
			states = append(states, state)
		}
	}
	scores := make(map[int]float64)
	hittingTimes, err := ParPyEHT(0, states, P)
	if err != nil {
		fmt.Println(err)
		fmt.Println("falling back on go implementation of hittingTime computation")
		for color, states := range colorStates {
			for _, state := range states {
				hit := ExpectedHittingTime(0, state, P)
				if min, has := scores[color]; !has || hit < min {
					scores[color] = hit
				}
			}
		}
	} else {
		for color, states := range colorStates {
			for _, state := range states {
				hit, has := hittingTimes[state]
				if !has {
					continue
				}
				if min, has := scores[color]; !has || hit < min {
					scores[color] = hit
				}
			}
		}
	}
	for color, hit := range scores {
		if hit <= 0 {
			scores[color] = math.Inf(1)
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
					"    markov-eval %v {\n\trank: %v,\n\thitting time: %v,\n\tfn: %v (%d),\n\tpos: %v\n    }\n",
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

func RankListMarkovChain(max int, m *Miner) (blockStates map[int][]int, P [][]float64) {
	jumpPr := 0.0
	groups := LocalizeNodes(m.Score).Group()
	colors := make([][]int, 0, len(groups))
	jumps := make(map[int]map[int]bool)
	for _, group := range groups {
		colorGroup := make([]int, 0, len(group))
		for _, n := range group {
			if _, has := jumps[n.Color]; !has {
				jumps[n.Color] = make(map[int]bool)
			}
			colorGroup = append(colorGroup, n.Color)
		}
		colors = append(colors, colorGroup)
	}
	return RankListWithJumpsMarkovChain(max, colors, jumpPr, jumps)
}

func RankListWithJumpsMarkovChain(max int, groups [][]int, prJump float64, jumps map[int]map[int]bool) (blockStates map[int][]int, P [][]float64) {
	groupStates := make(map[int]int)
	blockStates = make(map[int][]int)
	states := 0
	grpState := func(gid int) int {
		if s, has := groupStates[gid]; has {
			return s
		} else {
			state := states
			states++
			groupStates[gid] = state
			return state
		}
	}
	blkState := func(color int) int {
		if s, has := blockStates[color]; has {
			return s[0]
		} else {
			state := states
			states++
			blockStates[color] = append(blockStates[color], state)
			return state
		}
	}
	maxGid := -1
	for gid, group := range groups {
		if states >= max {
			maxGid = gid
			fmt.Println("warning hit max states", states, max)
			break
		}
		grpState(gid)
		for _, color := range group {
			blkState(color)
		}
	}
	if maxGid < 0 {
		maxGid = len(groups)
	}
	blocks := len(blockStates)
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	for gid, group := range groups {
		if gid >= maxGid {
			break
		}
		gState := grpState(gid)
		for _, color := range group {
			bState := blkState(color)
			P[gState][bState] = (1./2.) * 1/float64(len(group))
			P[bState][gState] = 1 - prJump
			totalJumps := 0
			for nColor := range jumps[color] {
				if _, has := blockStates[nColor]; has {
					totalJumps++
				}
			}
			for nColor := range jumps[color] {
				if neighbors, has := blockStates[nColor]; has {
					for _, neighbor := range neighbors {
						P[bState][neighbor] = prJump * 1/float64(totalJumps)
					}
				}
			}
		}
		if gid > 0 {
			prev := grpState(gid - 1)
			P[gState][prev] = (1./2.) * float64(blocks - 1)/float64(blocks)
			P[prev][gState] = (1./2.) * 1/float64(blocks)
		}
	}
	first := grpState(0)
	last := grpState(len(groupStates)-1)
	P[first][first] = (1./2.) * float64(blocks - 1)/float64(blocks)
	P[last][last]   = (1./2.) * 1/float64(blocks)
	return blockStates, P
}

func BehaviorJumps(max int, m *Miner) (blockStates map[int][]int, P [][]float64) {
	jumpPr := 1./10.
	groups := LocalizeNodes(m.Score).Group()
	colors := make([][]int, 0, len(groups))
	jumps := make(map[int]map[int]bool)
	for _, group := range groups {
		colorGroup := make([]int, 0, len(group))
		for _, n := range group {
			if _, has := jumps[n.Color]; !has {
				jumps[n.Color] = make(map[int]bool)
			}
			colorGroup = append(colorGroup, n.Color)
			edgesFrom := m.Lattice.Fail.EdgesFromColor[n.Color]
			edgesTo := m.Lattice.Fail.EdgesToColor[n.Color]
			for _, e := range edgesFrom {
				jumps[n.Color][e.TargColor] = true
			}
			for _, e := range edgesTo {
				jumps[n.Color][e.SrcColor] = true
			}
		}
		colors = append(colors, colorGroup)
	}
	return RankListWithJumpsMarkovChain(max, colors, jumpPr, jumps)
}

func SpacialJumps(max int, m *Miner) (blockStates map[int][]int, P [][]float64) {
	jumpPr := 1./10.
	groups := LocalizeNodes(m.Score).Group()
	colors := make([][]int, 0, len(groups))
	jumps := make(map[int]map[int]bool)
	fnBlks := make(map[string][]int)
	for _, group := range groups {
		colorGroup := make([]int, 0, len(group))
		for _, n := range group {
			if _, has := jumps[n.Color]; !has {
				jumps[n.Color] = make(map[int]bool)
			}
			colorGroup = append(colorGroup, n.Color)
			_, fnName, _ := m.Lattice.Info.Get(n.Color)
			fnBlks[fnName] = append(fnBlks[fnName], n.Color)
		}
		colors = append(colors, colorGroup)
	}
	for _, blks := range fnBlks {
		for _, a := range blks {
			for _, b := range blks {
				if a == b {
					continue
				}
				jumps[a][b] = true
			}
		}
	}
	return RankListWithJumpsMarkovChain(max, colors, jumpPr, jumps)
}


func DsgMarkovChain(max int, m *Miner) (blockStates map[int][]int, P [][]float64) {
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
	maxGid := -1
	for gid, group := range groups {
		if states >= max {
			maxGid = gid
			fmt.Println("warning hit max states", states, max)
			break
		}
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
	if maxGid < 0 {
		maxGid = len(groups)
	}
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	for gid, group := range groups {
		if gid >= maxGid {
			break
		}
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
	last := groupStates[maxGid-1]
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

func ParPyEHT(start int, states []int, transitions [][]float64) (map[int]float64, error) {
	cpus := runtime.NumCPU()
	work := make([][]int, cpus)
	for i, state := range states {
		w := i % len(work)
		work[w] = append(work[w], state)
	}
	type result struct {
		hits map[int]float64
		err error
	}
	hits := make(map[int]float64, len(states))
	results := make(chan result)
	for w := range work {
		go func(mine []int) {
			hits, err := PyExpectedHittingTimes(start, mine, transitions)
			results<-result{hits, err}
		}(work[w])
	}
	var err error
	for i := 0; i < cpus; i++ {
		r :=<-results
		if r.err != nil {
			err = r.err
			continue
		}
		for state, time := range r.hits {
			hits[state] = time
		}
	}
	if err != nil {
		return nil, err
	}
	return hits, nil
}

func PyExpectedHittingTimes(start int, states []int, transitions [][]float64) (map[int]float64, error) {
	type data struct {
		Start int
		States []int
		Transitions [][]float64
	}
	encoded, err := json.Marshal(data{start, states, transitions})
	if err != nil {
		return nil, err
	}
	var outbuf, errbuf bytes.Buffer
	inbuf := bytes.NewBuffer(encoded)
	c := exec.Command("hitting-times")
	c.Stdin = inbuf
	c.Stdout = &outbuf
	c.Stderr = &errbuf
	err = c.Start()
	if err != nil {
		return nil, err
	}
	err = c.Wait()
	if err != nil {
		return nil, fmt.Errorf("py hitting time err: %v\n`%v`\n`%v`", err, errbuf.String(), outbuf.String())
	}
	stderr := errbuf.String()
	if len(stderr) > 0 {
		return nil, fmt.Errorf("py hitting time err: %v", stderr)
	}
	if !c.ProcessState.Success() {
		return nil, fmt.Errorf("failed to have python compute hitting times: %v", outbuf.String())
	}
	var times map[string]float64
	err = json.Unmarshal(outbuf.Bytes(), &times)
	if err != nil {
		return nil, fmt.Errorf("py hitting time err, could not unmarshall: %v\n`%v`", err, outbuf.String())
	}
	hits := make(map[int]float64)
	for sState, time := range times {
		state, err := strconv.Atoi(sState)
		if err != nil {
			return nil, fmt.Errorf("py hitting time err, could not unmarshall: %v\n`%v`", err, outbuf.String())
		}
		hits[state] = time
	}
	return hits, nil
}

