package lattice

import (
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)

func (n *Node) MNI() int {
	sets := make([]map[int]bool, len(n.SubGraph.V))
	for _, emb := range n.Embeddings {
		for e := emb; e != nil; e = e.Prev {
			set := sets[e.SgIdx]
			if set == nil {
				set = make(map[int]bool)
				sets[e.SgIdx] = set
			}
			set[e.EmbIdx] = true
		}
	}
	min := len(n.Embeddings)
	for _, set := range sets {
		if len(set) < min {
			min = len(set)
		}
	}
	return min
}

func (n *Node) FIS() int {
	seen := make(map[int]bool, len(n.Embeddings)*len(n.SubGraph.V))
	fis := 0
	for _, emb := range n.Embeddings {
		saw := false
		for e := emb; e != nil; e = e.Prev {
			if seen[e.EmbIdx] {
				saw = true
			}
			seen[e.EmbIdx] = true
		}
		if !saw {
			fis++
		}
	}
	return fis
}

func fis(embs subgraph.Embeddings) subgraph.Embeddings {
	out := make(subgraph.Embeddings, 0, len(embs))
	seen := make(map[int]bool)
	for _, emb := range embs {
		saw := false
		for e := emb; e != nil; e = e.Prev {
			if seen[e.EmbIdx] {
				saw = true
			}
			seen[e.EmbIdx] = true
		}
		if !saw {
			out = append(out, emb)
		}
	}
	return out
}
