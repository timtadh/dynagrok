package eval

import (
	"fmt"
	"math"

	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/fault"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/opts"
)

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
			// fmt.Println(n)
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
				// if fnName == f.FnName && bbid == f.BasicBlockId {
				fmt.Println(pos)
				if pos == f.Position {
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

func SBBFLRankListEval(m *mine.Miner, faults []*fault.Fault, nodes []*mine.SearchNode, methodName, scoreName string) EvalResults {
	min := -1.0
	minScore := -1.0
	gid := 0
	var fault *fault.Fault
	groups := mine.GroupNodesByScore(nodes)
	for _, f := range faults {
		sum := 0.0
		for i, g := range groups {
			count := 0
			for _, n := range g {
				for _, v := range n.Node.SubGraph.V {
					b, fn, _ := m.Lattice.Info.Get(v.Color)
					if fn == f.FnName && b == f.BasicBlockId {
						count++
						break
					}
				}
			}
			if count > 0 {
				r := float64(len(g) - count)
				b := float64(count)
				score := ((b + r + 1) / (b + 1)) + sum
				if min <= 0 || score < min {
					gid = i
					fault = f
					min = score
					minScore = g[0].Score
				}
			}
			sum += float64(len(g))
		}
	}
	if min <= 0 {
		min = math.Inf(1)
	}
	r := &RankListEvalResult{
		MethodName:     methodName,
		ScoreName:      scoreName,
		RankScore:      min,
		Suspiciousness: minScore,
		LocalizedFault: fault,
	}
	fmt.Printf(
		"   %v + %v {\n        rank: %v, gid: %v group-size: %v\n        score: %v\n    }\n",
		methodName, scoreName,
		r.Rank(), gid, len(groups[gid]),
		r.RawScore(),
	)
	return EvalResults{r}
}
