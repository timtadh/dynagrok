package mine

import (
	"math/rand"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/heap"
)

type leap struct {
	k          int
	sigma      float64
}

func LEAP(k int, sigma float64) TopMiner {
	l := &leap{
		k:     k,
		sigma: sigma,
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
	max := newSLeap(l.k, l.sigma, sup(p)).mineFrom(m, start)
	prev := -1000.0
	cur := sum(max)
	for sup(p) >= m.MinFails && abs(cur - prev) > .01 {
		if true && len(max) > 0 {
			errors.Logf("DEBUG", "cur %v (%v - %v) |%v - %v| = %v", len(max), max[0].Score, max[len(max)-1].Score, prev, cur, abs(prev - cur))
		}
		p /= 2
		max = newSLeap(l.k, l.sigma, sup(p), startMax(max)).mineFrom(m, start)
		prev = cur
		cur = sum(max)
	}
	if true && len(max) > 0 {
		errors.Logf("DEBUG", "cur %v (%v - %v) |%v - %v| = %v", len(max), max[0].Score, max[len(max)-1].Score, prev, cur, abs(prev - cur))
	}
	max = newSLeap(l.k, 0, m.MinFails, startMax(max)).mineFrom(m, start)
	if true && len(max) > 0 {
		errors.Logf("DEBUG", "cur %v (%v - %v) |%v - %v| = %v", len(max), max[0].Score, max[len(max)-1].Score, prev, cur, abs(prev - cur))
	}
	return SliceToNodes(max)
}

type sLeap struct {
	k          int
	sigma      float64
	minFailSup int
	startMax   []*SearchNode
}

type sLeapOpt func(*sLeap)

func startMax(max []*SearchNode) sLeapOpt {
	return func(l *sLeap) {
		l.startMax = max
	}
}

func SLeap(k int, sigma float64, minFailSup int, opts ...sLeapOpt) TopMiner {
	return newSLeap(k, sigma, minFailSup, opts...)
}

func newSLeap(k int, sigma float64, minFailSup int, opts ...sLeapOpt) *sLeap {
	l := &sLeap{
		k:     k,
		sigma: sigma,
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
	pop := func(stack []*SearchNode) ([]*SearchNode, *SearchNode) {
		return stack[:len(stack)-1], stack[len(stack)-1]
	}
	insert := func(sorted []*SearchNode, item *SearchNode) []*SearchNode {
		label := string(item.Node.SubGraph.Label())
		for i := 0; i < len(sorted); i++ {
			if string(sorted[i].Node.SubGraph.Label()) == label {
				return sorted
			}
		}
		i := 0
		for ; i < len(sorted); i++ {
			a := item.Node.SubGraph
			b := sorted[i].Node.SubGraph
			if item.Score > sorted[i].Score {
				break
			} else if a.SubgraphOf(b) {
				return sorted
			}
		}
		sorted = sorted[:len(sorted)+1]
		for j := len(sorted) - 1; j > 0; j-- {
			if j == i {
				sorted[i] = item
				break
			}
			sorted[j] = sorted[j-1]
		}
		if i == 0 {
			sorted[i] = item
		}
		return sorted
	}
	priority := func(n *SearchNode) (int, *SearchNode) {
		return int(100000 * n.Score), n
	}
	checkMax := func(max []*SearchNode, cur *SearchNode) []*SearchNode {
		if cur.Node == nil || cur.Node.SubGraph == nil {
			return max
		} else if len(max) < l.k {
			return insert(max, cur)
		} else if cur.Score > max[len(max)-1].Score {
			max, _ = pop(max)
			return insert(max, cur)
		} else if cur.Score == max[len(max)-1].Score && rand.Float64() > .5 {
			max, _ = pop(max)
			return insert(max, cur)
		}
		return max
	}
	minFailSup := l.minFailSup
	if l.minFailSup < 0 {
		minFailSup = m.MinFails
	}
	max := make([]*SearchNode, 0, l.k)
	for _, n := range l.startMax {
		if n.Node != nil && n.Node.SubGraph != nil {
			max = append(max, n)
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
		if true && len(max) > 0 && cur.Node.SubGraph != nil {
			errors.Logf("DEBUG", "\n\t\t\tcur %v %v (%v - %v) %v %v", queue.Size(), len(max), max[0].Score, max[len(max)-1].Score, m.Score.Max(cur.Node), cur)
		}
		if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MaxEdges {
			max = checkMax(max, cur)
			continue
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		filteredKids := filterKids(minFailSup, m, cur.Score, kids)
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
					if false {
						errors.Logf("DEBUG", "\n\t\t\tleaping!!! %v %v (%v - %v) %v %v", queue.Size(), len(max), max[0].Score, max[len(max)-1].Score, m.Score.Max(cur.Node), cur)
					}
					continue mainLoop
				}
			}
		}
		for _, kid := range filteredKids {
			klabel := string(kid.Node.SubGraph.Label())
			if len(max) < l.k || m.Score.Max(kid.Node) >= max[len(max)-1].Score {
				if !seen[klabel] {
					queue.Push(priority(kid))
				}
			}
		}
		if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MinEdges && len(filteredKids) <= 0 {
			max = checkMax(max, cur)
		}
	}
	return max
}
