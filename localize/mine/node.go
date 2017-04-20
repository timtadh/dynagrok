package mine

import (
	"fmt"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
	"github.com/timtadh/dynagrok/localize/test"
)

type SearchNodes func() (*SearchNode, SearchNodes)

type SearchNode struct {
	Node  *lattice.Node
	Score float64
	Tests map[int]*test.Testcase
}

func NewSearchNode(n *lattice.Node, score float64) *SearchNode {
	return &SearchNode{
		Node:  n,
		Score: score,
		Tests: make(map[int]*test.Testcase),
	}
}

func (s *SearchNode) String() string {
	return fmt.Sprintf("%6.5v %v", s.Score, s.Node)
}

func (it SearchNodes) Slice() (nodes []*SearchNode) {
	for n, next := it(); next != nil; n, next = next() {
		nodes = append(nodes, n)
	}
	return nodes
}

func SliceToNodes(slice []*SearchNode) (sni SearchNodes) {
	i := 0
	sni = func() (*SearchNode, SearchNodes) {
		if i >= len(slice) {
			return nil, nil
		}
		n := slice[i]
		i++
		return n, sni
	}
	return sni
}

func RootNode(lat *lattice.Lattice) *SearchNode {
	return NewSearchNode(lat.Root(), -100000000000)
}

func ColorNode(lat *lattice.Lattice, score *Score, color int) *SearchNode {
	vsg := subgraph.Build(1, 0).FromVertex(color).Build()
	embIdxs := lat.Fail.ColorIndex[color]
	embs := make([]*subgraph.Embedding, 0, len(embIdxs))
	for _, embIdx := range embIdxs {
		embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
	}
	colorNode := lattice.NewNode(lat, vsg, embs)
	return NewSearchNode(colorNode, score.Score(colorNode))
}
