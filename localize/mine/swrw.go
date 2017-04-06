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
	sampleNonMax bool
}

type SWRWOpt func(*swrw)

func SWRWSampleNonMax(s *swrw) {
	s.sampleNonMax = true
}

func ScoreWeightedRandomWalk(opts ...SWRWOpt) Walker {
	s := &swrw{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (w *swrw) Walk(m *Miner) *SearchNode {
	return w.WalkFrom(m, RootNode(m.Lattice))
}

func (w *swrw) WalkFromColor(m *Miner, color int) *SearchNode {
	return w.WalkFrom(m, ColorNode(m.Lattice, m.Score, color))
}

func (w *swrw) WalkFrom(m *Miner, start *SearchNode) *SearchNode {
	cur := start
	prev := cur
	for cur != nil {
		if false {
			errors.Logf("DEBUG", "cur %v", cur)
		}
		if cur.Node.SubGraph != nil && len(cur.Node.SubGraph.E) >= m.MaxEdges {
			break
		}
		if rand.Float64() < 1/(float64(m.MaxEdges)) {
			prev = start
			cur = start
			continue
		}
		if w.sampleNonMax && rand.Float64() < 1/float64(m.MaxEdges) {
			break
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		prev = cur
		cur = weighted(filterKids(m.MinFails, m, cur.Score, kids))
	}
	return prev
}

func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}

func filterKids(minFailSup int, m *Miner, parentScore float64, kids []*lattice.Node) []*SearchNode {
	var epsilon float64 = 1e-17
	entries := make([]*SearchNode, 0, len(kids))
	for _, kid := range kids {
		if kid.FIS() < minFailSup {
			continue
		}
		kidScore := m.Score.Score(kid)
		_, prf := FailureProbability(m.Lattice, kid)
		_, pro := OkProbability(m.Lattice, kid)
		// errors.Logf("DEBUG", "kid %v %v", kidScore, kid)
		if (kidScore == parentScore && abs(1-prf/(pro+prf)) <= epsilon) || kidScore > parentScore {
			entries = append(entries, NewSearchNode(kid, kidScore))
		}
	}
	return entries
}

func weighted(slice []*SearchNode) *SearchNode {
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
		weights[i] = w + 1e-8
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
	for ; i < len(weights)-1 && r > weights[i]; i++ {
		r -= weights[i]
	}
	return i
}
