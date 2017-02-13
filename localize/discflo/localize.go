package discflo

import (
	"github.com/timtadh/dynagrok/localize/digraph"
	"github.com/timtadh/dynagrok/localize/stat"
)

// todo
// - make it possible to compute a statistical measure on a subgraph
// - use a subgraph measure to guide a discriminative search
// - make the measure statisfy downward closure?
//         (a < b) --> (m(a) >= m(b))
// - read the leap search paper again


// func RelativePrecision(sg *subgraph.SubGraph, fail, ok *stat.Digraph) float64 {
// 	f := float64(CountEmbs(sg, fail))
// 	o := float64(CountEmbs(sg, ok))
// 	F := float64(fail.Graphs)
// 	O := float64(ok.Graphs)
// 	return f/(f + o) - F/(F + O)
// }

func NewLattice(fail, ok *stat.Digraph) *digraph.Digraph {
	return nil
}
