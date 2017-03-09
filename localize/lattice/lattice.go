package lattice

import ()

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)

type Lattice struct {
	Fail, Ok                 *digraph.Indices
	Labels                   *digraph.Labels
	Info                     *digraph.Info
	NodeAttrs                map[int]map[string]interface{}
	frequentVertices         []*Node
}

func NewLattice(load func(l *Lattice) error) (l *Lattice, err error) {
	l = &Lattice{
		Labels: digraph.NewLabels(),
		Info: digraph.NewInfo(),
		NodeAttrs: make(map[int]map[string]interface{}),
	}
	err = load(l)
	if err != nil {
		return nil, err
	}
	errors.Logf("DEBUG", "computing starting points")
	l.frequentVertices = make([]*Node, 0, len(l.Labels.Labels()))
	for color, embIdxs := range l.Fail.ColorIndex {
		sg := subgraph.Build(1, 0).FromVertex(color).Build()
		embs := make([]*subgraph.Embedding, 0, len(embIdxs))
		for _, embIdx := range embIdxs {
			embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
		}
		if len(embs) >= 1 {
			n := NewNode(l, sg, embs)
			l.frequentVertices = append(l.frequentVertices, n)
		}
	}
	return l, nil
}

func (l *Lattice) Root() *Node {
	return NewNode(l, nil, nil)
}
