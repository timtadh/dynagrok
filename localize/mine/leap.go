package mine

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/heap"
)

type leap struct {
	k       int
	sigma   float64
	debug   int
	maximal bool
}

func LEAP(k int, sigma float64, maximal bool, debug int) TopMiner {
	l := &leap{
		k:       k,
		sigma:   sigma,
		debug:   debug,
		maximal: maximal,
	}
	return l
}

func (l *leap) Mine(m *Miner) SearchNodes {
	return l.MineFrom(m, RootNode(m.Lattice))
}

func (l *leap) MineFrom(m *Miner, start *SearchNode) SearchNodes {
	sup := func(p float64) int {
		F := float64(m.Lattice.Fail.G.Graphs)
		s := int(p * F)
		if s < 1 {
			return 1
		} else if s < m.MinFails {
			return m.MinFails
		}
		return s
	}
	sum := func(items []*SearchNode) (sum float64) {
		for _, item := range items {
			sum += item.Score
		}
		return sum
	}
	p := 1.0
	max := newSLeap(l.k, l.sigma, sup(p), SLeapMaximal(l.maximal), SLeapDebug(l.debug-1)).mineFrom(m, start)
	// max := WalkingTopColors(
	// 	ScoreWeightedRandomWalk(),
	// 	PercentOfColors(1), WalksPerColor(10))(m).Slice()
	// if len(max) > l.k {
	// 	max = max[:l.k]
	// }
	prev := -1000.0
	cur := sum(max)
	if l.debug > 0 {
		for sup(p) > m.MinFails && abs(cur-prev) > .01 {
			if true && len(max) > 0 {
				errors.Logf("DEBUG", "cur %v %v (%v - %v) |%v - %v| = %v", sup(p), len(max), max[0].Score, max[len(max)-1].Score, prev, cur, abs(prev-cur))
			}
			p /= 2
			max = newSLeap(l.k, l.sigma, sup(p), SLeapMaximal(l.maximal), SLeapDebug(l.debug-1), startMax(max)).mineFrom(m, start)
			prev = cur
			cur = sum(max)
		}
	}
	if l.debug > 0 && len(max) > 0 {
		errors.Logf("DEBUG", "(done) cur %v %v (%v - %v) |%v - %v| = %v", sup(p), len(max), max[0].Score, max[len(max)-1].Score, prev, cur, abs(prev-cur))
	}
	max = newSLeap(l.k, 0, m.MinFails, SLeapMaximal(l.maximal), SLeapDebug(l.debug-1), startMax(max)).mineFrom(m, start)
	if l.debug > 0 && len(max) > 0 {
		errors.Logf("DEBUG", "(final) cur %v %v (%v - %v) |%v - %v| = %v", sup(p), len(max), max[0].Score, max[len(max)-1].Score, prev, cur, abs(prev-cur))
	}
	return SliceToNodes(max)
}

type sLeap struct {
	k          int
	sigma      float64
	minFailSup int
	startMax   []*SearchNode
	debug      int
	maximal    bool
}

type sLeapOpt func(*sLeap)

func startMax(max []*SearchNode) sLeapOpt {
	return func(l *sLeap) {
		l.startMax = max
	}
}

func SLeapDebug(debug int) sLeapOpt {
	return func(l *sLeap) {
		l.debug = debug
	}
}

func SLeapMaximal(maximal bool) sLeapOpt {
	return func(l *sLeap) {
		l.maximal = maximal
	}
}

func SLeap(k int, sigma float64, minFailSup int, opts ...sLeapOpt) TopMiner {
	return newSLeap(k, sigma, minFailSup, opts...)
}

func newSLeap(k int, sigma float64, minFailSup int, opts ...sLeapOpt) *sLeap {
	l := &sLeap{
		k:          k,
		sigma:      sigma,
		minFailSup: minFailSup,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *sLeap) Mine(m *Miner) SearchNodes {
	return l.MineFrom(m, RootNode(m.Lattice))
}

func (l *sLeap) MineFrom(m *Miner, start *SearchNode) SearchNodes {
	return SliceToNodes(l.mineFrom(m, start))
}

func (l *sLeap) mineFrom(m *Miner, start *SearchNode) []*SearchNode {
	minFailSup := l.minFailSup
	if l.minFailSup < 0 {
		minFailSup = m.MinFails
	}
	best := heap.NewMinHeap(l.k)
	for _, n := range l.startMax {
		if n.Node != nil && n.Node.SubGraph != nil {
			best.Push(priority(n))
		}
	}
	queue := heap.NewMaxHeap(m.MaxEdges * 2)
	queue.Push(priority(start))
	seen := make(map[string]bool)
mainLoop:
	for queue.Size() > 0 {
		var cur *SearchNode
		cur = queue.Pop().(*SearchNode)
		var label string
		if cur.Node.SubGraph != nil {
			label = string(cur.Node.SubGraph.Label())
		}
		if seen[label] {
			continue
		}
		seen[label] = true
		if l.debug >= 1 {
			errors.Logf("DEBUG", "\n\t\t\tcur %v %v %v", queue.Size(), best.Size(), cur)
		}
		if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MaxEdges {
			checkMax(best, l.k, cur)
			continue
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		var filteredKids []*SearchNode
		if l.maximal {
			filteredKids = filterKids(minFailSup, m, cur.Score, kids)
		} else {
			filteredKids = scoreKids(minFailSup, m, kids)
		}
		for _, kid := range filteredKids {
			klabel := string(kid.Node.SubGraph.Label())
			if seen[klabel] {
				_, cp := FailureProbability(m.Lattice, cur.Node)
				_, kp := FailureProbability(m.Lattice, kid.Node)
				_, cn := OkProbability(m.Lattice, cur.Node)
				_, kn := OkProbability(m.Lattice, kid.Node)
				dp := cp - kp // probability of an embedding of cur without kid in positive
				pc := (2 * dp) / (cp + kp)
				dn := cn - kn // probability of an embedding of cur without kid in negative
				nc := (2 * dn) / (cn + kn)
				if pc < l.sigma && nc < l.sigma {
					if l.debug >= 2 {
						errors.Logf("DEBUG", "\n\t\t\tleaping!!! %v %v %v", queue.Size(), best.Size(), cur)
					}
					continue mainLoop
				}
			}
		}
		if l.maximal {
			hadKid := false
			for _, kid := range filteredKids {
				klabel := string(kid.Node.SubGraph.Label())
				if best.Size() < l.k || m.Score.Max(kid.Node) > best.Peek().(*SearchNode).Score {
					hadKid = true
					if !seen[klabel] {
						queue.Push(priority(kid))
					}
				}
			}
			if !hadKid && cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MinEdges {
				checkMax(best, l.k, cur)
			}
		} else {
			for _, kid := range filteredKids {
				klabel := string(kid.Node.SubGraph.Label())
				if best.Size() < l.k || m.Score.Max(kid.Node) > best.Peek().(*SearchNode).Score {
					if !seen[klabel] {
						queue.Push(priority(kid))
					}
				}
			}
			if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MinEdges {
				checkMax(best, l.k, cur)
			}
		}
	}
	return pqToSlice(best)
}
