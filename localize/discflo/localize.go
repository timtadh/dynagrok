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
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/dynagrok/localize/stat"
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

func Localize(tests []*test.Testcase, score Score, lat *lattice.Lattice) error {
	WALKS := 10
	nodes := make([]*SearchNode, 0, WALKS)
	seen := make(map[string]bool, WALKS)
	for i := 0; i < WALKS; i++ {
		n := Walk(score, lat)
		if n.Node.SubGraph == nil || len(n.Node.SubGraph.E) < 2 {
			continue
		}
		label := string(n.Node.SubGraph.Label())
		if !seen[label] {
			nodes = append(nodes, n)
			seen[label] = true
		}
	}
	if len(nodes) == 0 {
		fmt.Println("no graphs")
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})
	for i := 0; i < 10 && i < len(nodes); i++ {
		fmt.Println(nodes[i])
		fmt.Printf("------------ ranks %d ----------------\n", i)
		fmt.Println(RankNodes(score, lat, nodes[i].Node.SubGraph))
		fmt.Println("--------------------------------------")
		count := 0
		for {
			if count >= len(tests) {
				break
			}
			count++
			j := rand.Intn(len(tests))
			t := tests[j]
			min, err := t.Minimize(lat, nodes[i].Node.SubGraph)
			if err != nil {
				return err
			}
			if min == nil {
				continue
			}
			fmt.Printf("------------ min test %d %d ----------\n", i, j)
			fmt.Println(min)
			fmt.Println("--------------------------------------")
			break
		}
	}
	return nil
}

func RankNodes(score Score, lat *lattice.Lattice, sg *subgraph.SubGraph) stat.Result {
	result := make(stat.Result, 0, len(sg.V))
	for i := range sg.V {
		color := sg.V[i].Color
		vsg := subgraph.Build(1, 0).FromVertex(color).Build()
		embIdxs := lat.Fail.ColorIndex[color]
		embs := make([]*subgraph.Embedding, 0, len(embIdxs))
		for _, embIdx := range embIdxs {
			embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
		}
		n := lattice.NewNode(lat, vsg, embs)
		s := score(lat, n)
		result = append(result, stat.Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			s,
		})
	}
	result.Sort()
	return result
}

func Walk(score Score, lat *lattice.Lattice) (*SearchNode) {
	cur := &SearchNode{
		Node: lat.Root(),
		Score: -100000000,
	}
	i := 0
	prev := cur
	for cur != nil {
		if false {
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
