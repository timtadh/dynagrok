package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
	"github.com/timtadh/dynagrok/localize/fault"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/opts"
	matrix "github.com/timtadh/go.matrix"
)

var Chains = map[string][]string{
	"CBSFL": []string{
		"Ranked-List",
		"Spacial-Jumps",
		"Behavioral-Jumps",
		"Behavioral+Spacial-Jumps",
	},
	"DISCFLO": []string{
		"DF-Jumps",
	},
	"DISCFLO + FP-Filter": []string{
		"DF-Jumps",
	},
	"SBBFL": []string{
		"SB-List",
	},
}

func Evaluate(faults []*fault.Fault, o *discflo.Options, score mine.ScoreFunc, evalName, methodName, scoreName, chainName string, maxStates int, jumpPrs []float64) (EvalResults, error) {
	m := mine.NewMiner(o.Miner, o.Lattice, score, o.Opts...)
	switch evalName {
	case "RankList":
		var groups [][]ColorScore
		if methodName == "CBSFL" {
			groups = CBSFL(&o.Options, score)
		} else if strings.HasPrefix(methodName, "DISCFLO") {
			groups = Discflo(o, score)
		} else {
			return nil, fmt.Errorf("no localization method named %v for eval method %v", methodName, evalName)
		}
		return RankListEval(faults, o.Lattice, methodName, scoreName, groups), nil
	case "Markov":
		results := make(EvalResults, 0, 10)
		for _, jumpPr := range jumpPrs {
			var colors map[int][]int
			var P [][]float64
			jumpChain := chainName
			if methodName == "CBSFL" {
				switch chainName {
				case "Ranked-List":
					colors, P = RankListMarkovChain(maxStates, m)
					return MarkovEval(faults, o.Lattice, methodName, scoreName, chainName, colors, P), nil
				case "Spacial-Jumps":
					colors, P = SpacialJumps(jumpPr, maxStates, m)
					jumpChain = fmt.Sprintf("%v(%g)", chainName, jumpPr)
				case "Behavioral-Jumps":
					colors, P = BehavioralJumps(jumpPr, maxStates, m)
					jumpChain = fmt.Sprintf("%v(%g)", chainName, jumpPr)
				case "Behavioral+Spacial-Jumps":
					colors, P = BehavioralAndSpacialJumps(jumpPr, maxStates, m)
					jumpChain = fmt.Sprintf("%v(%g)", chainName, jumpPr)
				default:
					return nil, fmt.Errorf("no chain named %v", methodName)
				}
			} else if methodName == "SBBFL" {
				colors, P = DsgMarkovChain(maxStates, m.Mine(context.TODO()).Unique(), 0, nil)
				return MarkovEval(faults, o.Lattice, methodName, scoreName, chainName, colors, P), nil
			} else if strings.HasPrefix(methodName, "DISCFLO") {
				var err error
				colors, P, err = DiscfloMarkovChain(jumpPr, maxStates, o, score)
				if err != nil {
					return nil, err
				}
				jumpChain = fmt.Sprintf("%v(%g)", chainName, jumpPr)
			} else {
				return nil, fmt.Errorf("no localization method named %v for eval method %v", methodName, evalName)
			}
			r := MarkovEval(faults, o.Lattice, methodName, scoreName, jumpChain, colors, P)
			results = append(results, r...)
		}
		return results, nil
	default:
		return nil, fmt.Errorf("no evaluation method named %v", evalName)
	}
}

type ColorScore struct {
	Color int
	Score float64
}

func CBSFL(o *opts.Options, s mine.ScoreFunc) [][]ColorScore {
	miner := mine.NewMiner(o.Miner, o.Lattice, s, o.Opts...)
	groups := make([][]ColorScore, 0, 10)
	for _, group := range mine.LocalizeNodes(miner.Score).Group() {
		colorGroup := make([]ColorScore, 0, len(group))
		for _, n := range group {
			colorGroup = append(colorGroup, ColorScore{n.Color, n.Score})
		}
		groups = append(groups, colorGroup)
	}
	return groups
}

func Discflo(o *discflo.Options, s mine.ScoreFunc) [][]ColorScore {
	miner := mine.NewMiner(o.Miner, o.Lattice, s, o.Opts...)
	c, err := discflo.Localizer(o)(miner)
	if err != nil {
		panic(err)
	}
	groups := make([][]ColorScore, 0, 10)
	for _, group := range c.RankColors(miner).ScoredLocations().Group() {
		colorGroup := make([]ColorScore, 0, len(group))
		for _, n := range group {
			colorGroup = append(colorGroup, ColorScore{n.Color, n.Score})
		}
		groups = append(groups, colorGroup)
	}
	return groups
}

func RankListEval(faults []*fault.Fault, lat *lattice.Lattice, methodName, scoreName string, groups [][]ColorScore) (results EvalResults) {
	for _, f := range faults {
		sum := 0
		for gid, group := range groups {
			for _, cs := range group {
				bbid, fnName, pos := lat.Info.Get(cs.Color)
				if fnName == f.FnName && bbid == f.BasicBlockId {
					fmt.Printf(
						"   %v + %v {\n        rank: %v, gid: %v, group-size: %v\n        score: %v,\n        fn: %v (%d),\n        pos: %v\n    }\n",
						methodName, scoreName,
						float64(sum)+float64(len(group))/2, gid, len(group),
						cs.Score,
						fnName,
						bbid,
						pos,
					)
					r := &RankListEvalResult{
						MethodName:     methodName,
						ScoreName:      scoreName,
						RankScore:      float64(sum) + float64(len(group))/2,
						Suspiciousness: cs.Score,
						LocalizedFault: f,
						Loc: &mine.Location{
							Color:        cs.Color,
							BasicBlockId: bbid,
							FnName:       fnName,
							Position:     pos,
						},
					}
					results = append(results, r)
				}
			}
			sum += len(group)
		}
	}
	return results
}

func MarkovEval(faults []*fault.Fault, lat *lattice.Lattice, methodName, scoreName, chainName string, colorStates map[int][]int, P [][]float64) (results EvalResults) {
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
					"    {\n        group: %v, size: %d,\n        rank: %v, hitting time: %v,\n        fn: %v (%d),\n        pos: %v\n    }\n",
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
					"    %v + %v + Markov%v {\n        rank: %v,\n        hitting time: %v,\n        fn: %v (%d),\n        pos: %v\n    }\n",
					methodName, scoreName, chainName,
					ranks[color],
					score,
					fn, b, pos,
				)
				r := &MarkovEvalResult{
					MethodName:  methodName,
					ScoreName:   scoreName,
					ChainName:   chainName,
					HT_Rank:     ranks[color],
					HittingTime: score,
					fault:       f,
					loc: &mine.Location{
						Color:        color,
						BasicBlockId: b,
						FnName:       fn,
						Position:     pos,
					},
				}
				results = append(results, r)
				break
			}
		}
	}
	return results
}

func DiscfloMarkovChain(jumpPr float64, max int, o *discflo.Options, score mine.ScoreFunc) (blockStates map[int][]int, P [][]float64, err error) {
	opts := o.Copy()
	opts.Score = score
	localizer := models.Localize(opts)
	clusters, err := localizer.Clusters()
	if err != nil {
		return nil, nil, err
	}
	groups := clusters.Blocks().Group()
	neighbors := make(map[int]map[int]bool)
	colors := make([][]int, 0, len(groups))
	for _, group := range groups {
		colorGroup := make([]int, 0, len(group))
		for _, block := range group {
			colorGroup = append(colorGroup, block.Color)
			neighbors[block.Color] = make(map[int]bool)
			for _, cluster := range block.In {
				for _, n := range cluster.Nodes {
					for _, v := range n.Node.SubGraph.V {
						neighbors[block.Color][v.Color] = true
					}
				}
			}
		}
		colors = append(colors, colorGroup)
	}
	blockStates, P = RankListWithJumpsMarkovChain(max, colors, jumpPr, neighbors)
	return blockStates, P, nil
}

func RankListMarkovChain(max int, m *mine.Miner) (blockStates map[int][]int, P [][]float64) {
	groups := mine.LocalizeNodes(m.Score).Group()
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
	return RankListWithJumpsMarkovChain(max, colors, 0, jumps)
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
			P[gState][bState] = (1. / 2.) * 1 / float64(len(group))
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
						P[bState][neighbor] = prJump * 1 / float64(totalJumps)
					}
				}
			}
		}
		if gid > 0 {
			prev := grpState(gid - 1)
			P[gState][prev] = (1. / 2.) * float64(blocks-1) / float64(blocks)
			P[prev][gState] = (1. / 2.) * 1 / float64(blocks)
		}
	}
	first := grpState(0)
	last := grpState(len(groupStates) - 1)
	P[first][first] = (1. / 2.) * float64(blocks-1) / float64(blocks)
	P[last][last] = (1. / 2.) * 1 / float64(blocks)
	return blockStates, P
}

func BehavioralJumpMatrix(m *mine.Miner) (groupsToColors [][]int, jumps map[int]map[int]bool) {
	groups := mine.LocalizeNodes(m.Score).Group()
	groupsToColors = make([][]int, 0, len(groups))
	jumps = make(map[int]map[int]bool)
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
		groupsToColors = append(groupsToColors, colorGroup)
	}
	return groupsToColors, jumps
}

func BehavioralJumps(jumpPr float64, max int, m *mine.Miner) (blockStates map[int][]int, P [][]float64) {
	groupsToColors, jumps := BehavioralJumpMatrix(m)
	return RankListWithJumpsMarkovChain(max, groupsToColors, jumpPr, jumps)
}

func SpacialJumpMatrix(m *mine.Miner) (groupsToColors [][]int, jumps map[int]map[int]bool) {
	groups := mine.LocalizeNodes(m.Score).Group()
	groupsToColors = make([][]int, 0, len(groups))
	jumps = make(map[int]map[int]bool)
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
		groupsToColors = append(groupsToColors, colorGroup)
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
	return groupsToColors, jumps
}

func SpacialJumps(jumpPr float64, max int, m *mine.Miner) (blockStates map[int][]int, P [][]float64) {
	groupsToColors, jumps := SpacialJumpMatrix(m)
	return RankListWithJumpsMarkovChain(max, groupsToColors, jumpPr, jumps)
}

func BehavioralAndSpacialJumpMatrix(m *mine.Miner) (groupsToColors [][]int, jumps map[int]map[int]bool) {
	groups := mine.LocalizeNodes(m.Score).Group()
	groupsToColors = make([][]int, 0, len(groups))
	jumps = make(map[int]map[int]bool)
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
			edgesFrom := m.Lattice.Fail.EdgesFromColor[n.Color]
			edgesTo := m.Lattice.Fail.EdgesToColor[n.Color]
			for _, e := range edgesFrom {
				jumps[n.Color][e.TargColor] = true
			}
			for _, e := range edgesTo {
				jumps[n.Color][e.SrcColor] = true
			}
		}
		groupsToColors = append(groupsToColors, colorGroup)
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
	return groupsToColors, jumps
}

func BehavioralAndSpacialJumps(jumpPr float64, max int, m *mine.Miner) (blockStates map[int][]int, P [][]float64) {
	groupsToColors, jumps := BehavioralAndSpacialJumpMatrix(m)
	return RankListWithJumpsMarkovChain(max, groupsToColors, jumpPr, jumps)
}

func ControlChain(jumps map[int]map[int]bool) (blockStates map[int][]int, P [][]float64) {
	blockStates = make(map[int][]int)
	states := 0
	start := 0
	states++
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
	for a := range jumps {
		blkState(a)
	}
	P = make([][]float64, 0, states)
	for i := 0; i < states; i++ {
		P = append(P, make([]float64, states))
	}
	returnPr := 0.01
	for a, aJumps := range jumps {
		P[start][a] = 1 / float64(len(blockStates))
		if len(aJumps) <= 0 {
			P[a][start] = 1
		} else {
			P[a][start] = returnPr
		}
		for b := range aJumps {
			P[a][b] = (1 / float64(len(aJumps))) * (1 - returnPr)
		}
	}
	return blockStates, P
}

func DsgMarkovChain(max int, nodes []*mine.SearchNode, jumpPr float64, jumps map[int]map[int]bool) (blockStates map[int][]int, P [][]float64) {
	groups := mine.GroupNodesByScore(nodes)
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
			graphStates[graph{gid, nid}] = states
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
			graphState := graphStates[graph{gid, nid}]
			P[groupState][graphState] = (1. / 2.) * 1 / float64(len(group))
			P[graphState][groupState] = (1. / 2.)
			unique := make(map[int]bool)
			for _, v := range n.Node.SubGraph.V {
				unique[v.Color] = true
			}
			for color := range unique {
				for _, blockState := range blockStates[color] {
					P[graphState][blockState] = (1. / 2.) * 1 / float64(len(unique))
					P[blockState][graphState] = (1 / float64(graphsPerColor[color])) * (1 - jumpPr)
				}
			}
		}
		if gid > 0 {
			prev := groupStates[gid-1]
			P[groupState][prev] = (1. / 2.) * float64(graphs-1) / float64(graphs)
			P[prev][groupState] = (1. / 2.) * 1 / float64(graphs)
		}
	}
	if jumpPr > 0 {
		for a, aStates := range blockStates {
			aTotal := 0
			for b := range jumps[a] {
				for range aStates {
					for range blockStates[b] {
						aTotal++
					}
				}
			}
			for b, bStates := range blockStates {
				if jumps[a][b] {
					for _, A := range aStates {
						for _, B := range bStates {
							P[A][B] = jumpPr * (1 / float64(aTotal))
						}
					}
				}
			}
		}
	}
	first := groupStates[0]
	last := groupStates[maxGid-1]
	P[first][first] = (1. / 2.) * float64(graphs-1) / float64(graphs)
	P[last][last] = (1. / 2.) * 1 / float64(graphs)
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
					M.Set(t, s, P.Get(t, s)+sum)
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
	if states == nil {
		panic("states is nil")
	}
	cpus := runtime.NumCPU()
	work := make([][]int, cpus)
	for i, state := range states {
		w := i % len(work)
		work[w] = append(work[w], state)
	}
	type result struct {
		hits map[int]float64
		err  error
	}
	hits := make(map[int]float64, len(states))
	results := make(chan result)
	expected := 0
	for w := range work {
		if len(work[w]) > 0 {
			expected++
			go func(mine []int) {
				hits, err := PyExpectedHittingTimes(start, mine, transitions)
				results <- result{hits, err}
			}(work[w])
		}
	}
	var err error
	for i := 0; i < expected; i++ {
		r := <-results
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
	if states == nil {
		panic("states is nil")
	}
	type data struct {
		Start       int
		States      []int
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
