package discflo

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/test"
)

type Location struct {
	*mine.ScoredLocation
	Clusters []*Cluster
}

func (l *Location) ShortString() string {
	return fmt.Sprintf("%v", &l.Location)
}

type Result []Location

func (r Result) ScoredLocations() mine.ScoredLocations {
	result := make(mine.ScoredLocations, 0, len(r))
	for _, l := range r {
		result = append(result, l.ScoredLocation)
	}
	return result
}

func (r Result) String() string {
	parts := make([]string, 0, len(r))
	for _, l := range r {
		parts = append(parts, l.String())
	}
	return strings.Join(parts, "\n")
}

func (r Result) Sort() {
	sort.SliceStable(r, func(i, j int) bool {
		return r[i].Score > r[j].Score
	})
}

type discflo struct {
	miner   *mine.Miner
	tests   []*test.Testcase
	oracle  test.Executor
	epsilon float64
	debug   int
}

type DiscfloOption func(*discflo)

func Tests(tests []*test.Testcase) DiscfloOption {
	return func(d *discflo) {
		d.tests = tests
	}
}

func Oracle(oracle test.Executor) DiscfloOption {
	return func(d *discflo) {
		d.oracle = oracle
	}
}

func DbScanEpsilon(epsilon float64) DiscfloOption {
	return func(d *discflo) {
		d.epsilon = epsilon
	}
}

func DebugLevel(i int) DiscfloOption {
	return func(d *discflo) {
		d.debug = i
	}
}

func Localize(m *mine.Miner, opts ...DiscfloOption) (Clusters, error) {
	d := &discflo{
		miner:   m,
		epsilon: .05,
		debug:   0,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d.localize()
}

func (d *discflo) localize() (Clusters, error) {
	added := make(map[string]bool)
	db := NewDbScan(d.epsilon)
	i := 0
	for n, next := d.miner.Mine()(); next != nil; n, next = next() {
		if n.Node.SubGraph == nil {
			continue
		}
		label := string(n.Node.SubGraph.Label())
		if added[label] {
			continue
		}
		added[label] = true
		db.Add(n)
		if d.debug >= 1 {
			errors.Logf("DEBUG", "found %d %v", i, n)
		}
		i++
	}
	return d.clusters(db)
}

func (d *discflo) clusters(db *DbScan) (Clusters, error) {
	clusters := db.Clusters()
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Score > clusters[j].Score
	})
	for _, c := range clusters {
		sort.Slice(c.Nodes, func(i, j int) bool {
			return c.Nodes[i].Score > c.Nodes[j].Score
		})
	}
	passing := make([]*mine.SearchNode, 0, 10)
	filtered := make([]*Cluster, 0, 10)
	if len(d.tests) > 0 {
		for i, c := range clusters {
			if len(filtered) >= 5 && i > 5 || len(filtered) >= 2 && i > 10 || len(filtered) >= 1 && i > 15 {
				for _, x := range clusters[i:] {
					filtered = append(filtered, x)
				}
				break
			}
			fmt.Printf("------------ cluster %d --------------\n", i)
			fmt.Println(c)
			fmt.Println("--------------------------------------")
			filterCount := 0
			for j, n := range c.Nodes {
				fmt.Println(n)
				fmt.Printf("------------ node %d -----------------\n", j)
				fmt.Println(RankNodes(d.miner, n.Node.SubGraph))
				fmt.Println("--------------------------------------")
				for count := 0; count < len(d.tests); count++ {
					j := rand.Intn(len(d.tests))
					t := d.tests[j]
					min, err := t.Minimize(d.miner.Lattice, n.Node.SubGraph)
					if err != nil {
						return nil, err
					}
					if min == nil {
						continue
					}
					n.Tests[j] = min
					fmt.Printf("------------ min test %d %d ----------\n", i, j)
					fmt.Print(min)
					if len(min.Case) <= 0 || min.Case[len(min.Case)-1] != '\n' {
						fmt.Println()
					}
					fmt.Println("--------------------------------------")
					break
				}
				if len(n.Tests) <= 0 {
					// skip this graph
					errors.Logf("INFO", "filtered (no test) %d %v", i, n)
					fmt.Println("--------------------------------------")
				} else if d.oracle == nil {
					filtered = append(filtered, c)
					break
				} else {
					var profile []byte
					var failures []byte
					var ok bool
					var t *test.Testcase
					for _, x := range n.Tests {
						t = x
						break
					}
					for len(profile) <= 0 {
						var err error
						_, _, profile, failures, ok, err = t.ExecuteWith(d.oracle)
						if err != nil {
							return nil, err
						}
					}
					if false {
						errors.Logf("INFO", "ran failure oracle %v %v %v", len(t.Case), len(failures), ok)
					}
					if len(failures) > 0 {
						filtered = append(filtered, c)
						break
					} else {
						errors.Logf("INFO", "filtered (passing test) %d %v", j, n)
						fmt.Println("--------------------------------------")
						passing = append(passing, n)
						filterCount++
						if filterCount >= 2 {
							break
						}
					}
				}
			}
		}
	} else {
		filtered = clusters
	}
	return filtered, nil
}

func (clusters Clusters) Colors() map[int][]*Cluster {
	colors := make(map[int][]*Cluster)
	for _, clstr := range clusters {
		added := make(map[int]bool)
		if false {
			errors.Logf("DEBUG", "%v", clstr)
		}
		for _, n := range clstr.Nodes {
			for j := range n.Node.SubGraph.V {
				if added[n.Node.SubGraph.V[j].Color] {
					continue
				}
				colors[n.Node.SubGraph.V[j].Color] = append(colors[n.Node.SubGraph.V[j].Color], clstr)
				added[n.Node.SubGraph.V[j].Color] = true
			}
		}
	}
	return colors
}

func (clusters Clusters) RankColors(m *mine.Miner) Result {
	return RankColors(m, clusters.Colors())
}

func ScoreColor(m *mine.Miner, color int, in []*Cluster) float64 {
	abs := func(x float64) float64 {
		if x < 0 {
			return -x
		}
		return x
	}
	epsilon := .025
	n := mine.ColorNode(m.Lattice, m.Score, color)
	var s float64
	t := 0
	for _, c := range in {
		rm := s / float64(t)
		if t < 1 || abs(c.Score-rm) < epsilon {
			s += c.Score
			t++
		} else {
			if false {
				errors.Logf("DEBUG", "skipped %v %v %v", m.Lattice.Labels.Label(color), rm, c.Score)
			}
		}
	}
	return n.Score * (s / float64(t))
}

func RankColors(m *mine.Miner, colors map[int][]*Cluster) Result {
	result := make(Result, 0, len(colors))
	for color, clusters := range colors {
		bbid, fnName, pos := m.Lattice.Info.Get(color)
		result = append(result, Location{
			&mine.ScoredLocation{
				mine.Location{
					Color:        color,
					Position:     pos,
					FnName:       fnName,
					BasicBlockId: bbid,
				},
				ScoreColor(m, color, clusters),
			},
			clusters,
		})
	}
	result.Sort()
	return result
}

func RankNodes(m *mine.Miner, sg *subgraph.SubGraph) mine.ScoredLocations {
	result := make(mine.ScoredLocations, 0, len(sg.V))
	for i := range sg.V {
		color := sg.V[i].Color
		n := mine.ColorNode(m.Lattice, m.Score, color)
		bbid, fnName, pos := m.Lattice.Info.Get(color)
		result = append(result, &mine.ScoredLocation{
			mine.Location{
				Color:        color,
				Position:     pos,
				FnName:       fnName,
				BasicBlockId: bbid,
			},
			n.Score,
		})
	}
	result.Sort()
	return result
}
