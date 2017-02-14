package discflo

import (
	"fmt"
	"math"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/heap"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
)

// todo
// - make it possible to compute a statistical measure on a subgraph
// - use a subgraph measure to guide a discriminative search
// - make the measure statisfy downward closure?
//         (a < b) --> (m(a) >= m(b))
// - read the leap search paper again

func Importance(lat *lattice.Lattice, n *lattice.Node) float64 {
	var f, pr_o float64
	E := float64(25) // float64(len(lat.Fail.G.E))
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	f = (float64(n.FIS()))
	pr_f := f/(F+O)
	size, support, err := n.SubGraph.SupportOf(lat.Ok)
	if err != nil {
		panic(err) // should never happen
	}
	e := float64(len(n.SubGraph.E)) + 1
	pr_o = (float64(size + 1)/(e)) * (float64(support)/(F+O))
	a := pr_f/(pr_f + pr_o)
	b := F/(F + O)
	s := ((e+1)/E) * (a - b)
	if false {
		errors.Logf("DEBUG", "pr_o %v, pr_f %v a %v b %v s %v %v", pr_o, pr_f, a, b, s, n)
	}
	return s
}

func MaxImportance(lat *lattice.Lattice, n *lattice.Node) float64 {
	var f, pr_o float64
	E := float64(25)
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	f = (float64(n.FIS()))
	pr_f := f/(F+O)
	size, support, err := n.SubGraph.SupportOf(lat.Ok)
	if err != nil {
		panic(err) // should never happen
	}
	e := E
	pr_o = (float64(size + 1)/(e)) * (float64(support)/(F+O))
	a := pr_f/(pr_f + pr_o)
	b := F/(F + O)
	s := (a - b)
	return s
}

func Gtest(lat *lattice.Lattice, n *lattice.Node) float64 {
	var f, o float64
	// E := float64(len(lat.Fail.G.E))
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	if n.SubGraph != nil {
		e := float64(len(n.SubGraph.E)) + 1
		f = float64(len(n.Embeddings)) + 1
		size, support, err := n.SubGraph.SupportOf(lat.Ok)
		if err != nil {
			panic(err) // should never happen
		}
		if support == 0 {
			o = 0
		} else {
			o = ((float64(size + 1)/(e)) * float64(support))
		}
		// o = float64(CountEmbs(n.SubGraph, lat.Ok))
		if false {
			errors.Logf("DEBUG", "f %v o %v, size %v, support %v of %v", f, o, size, support, n)
		}
	} else {
		f = F
		o = O
	}
	return 2 * F * (f * math.Log(f/o) + (1 - f) * math.Log((1-f)/(1-o)))
}

func Localize(lat *lattice.Lattice) {
	var scoreFn func(*lattice.Lattice, *lattice.Node) float64 = Importance
	// var maxFn func(*lattice.Lattice, *lattice.Node) float64 = MaxImportance
	type entry struct {
		node  *lattice.Node
		score float64
	}
	priority := func(f float64) int {
		return int(f * 1000000)
	}
	h := heap.NewMaxHeap(10)
	r := lat.Root()
	rs := float64(-1000000)
	h.Push(priority(rs), entry{r, rs})
	var best *lattice.Node
	var max float64
	for h.Size() > 0 {
		for _, x := range h.PopGroup() {
			e := x.(entry)
			n := e.node
			score := e.score
			// if n.SubGraph != nil && len(n.SubGraph.E) > 5 {
			// 	continue
			// }
			if best == nil || score > max {
				best = n
				max = score
			// } else if maxFn(lat, n) < max {
			// 	fmt.Println("skip", max, score, maxFn(lat, n), n)
			// 	continue
			}
			if true {
				fmt.Println(max, score, n)
			}
			kids, err := n.CanonKids()
			if err != nil {
				panic(err)
			}
			for _, kid := range kids {
				if kid.FIS() < 5 {
					continue
				}
				kidScore := scoreFn(lat, kid)
				if kidScore > score {
					h.Push(priority(kidScore), entry{kid, kidScore})
				}
			}
		}
	}
	fmt.Println(max, best)
}
