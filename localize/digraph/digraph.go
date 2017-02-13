package digraph

import ()

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/digraph/digraph"
	"github.com/timtadh/dynagrok/localize/digraph/subgraph"
)

type Digraph struct {
	Support                  int
	G                        *digraph.Digraph
	Indices                  *digraph.Indices
	Labels                   *digraph.Labels
	NodeAttrs                map[int]map[string]interface{}
	Positions                map[int]string
	FnNames                  map[int]string
	BBIds                    map[int]int
	FrequentVertices         []*Node
}

func NewDigraph(support int, load func(d *Digraph) error) (g *Digraph, err error) {
	dt := &Digraph{
		Support: support,
	}
	err = load(dt)
	if err != nil {
		return nil, err
	}
	errors.Logf("DEBUG", "computing starting points")
	for color, embIdxs := range dt.Indices.ColorIndex {
		sg := subgraph.Build(1, 0).FromVertex(color).Build()
		embs := make([]*subgraph.Embedding, 0, len(embIdxs))
		for _, embIdx := range embIdxs {
			embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
		}
		if len(embs) >= dt.Support {
			n := NewNode(dt, sg, embs)
			dt.FrequentVertices = append(dt.FrequentVertices, n)
		}
	}
	return dt, nil
}

func (g *Digraph) Root() *Node {
	return NewNode(g, nil, nil)
}
