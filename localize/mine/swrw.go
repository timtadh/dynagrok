package mine

import (
	"math/rand"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
)

type swrw struct {
	seen map[string]bool
}

func ScoreWeightedRandomWalk() Walker {
	return &swrw{
		seen: make(map[string]bool),
	}
}

func (w *swrw) Walk(m *Miner) (*SearchNode) {
	return w.WalkFrom(m, RootNode(m.Lattice))
}

func (w *swrw) WalkFromColor(m *Miner, color int) (*SearchNode) {
	return w.WalkFrom(m, ColorNode(m.Lattice, m.Score, color))
}

func (w *swrw) WalkFrom(m *Miner, start *SearchNode) (*SearchNode) {
	cur := start
	prev := cur
	for cur != nil {
		if false {
			errors.Logf("DEBUG", "cur %v", cur)
		}
		var label string
		if cur.Node.SubGraph != nil {
			label = string(cur.Node.SubGraph.Label())
		}
		if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MaxEdges {
			break
		}
		if w.seen[label] && rand.Float64() < 1/(float64(m.MaxEdges)) {
			prev = start
			cur = start
			continue
		}
		w.seen[label] = true
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		prev = cur
		cur = weighted(filterKids(m, cur.Score, kids))
	}
	return prev
}

func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}

func filterKids(m *Miner, parentScore float64, kids []*lattice.Node) ([]*SearchNode) {
	var epsilon float64 = 0
	entries := make([]*SearchNode, 0, len(kids))
	for _, kid := range kids {
		if kid.FIS() < m.MinFails {
			continue
		}
		kidScore := m.Score.Score(kid)
		_, prf := FailureProbability(m.Lattice, kid)
		_, pro := OkProbability(m.Lattice, kid)
		// errors.Logf("DEBUG", "kid %v %v", kidScore, kid)
		if (abs(parentScore - kidScore) <= epsilon && abs(1 - prf/(pro + prf)) <= epsilon) || kidScore > parentScore {
			entries = append(entries, &SearchNode{kid, kidScore, nil})
		}
	}
	return entries
}

func weighted(slice []*SearchNode) (*SearchNode) {
	if len(slice) <= 0 {
		return nil
	}
	if len(slice) == 1 {
		return slice[0]
	}
	i := weightedSample(weights(slice))
	return slice[i]
}

func weights(slice []*SearchNode) []float64 {
	weights := make([]float64, 0, len(slice))
	min := 0.0
	for i, v := range slice {
		w := v.Score
		weights = append(weights, w)
		if i <= 0 || w < min {
			min = w
		}
	}
	if min < 0 {
		for i, w := range weights {
			weights[i] = w - min
		}
	}
	for i, w := range weights {
		if weights[i] < 0 {
			panic("weight < 0")
		}
		weights[i] = w + .001
	}
	return weights
}

func weightedSample(weights []float64) int {
	var total float64
	for _, w := range weights {
		total += w
	}
	i := 0
	r := total * rand.Float64()
	for ; i < len(weights) - 1 && r > weights[i]; i++ {
		r -= weights[i]
	}
	return i
}
