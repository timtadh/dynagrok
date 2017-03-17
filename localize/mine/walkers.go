package mine

import (
	"github.com/timtadh/data-structures/errors"
)

type Walker interface {
	Walk(*Miner) *SearchNode
	WalkFrom(*Miner, *SearchNode) *SearchNode
	WalkFromColor(*Miner, int) *SearchNode
}


func Walking(walker Walker, walks int) MinerFunc {
	return func(m *Miner) SearchNodes {
		return WalksToNodes(m, walker.Walk, walks)
	}
}

type topColorOpts struct {
	percentOfColors float64
	walksPerColor   int
	minGroups       int
}

type TopColorOpt func(*topColorOpts)

func PercentOfColors(p float64) TopColorOpt {
	return func(o *topColorOpts) {
		o.percentOfColors = p
	}
}

func WalksPerColor(w int) TopColorOpt {
	return func(o *topColorOpts) {
		o.walksPerColor = w
	}
}

func MinGroupsWalked(m int) TopColorOpt {
	return func(o *topColorOpts) {
		o.minGroups = m
	}
}

func WalkingTopColors(walker Walker, opts ...TopColorOpt) MinerFunc {
	o := &topColorOpts{
		percentOfColors: .0625,
		walksPerColor:   2,
		minGroups:       2,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(m *Miner) (sni SearchNodes) {
		labels := len(m.Lattice.Labels.Labels())
		total := int(o.percentOfColors * float64(labels))
		if total < 10 {
			total = 10
		} else if total > 500 {
			total = 500
		}
		if total > labels {
			total = labels
		}

		added := make(map[string]bool)
		prevScore := 0.0
		groups := 0
		count := 0
		locations := LocalizeNodes(m.Score)
		i := 0
		w := 0
		sni = func() (*SearchNode, SearchNodes) {
		start:
			if w >= o.walksPerColor {
				l := locations[i]
				if prevScore - l.Score  > .0001 {
					groups++
				}
				prevScore = l.Score
				w = 0
				i++
			}
			if i >= len(locations) || i >= total && groups >= o.minGroups {
				return nil, nil
			}
			color := locations[i].Color
			var n *SearchNode
			n = walker.WalkFromColor(m, color)
			w++
			if n.Node.SubGraph == nil || len(n.Node.SubGraph.E) == 0 {
				goto start
			}
			label := string(n.Node.SubGraph.Label())
			if added[label] {
				goto start
			}
			added[label] = true
			count++
			if true {
				errors.Logf("DEBUG", "found %d/%v %d/%d %d %v", groups, o.minGroups, i, total, count, n)
			}
			return n, sni
		}
		return sni
	}
}

