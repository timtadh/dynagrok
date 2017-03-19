package mine

import (
	"math/rand"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/heap"
)

type branchBound struct {
	k int
}

func BranchAndBound(k int) TopMiner {
	return &branchBound{
		k: k,
	}
}

func (b *branchBound) Mine(m *Miner) SearchNodes {
	return b.MineFrom(m, RootNode(m.Lattice))
}

func (b *branchBound) MineFrom(m *Miner, start *SearchNode) SearchNodes {
	pop := func(stack []*SearchNode) ([]*SearchNode, *SearchNode) {
		return stack[:len(stack)-1], stack[len(stack)-1]
	}
	insert := func(sorted []*SearchNode, item *SearchNode) []*SearchNode {
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
		if len(max) < b.k {
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
	max := make([]*SearchNode, 0, b.k)
	queue := heap.NewMaxHeap(m.MaxEdges * 2)
	queue.Push(priority(start))
	seen := make(map[string]bool)
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
		filtered := filterKids(m.MinFails, m, cur.Score, kids)
		for _, kid := range filtered {
			klabel := string(kid.Node.SubGraph.Label())
			if len(max) < b.k || m.Score.Max(kid.Node) >= max[len(max)-1].Score {
				if !seen[klabel] {
					queue.Push(priority(kid))
				}
			}
		}
		if len(filtered) <= 0 && cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MinEdges {
			max = checkMax(max, cur)
		}
	}
	return SliceToNodes(max)
}
