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


type SearchNode struct {
	Node  *lattice.Node
	Score float64
}

func (s *SearchNode) String() string {
	return fmt.Sprintf("%v %v", s.Score, s.Node)
}

func Localize(score Score, lat *lattice.Lattice) {
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
	i := 0
	prev := cur
	for cur != nil {
		if true {
			errors.Logf("DEBUG", "cur %v", cur)
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		prev = cur
		cur = uniform(filterKids(score, cur.Score, lat, kids))
		if i == 1 {
		}
		i++
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
