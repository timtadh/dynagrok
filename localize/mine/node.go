package mine

import (
	"fmt"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)


type SearchNodes func() (*SearchNode, SearchNodes)

type SearchNode struct {
	Node  *lattice.Node
	Score float64
	Test  *test.Testcase
}

func (s *SearchNode) String() string {
	return fmt.Sprintf("%v %v", s.Score, s.Node)
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

func WalksToNodes(m *Miner, walk Walk, walks int) (sni SearchNodes) {
	i := 0
	sni = func() (*SearchNode, SearchNodes) {
		if i >= walks {
			return nil, nil
		}
		var n *SearchNode
		for i < walks {
			n = walk(m)
			i++
			if n.Node != nil && n.Node.SubGraph != nil {
				break
			}
		}
		if n.Node == nil || n.Node.SubGraph == nil {
			return nil, nil
		}
		return n, sni
	}
	return sni
}

func RootNode(lat *lattice.Lattice) *SearchNode {
	return &SearchNode{
		Node: lat.Root(),
		Score: -100000000000,
	}
}

func ColorNode(lat *lattice.Lattice, score *Score, color int) *SearchNode {
	vsg := subgraph.Build(1, 0).FromVertex(color).Build()
	embIdxs := lat.Fail.ColorIndex[color]
	embs := make([]*subgraph.Embedding, 0, len(embIdxs))
	for _, embIdx := range embIdxs {
		embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
	}
	colorNode := lattice.NewNode(lat, vsg, embs)
	return &SearchNode{
		Node: colorNode,
		Score: score.Score(colorNode),
	}
}

