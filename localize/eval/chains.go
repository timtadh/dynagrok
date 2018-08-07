package eval

import (
	"fmt"
	"strings"

	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
	"github.com/timtadh/dynagrok/localize/mine"
)

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
	for A, aJumps := range jumps {
		a := blockStates[A][0]
		P[start][a] = 1 / float64(len(blockStates))
		if len(aJumps) <= 0 {
			P[a][start] = 1
		} else {
			P[a][start] = returnPr
		}
		for B := range aJumps {
			b := blockStates[B][0]
			P[a][b] = (1 / float64(len(aJumps))) * (1 - returnPr)
		}
	}
	return blockStates, P
}

func DsgMarkovChain(max int, m *mine.Miner, nodes []*mine.SearchNode, jumpPr float64, jumps map[int]map[int]bool) (blockStates map[int][]int, P [][]float64) {
	labels := m.Lattice.Labels.Labels()
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
	for color := range labels {
		if states >= max {
			fmt.Println("warning hit max states", states, max)
			break
		}
		if _, has := blockStates[color]; !has {
			blockStates[color] = append(blockStates[color], states)
			states++
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
