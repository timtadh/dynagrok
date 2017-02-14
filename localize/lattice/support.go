package lattice

import (
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)

func (n *Node) support(V int, embs []*subgraph.Embedding) int {
	return n.mni(V, embs)
}

func (n *Node) mni(V int, embs []*subgraph.Embedding) int {
	sets := make([]map[int]bool, V)
	for _, emb := range embs {
		for e := emb; e != nil; e = e.Prev {
			set := sets[e.SgIdx]
			if set == nil {
				set = make(map[int]bool)
				sets[e.SgIdx] = set
			}
			set[e.EmbIdx] = true
		}
	}
	min := len(embs)
	for _, set := range sets {
		if len(set) < min {
			min = len(set)
		}
	}
	return min
}

func (n *Node) fis(V int, embs []*subgraph.Embedding) int {
	seen := make(map[int]bool, len(embs)*V)
	fis := 0
	for _, emb := range embs {
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
