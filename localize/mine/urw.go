package mine

import (
	"math/rand"
)

import (
	"github.com/timtadh/data-structures/errors"
)

type urw struct{}

func UnweightedRandomWalk() Walker {
	return &urw{}
}

func (w *urw) Walk(m *Miner) *SearchNode {
	return w.WalkFrom(m, RootNode(m.Lattice))
}

func (w *urw) WalkFromColor(m *Miner, color int) *SearchNode {
	return w.WalkFrom(m, ColorNode(m.Lattice, m.Score, color))
}

func (w *urw) WalkFrom(m *Miner, start *SearchNode) *SearchNode {
	cur := start
	prev := cur
	for cur != nil {
		if false {
			errors.Logf("DEBUG", "cur %v", cur)
		}
		if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MaxEdges {
			break
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		prev = cur
		cur = uniform(filterKids(m.MinFails, m, cur.Score, kids))
	}
	return prev
}

func uniform(slice []*SearchNode) *SearchNode {
	if len(slice) > 0 {
		return slice[rand.Intn(len(slice))]
	}
	return nil
}
