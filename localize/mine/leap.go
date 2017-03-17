package mine

import (
	"math/rand"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/heap"
)

type sLeap struct {
	k int
	sigma float64
}

func SLeap(k int, sigma float64) TopMiner {
	return &sLeap{
		k: k,
		sigma: sigma,
	}
}

func (l *sLeap) Mine(m *Miner) SearchNodes {
	return l.MineFrom(m, RootNode(m.Lattice))
}

func (l *sLeap) MineFrom(m *Miner, start *SearchNode) SearchNodes {
	pop := func(stack []*SearchNode) ([]*SearchNode, *SearchNode) {
		return stack[:len(stack)-1], stack[len(stack)-1]
	}
	insert := func(sorted []*SearchNode, item *SearchNode) []*SearchNode {
		i := 0
		for ; i < len(sorted); i++ {
			if item.Score > sorted[i].Score {
				break
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
		if len(max) < l.k {
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
	max := make([]*SearchNode, 0, l.k)
	queue := heap.NewMaxHeap(m.MaxEdges*2)
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
		if true && len(max) > 0 {
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
		filteredKids := filterKids(m, cur.Score, kids)
		for _, kid := range filteredKids {
			klabel := string(kid.Node.SubGraph.Label())
			if seen[klabel] {
				_, cp := FailureProbability(m.Lattice, cur.Node)
				_, kp := FailureProbability(m.Lattice, kid.Node)
				_, cn := OkProbability(m.Lattice, cur.Node)
				_, kn := OkProbability(m.Lattice, kid.Node)
				dp := cp - kp         // probability of an embedding of cur without kid in positive
				pc := (2*dp)/(cp + kp)
				dn := cn - kn         // probability of an embedding of cur without kid in negative
				nc := (2*dn)/(cn + kn)
				if pc < l.sigma && nc < l.sigma {
					if false {
						errors.Logf("DEBUG", "\n\t\t\tleaping!!! %v %v (%v - %v) %v %v", queue.Size(), len(max), max[0].Score, max[len(max)-1].Score, m.Score.Max(cur.Node), cur)
					}
					continue mainLoop
				}
			}
		}
		anyKids := false
		for _, kid := range filteredKids {
			klabel := string(kid.Node.SubGraph.Label())
			if len(max) < l.k || m.Score.Max(kid.Node) >= max[len(max)-1].Score {
				anyKids = true
				if !seen[klabel] {
					queue.Push(priority(kid))
				}
			}
		}
		if !anyKids && cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MinEdges {
			max = checkMax(max, cur)
		}
	}
	return SliceToNodes(max)
}

