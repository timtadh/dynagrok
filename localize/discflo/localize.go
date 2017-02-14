package discflo

import (
	"fmt"
	"math/rand"
	"sort"
)

import (
	"github.com/timtadh/data-structures/errors"
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

type Score func(lat *lattice.Lattice, n *lattice.Node) float64

func Importance(lat *lattice.Lattice, n *lattice.Node) float64 {
	var f, pr_o float64
	E := float64(len(lat.Fail.G.E))
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

func QuickImportance(lat *lattice.Lattice, n *lattice.Node) float64 {
	var f, pr_o float64
	E := float64(len(lat.Fail.G.E))
	F := float64(lat.Fail.G.Graphs)
	O := float64(lat.Ok.G.Graphs)
	f = (float64(n.FIS()))
	pr_f := f/(F+O)
	if len(n.SubGraph.E) > 0 {
		var o float64
		for i := range n.SubGraph.E {
			count := lat.Ok.EdgeCounts[n.SubGraph.Colors(i)]
			o += float64(count)/O
		}
		pr_o = o/float64(len(n.SubGraph.E))
	} else {
		pr_o = 1
	}
	e := float64(len(n.SubGraph.E)) + 1
	a := pr_f/(pr_f + pr_o)
	b := F/(F + O)
	s := ((e+1)/E) * (a - b)
	if false {
		errors.Logf("DEBUG", "pr_o %v, pr_f %v a %v b %v s %v %v", pr_o, pr_f, a, b, s, n)
	}
	return s
}


type SearchNode struct {
	Node  *lattice.Node
	Score float64
}

func (s *SearchNode) String() string {
	return fmt.Sprintf("%v %v", s.Score, s.Node)
}

func Localize(lat *lattice.Lattice) {
	var score func(*lattice.Lattice, *lattice.Node) float64 = QuickImportance
	nodes := make([]*SearchNode, 0, 100)
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		n := Walk(score, lat)
		label := string(n.Node.SubGraph.Label())
		if !seen[label] {
			nodes = append(nodes, n)
			seen[label] = true
		}
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})
	for i := 0; i < 10 && i < len(nodes); i++ {
		fmt.Println(nodes[i])
	}
}

func Walk(score Score, lat *lattice.Lattice) (*SearchNode) {
	cur := &SearchNode{
		Node: lat.Root(),
		Score: -100000000,
	}
	prev := cur
	for cur != nil {
		fmt.Println("cur", cur)
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		prev = cur
		cur = uniform(filterKids(score, cur.Score, lat, kids))
	}
	return prev
}

func filterKids(score Score, parentScore float64, lat *lattice.Lattice, kids []*lattice.Node) []*SearchNode {
	entries := make([]*SearchNode, 0, len(kids))
	for _, kid := range kids {
		if kid.FIS() < 2 {
			continue
		}
		kidScore := score(lat, kid)
		if kidScore > parentScore {
			entries = append(entries, &SearchNode{kid, kidScore})
		}
	}
	return entries
}

func uniform(slice []*SearchNode) (*SearchNode) {
	if len(slice) > 0 {
		return slice[rand.Intn(len(slice))]
	}
	return nil
}
