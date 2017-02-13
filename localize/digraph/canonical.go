package digraph

import (
	"bytes"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/digraph/subgraph"
)

func isCanonicalExtension(cur *subgraph.SubGraph, ext *subgraph.SubGraph) (bool, error) {
	// errors.Logf("DEBUG", "is %v a canonical ext of %v", ext.Label(), n)
	parent, err := firstParent(subgraph.Build(len(ext.V), len(ext.E)).From(ext))
	if err != nil {
		return false, err
	} else if parent == nil {
		return false, errors.Errorf("ext %v of node %v has no parents", ext, cur)
	}
	if bytes.Equal(parent.Build().Label(), cur.Label()) {
		return true, nil
	}
	return false, nil
}

func computeParent(b *subgraph.Builder, i int, parents []*subgraph.Builder) ([]*subgraph.Builder, error) {
	if len(b.V) == 1 && len(b.E) == 1 {
		parents = append(
			parents,
			subgraph.Build(1, 0).FromVertex(b.V[b.E[0].Src].Color),
		)
	} else if len(b.V) == 2 && len(b.E) == 1 {
		parents = append(
			parents,
			subgraph.Build(1, 0).FromVertex(b.V[b.E[0].Src].Color),
			subgraph.Build(1, 0).FromVertex(b.V[b.E[0].Targ].Color),
		)
	} else {
		nb := b.Copy()
		err := nb.RemoveEdge(i)
		if err != nil {
			return nil, err
		}
		if nb.Connected() {
			parents = append(parents, nb)
		}
	}
	return parents, nil
}

func firstParent(b *subgraph.Builder) (_ *subgraph.Builder, err error) {
	if len(b.E) <= 0 {
		return nil, nil
	}
	parents := make([]*subgraph.Builder, 0, 10)
	for i := len(b.E) - 1; i >= 0; i-- {
		parents, err = computeParent(b, i, parents)
		if err != nil {
			return nil, err
		}
		if len(parents) > 0 {
			return parents[0], nil
		}
	}
	return nil, errors.Errorf("no parents for %v", b)
}

func AllParents(b *subgraph.Builder) (parents []*subgraph.Builder, err error) {
	if len(b.E) <= 0 {
		return nil, nil
	}
	parents = make([]*subgraph.Builder, 0, 10)
	for i := len(b.E) - 1; i >= 0; i-- {
		parents, err = computeParent(b, i, parents)
		if err != nil {
			return nil, err
		}
	}
	return parents, nil
}
