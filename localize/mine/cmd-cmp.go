package mine

import (
	"fmt"
	"time"
	"sort"
	"math"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

func NewCompareParser(c *cmd.Config, o *Options) cmd.Runnable {
	var o1, o2 Options
	var wo walkOpts
	bb := NewBranchAndBoundParser(c, &o1)
	sleap := NewSLeapParser(c, &o1)
	leap := NewLeapParser(c, &o1)
	urw := NewURWParser(c, &o2, &wo)
	swrw := NewSWRWParser(c, &o2, &wo)
	walks := NewWalksParser(c, &o2, &wo)
	topColors := NewWalkTopColorsParser(c, &o2, &wo)
	walkTypes := cmd.Commands(map[string]cmd.Runnable{
		walks.Name():     walks,
		topColors.Name(): topColors,
	})
	return cmd.Concat(
		cmd.Cmd(
		"compare",
		`[options]`,
		`
Compare a walk based method against leap, s-leap, or branch and bound.

Option Flags
    -h,--help                         Show this message
`,
		"",
		[]string{},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			o1 = *o.Copy()
			o2 = *o.Copy()
			return args, nil
		}),
		cmd.Commands(map[string]cmd.Runnable{
			bb.Name(): bb,
			sleap.Name(): sleap,
			leap.Name(): leap,
		}),
		cmd.Commands(map[string]cmd.Runnable{
			urw.Name():   cmd.Concat(urw, walkTypes),
			swrw.Name():  cmd.Concat(swrw, walkTypes),
		}),
		cmd.BareCmd(
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			min := func(a, b int) int {
				if a < b {
					return a
				}
				return b
			}
			timeit := func(m *Miner) ([]*SearchNode, time.Duration) {
				s := time.Now()
				nodes := m.Mine().unique()
				e := time.Now()
				return nodes, e.Sub(s)
			}
			sum := func(nodes []*SearchNode) float64 {
				sum := 0.0
				for _, n := range nodes {
					sum += n.Score
				}
				return sum
			}
			mean := func(nodes []*SearchNode) float64 {
				return sum(nodes)/float64(len(nodes))
			}
			stddev := func(nodes []*SearchNode) float64 {
				u := mean(nodes)
				variance := 0.0
				for _, n := range nodes {
					variance += (n.Score - u) * (n.Score - u)
				}
				if len(nodes) > 2 {
					variance = (1/(float64(len(nodes)) - 1)) * variance
				} else {
					variance = (1/float64(len(nodes))) * variance
				}
				return math.Sqrt(variance)
			}
			stderr := func(X, Y []*SearchNode) float64 {
				T := min(len(X), len(Y))
				variance := 0.0
				for i := 0; i < T; i++ {
					variance += (X[i].Score - Y[i].Score) * (X[i].Score - Y[i].Score)
				}
				if T > 2 {
					variance = (1/(float64(T) - 1)) * variance
				} else {
					variance = (1/float64(T)) * variance
				}
				return math.Sqrt(variance)
			}
			statsHeader := func() {
				fmt.Printf(
					"%-20v %15v %15v %15v %15v\n", "", "sum", "mean", "stddev", "duration")

			}
			stats := func(name string, nodes []*SearchNode, dur time.Duration) {
				fmt.Printf(
					"%-20v %15.5g %15.5g %15.5g %15v\n", name, sum(nodes), mean(nodes), stddev(nodes), dur)
			}
			output := func(name string, nodes []*SearchNode) {
				fmt.Println()
				fmt.Println(name)
				for i, n := range nodes {
					fmt.Printf("  - subgraph %-5d %v\n", i, n)
					fmt.Println()
				}
				fmt.Println()
			}
			a := NewMiner(o1.Miner, o1.Lattice, o1.Score, o1.Opts...)
			b := NewMiner(o2.Miner, o2.Lattice, o2.Score, o2.Opts...)
			A, aTime := timeit(a)
			B, bTime := timeit(b)
			A = A[:min(len(A), len(B))]
			B = B[:min(len(A), len(B))]

			output(o1.MinerName, A)
			output(o2.MinerName, B)
			statsHeader()
			stats(o1.MinerName, A, aTime)
			stats(o2.MinerName, B, bTime)
			fmt.Printf("%-20v %15.5g\n", "stderr:", stderr(A, B))
			fmt.Println()
			return args, nil
		}),
	)
}

func (nodes SearchNodes) unique() (unique []*SearchNode) {
	added := make(map[string]bool)
	for n, next := nodes(); next != nil; n, next = next() {
		if n.Node.SubGraph == nil {
			continue
		}
		label := string(n.Node.SubGraph.Label())
		if added[label] {
			continue
		}
		added[label] = true
		unique = append(unique, n)
	}
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Score > unique[j].Score
	})
	return unique
}
