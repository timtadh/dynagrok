package subgraph

import (
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/data-structures/types"
)

import ()

type Embeddings []*Embedding

type Embedding struct {
	VertexEmbedding
	Prev *Embedding
}

type VertexEmbedding struct {
	SgIdx, EmbIdx int
}

func StartEmbedding(v VertexEmbedding) *Embedding {
	return &Embedding{VertexEmbedding: v, Prev: nil}
}

func (emb *Embedding) Extend(v VertexEmbedding) *Embedding {
	return &Embedding{VertexEmbedding: v, Prev: emb}
}

func (v *VertexEmbedding) Equals(o types.Equatable) bool {
	a := v
	switch b := o.(type) {
	case *VertexEmbedding:
		return a.EmbIdx == b.EmbIdx && a.SgIdx == b.SgIdx
	default:
		return false
	}
}

func (v *VertexEmbedding) Less(o types.Sortable) bool {
	a := v
	switch b := o.(type) {
	case *VertexEmbedding:
		return a.EmbIdx < b.EmbIdx || (a.EmbIdx == b.EmbIdx && a.SgIdx < b.SgIdx)
	default:
		return false
	}
}

func (v *VertexEmbedding) Hash() int {
	return v.EmbIdx*3 + v.SgIdx*5
}

func (v *VertexEmbedding) Translate(orgLen int, vord []int) *VertexEmbedding {
	idx := v.SgIdx
	if idx >= orgLen {
		idx = len(vord) + (idx - orgLen)
	}
	if idx < len(vord) {
		idx = vord[idx]
	}
	return &VertexEmbedding{
		SgIdx:  idx,
		EmbIdx: v.EmbIdx,
	}
}

func (emb *Embedding) Translate(orgLen int, vord []int) *Embedding {
	if emb == nil {
		return nil
	}
	return &Embedding{
		VertexEmbedding: *emb.VertexEmbedding.Translate(orgLen, vord),
		Prev:            emb.Prev.Translate(orgLen, vord),
	}
}

func (embs Embeddings) Translate(orgLen int, vord []int) Embeddings {
	translated := make(Embeddings, 0, len(embs))
	for _, emb := range embs {
		translated = append(translated, emb.Translate(orgLen, vord))
	}
	return translated
}

func (emb *Embedding) Slice(sg *SubGraph) []int {
	ids := make([]int, len(sg.V))
	for i := 0; i < len(sg.V); i++ {
		ids[i] = -1
	}
	for e := emb; e != nil; e = e.Prev {
		ids[e.SgIdx] = e.EmbIdx
	}
	return ids
}

func (emb *Embedding) list(length int) []int {
	l := make([]int, length)
	for e := emb; e != nil; e = e.Prev {
		l[e.SgIdx] = e.EmbIdx
	}
	return l
}

func (emb *Embedding) hasId(id int) bool {
	for c := emb; c != nil; c = c.Prev {
		if id == c.EmbIdx {
			return true
		}
	}
	return false
}

func (emb *Embedding) String() string {
	items := make([]string, 0, 10)
	for e := emb; e != nil; e = e.Prev {
		items = append(items, fmt.Sprintf("<sg-idx: %v, emb-idx: %v>", e.SgIdx, e.EmbIdx))
	}
	return fmt.Sprintf("(%v)", strings.Join(items, ", "))
}
