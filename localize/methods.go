package localize

import (
	"github.com/timtadh/sfp/types/digraph/digraph"
)

type Method func(labels *digraph.Labels, fail *digraph.Digraph, failAttrs VertexAttrs, ok *digraph.Digraph, okAttrs VertexAttrs)

var Methods = map[string]Method{
	"pr-fail-given-line": prFailGivenLine,
}

func prFailGivenLine(labels *digraph.Labels, fail *digraph.Digraph, failAttrs VertexAttrs, ok *digraph.Digraph, okAttrs VertexAttrs) {
}

