package lattice

import ()

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
)

func (n *Node) findChildren(allow func(*subgraph.SubGraph) (bool, error)) (nodes []*Node, err error) {
	if false {
		errors.Logf("DEBUG", "findChildren %v", n)
	}
	if n.SubGraph == nil {
		for _, n := range n.dt.FrequentVertices { 
			nodes = append(nodes, n)
		}
		return nodes, nil
	}
	unsupported := n.unsupportedExts
	vords := make([][]int, 0, 10)
	builder := n.SubGraph.Builder()
	seen := make(map[string]bool)
	exts := n.extensions(unsupported)
	for ext, embs := range exts {
		// ext := p.ext
		// embs := p.embs
		if len(embs) < n.dt.Support {
			unsupported[ext] = true
			continue
		}
		b := builder.Copy()
		_, _, err := b.Extend(&ext)
		if err != nil {
			return nil, err
		}
		support := n.support(len(b.V), embs)
		if support < n.dt.Support {
			unsupported[ext] = true
			continue
		}
		vord, eord := b.CanonicalPermutation()
		extended := b.BuildFromPermutation(vord, eord)
		label := string(extended.Label())
		if seen[label] {
			continue
		}
		seen[label] = true
		if allow != nil {
			allowed, err := allow(extended)
			if err != nil {
				return nil, err
			}
			if !allowed {
				continue
			}
		}
		tembs := embs.Translate(len(extended.V), vord)
		nodes = append(nodes, NewNode(n.dt, extended, tembs))
		vords = append(vords, vord)
	}
	for i, c := range nodes {
		c.addUnsupportedExts(unsupported, len(n.SubGraph.V), vords[i])
	}
	return nodes, nil
}

func (n *Node) extensions(unsupported map[subgraph.Extension]bool) map[subgraph.Extension]subgraph.Embeddings {
	// exts := make(extensions, 0, 10)
	exts := make(map[subgraph.Extension]subgraph.Embeddings)
	add := n.validExtChecker(unsupported, func(emb *subgraph.Embedding, ext *subgraph.Extension) {
		// exts = append(exts, extension{ext, emb})
		exts[*ext] = append(exts[*ext], emb)
	})
	for _, embedding := range n.Embeddings {
		for emb := embedding; emb != nil; emb = emb.Prev {
			for _, e := range n.dt.G.Kids[emb.EmbIdx] {
				edge := &n.dt.G.E[e]
				add(embedding, edge, emb.SgIdx, -1)
			}
			for _, e := range n.dt.G.Parents[emb.EmbIdx] {
				edge := &n.dt.G.E[e]
				add(embedding, edge, -1, emb.SgIdx)
			}
		}
	}
	// return exts.partition()
	return exts
}

func (n *Node) validExtChecker(unsupported map[subgraph.Extension]bool, do func(*subgraph.Embedding, *subgraph.Extension)) func (*subgraph.Embedding, *digraph.Edge, int, int) {
	return func(emb *subgraph.Embedding, e *digraph.Edge, src, targ int) {
		if n.dt.Indices.EdgeCounts[n.dt.Indices.Colors(e)] < n.dt.Support {
			return
		}
		emb, ext := n.extension(emb, e, src, targ)
		if n.SubGraph.HasExtension(ext) {
			return
		}
		if unsupported[*ext] {
			return
		}
		do(emb, ext)
	}
}

func (n *Node) extension(embedding *subgraph.Embedding, e *digraph.Edge, src, targ int) (*subgraph.Embedding, *subgraph.Extension) {
	hasTarg := false
	hasSrc := false
	var srcIdx int = len(n.SubGraph.V)
	var targIdx int = len(n.SubGraph.V)
	if src >= 0 {
		hasSrc = true
		srcIdx = src
	}
	if targ >= 0 {
		hasTarg = true
		targIdx = targ
	}
	for emb := embedding; emb != nil; emb = emb.Prev {
		if hasTarg && hasSrc {
			break
		}
		if !hasSrc && e.Src == emb.EmbIdx {
			hasSrc = true
			srcIdx = emb.SgIdx
		}
		if !hasTarg && e.Targ == emb.EmbIdx {
			hasTarg = true
			targIdx = emb.SgIdx
		}
	}
	ext := subgraph.NewExt(
		subgraph.Vertex{Idx: srcIdx, Color: n.dt.G.V[e.Src].Color},
		subgraph.Vertex{Idx: targIdx, Color: n.dt.G.V[e.Targ].Color},
		e.Color)
	var newVE *subgraph.VertexEmbedding = nil
	if !hasSrc && !hasTarg {
		panic("both src and targ unattached")
	} else if !hasSrc {
		newVE = &subgraph.VertexEmbedding{
			SgIdx: srcIdx,
			EmbIdx: e.Src,
		}
	} else if !hasTarg {
		newVE = &subgraph.VertexEmbedding{
			SgIdx: targIdx,
			EmbIdx: e.Targ,
		}
	}
	if newVE != nil {
		embedding = embedding.Extend(*newVE)
	}
	return embedding, ext
}
