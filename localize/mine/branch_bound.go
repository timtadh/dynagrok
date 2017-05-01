package mine

import (
	"math/rand"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/heap"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
)

type branchBound struct {
	k       int
	debug   bool
	maximal bool
}

func BranchAndBound(k int, debug bool) TopMiner {
	return &branchBound{
		k:       k,
		debug:   debug,
		maximal: false,
	}
}

func (b *branchBound) Mine(m *Miner) SearchNodes {
	return b.MineFrom(m, RootNode(m.Lattice))
}

func (b *branchBound) MineFrom(m *Miner, start *SearchNode) SearchNodes {
	best := heap.NewMinHeap(b.k)
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
		if b.debug {
			errors.Logf("DEBUG", "\n\t\t\tcur %v %v %v", queue.Size(), best.Size(), cur)
		}
		if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MaxEdges {
			checkMax(best, b.k, cur)
			continue
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		if b.maximal {
			hadKid := false
			scored := filterKids(m.MinFails, m, cur.Score, kids)
			for _, kid := range scored {
				klabel := string(kid.Node.SubGraph.Label())
				if best.Size() < b.k || m.Score.Max(kid.Node) >= best.Peek().(*SearchNode).Score {
					hadKid = true
					if !seen[klabel] {
						queue.Push(priority(kid))
					}
				}
			}
			if !hadKid && cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MinEdges {
				checkMax(best, b.k, cur)
			}
		} else {
			// hadKid := false
			scored := scoreKids(m.MinFails, m, kids)
			for _, kid := range scored {
				klabel := string(kid.Node.SubGraph.Label())
				if best.Size() < b.k || m.Score.Max(kid.Node) >= best.Peek().(*SearchNode).Score {
					// hadKid = true
					if !seen[klabel] {
						queue.Push(priority(kid))
					}
				}
			}
			// if !hadKid && cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MinEdges {
			if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MinEdges {
				checkMax(best, b.k, cur)
			}
		}
	}
	return SliceToNodes(pqToSlice(best))
}

func scoreKids(minFailSup int, m *Miner, kids []*lattice.Node) []*SearchNode {
	entries := make([]*SearchNode, 0, len(kids))
	for _, kid := range kids {
		if kid.FIS() < minFailSup {
			continue
		}
		kidScore := m.Score.Score(kid)
		entries = append(entries, NewSearchNode(kid, kidScore))
	}
	return entries
}

func priority(n *SearchNode) (int, *SearchNode) {
	return int(100000 * n.Score), n
}

func checkMax(best *heap.Heap, k int, cur *SearchNode) {
	if cur.Node.SubGraph == nil {
		return
	}
	if best.Size() < k {
		best.Push(priority(cur))
	} else if cur.Score > best.Peek().(*SearchNode).Score {
		best.Pop()
		best.Push(priority(cur))
	} else if cur.Score == best.Peek().(*SearchNode).Score && rand.Float64() > .5 {
		best.Pop()
		best.Push(priority(cur))
	}
}

func pqToSlice(best *heap.Heap) []*SearchNode {
	max := make([]*SearchNode, 0, best.Size())
	for best.Size() > 0 {
		max = append(max, best.Pop().(*SearchNode))
	}
	return max
}
