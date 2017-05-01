package mine

import (
	"fmt"
	"math"
	"sort"
	"time"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

func algorithmParser(c *cmd.Config) func(o *Options, args []string) (*Options, []string, *cmd.Error) {
	var wo walkOpts
	return func(o *Options, args []string) (*Options, []string, *cmd.Error) {
		bb := NewBranchAndBoundParser(c, o)
		sleap := NewSLeapParser(c, o)
		leap := NewLeapParser(c, o)
		urw := NewURWParser(c, o, &wo)
		swrw := NewSWRWParser(c, o, &wo)
		walks := NewWalksParser(c, o, &wo)
		topColors := NewWalkTopColorsParser(c, o, &wo)
		walkTypes := cmd.Commands(map[string]cmd.Runnable{
			walks.Name():     walks,
			topColors.Name(): topColors,
		})
		options := map[string]cmd.Runnable{
			bb.Name():    bb,
			sleap.Name(): sleap,
			leap.Name():  leap,
			urw.Name():   cmd.Concat(urw, walkTypes),
			swrw.Name():  cmd.Concat(swrw, walkTypes),
		}
		command := cmd.Commands(options)
		if len(args) <= 0 {
			return nil, nil, nil
		}
		if _, has := options[args[0]]; !has {
			return nil, args, nil
		}
		leftover, err := command.Run(args)
		if err != nil {
			return nil, nil, err
		}
		return o, leftover, nil
	}
}

func NewCompareParser(c *cmd.Config, o *Options) cmd.Runnable {
	parser := algorithmParser(c)
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
				opts := make([]*Options, 0, 10)
				for {
					var opt *Options
					var err *cmd.Error
					opt, args, err = parser(o.Copy(), args)
					if err != nil {
						return nil, err
					}
					if opt == nil {
						break
					}
					opts = append(opts, opt)
				}
				min := func(a, b int) int {
					if a < b {
						return a
					}
					return b
				}
				timeit := func(m *Miner) ([]*SearchNode, time.Duration) {
					s := time.Now()
					nodes := m.Mine().Unique()
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
					return sum(nodes) / float64(len(nodes))
				}
				stddev := func(nodes []*SearchNode) float64 {
					u := mean(nodes)
					variance := 0.0
					for _, n := range nodes {
						variance += (n.Score - u) * (n.Score - u)
					}
					if len(nodes) > 2 {
						variance = (1 / (float64(len(nodes)) - 1)) * variance
					} else {
						variance = (1 / float64(len(nodes))) * variance
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
						variance = (1 / (float64(T) - 1)) * variance
					} else {
						variance = (1 / float64(T)) * variance
					}
					if sum(X) >= sum(Y) {
						return math.Sqrt(variance)
					} else {
						return -math.Sqrt(variance)
					}
				}
				statsHeader := func() {
					fmt.Printf(
						"%-30v %15v %15v %15v %15v %15v\n", "", "sum", "mean", "stddev", "stderr", "duration")

				}
				stats := func(name string, base, nodes []*SearchNode, dur time.Duration) {
					fmt.Printf(
						"%-30v %15.5g %15.5g %15.5g %15.5g %15v\n", name, sum(nodes), mean(nodes), stddev(nodes), stderr(base, nodes), dur)
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
				minout := -1
				outputs := make([][]*SearchNode, 0, len(opts))
				times := make([]time.Duration, 0, len(opts))
				for _, opt := range opts {
					a := NewMiner(opt.Miner, opt.Lattice, opt.Score, opt.Opts...)
					A, aTime := timeit(a)
					outputs = append(outputs, A)
					times = append(times, aTime)
					if minout < 0 || len(A) < minout {
						minout = len(A)
					}
				}
				for i := range outputs {
					output(opts[i].MinerName, outputs[i][:minout])
				}
				statsHeader()
				for i := range outputs {
					stats(opts[i].MinerName, outputs[0][:minout], outputs[i][:minout], times[i])
				}
				// a := NewMiner(o1.Miner, o1.Lattice, o1.Score, o1.Opts...)
				// b := NewMiner(o2.Miner, o2.Lattice, o2.Score, o2.Opts...)
				// A, aTime := timeit(a)
				// B, bTime := timeit(b)
				// A = A[:min(len(A), len(B))]
				// B = B[:min(len(A), len(B))]

				// output(o1.MinerName, A)
				// output(o2.MinerName, B)
				// statsHeader()
				// stats(o1.MinerName, A, aTime)
				// stats(o2.MinerName, B, bTime)
				// fmt.Printf("%-20v %15.5g\n", "stderr:", stderr(A, B))
				fmt.Println()
				return args, nil
			}),
	)
}

func (nodes SearchNodes) Unique() (unique []*SearchNode) {
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

func (nodes SearchNodes) Group() [][]*SearchNode {
	unique := nodes.Unique()
	groups := make([][]*SearchNode, 0, 10)
	for _, n := range unique {
		lg := len(groups)
		if lg > 0 && n.Score == groups[lg-1][0].Score {
			groups[lg-1] = append(groups[lg-1], n)
		} else {
			groups = append(groups, make([]*SearchNode, 0, 10))
			groups[lg] = append(groups[lg], n)
		}
	}
	return groups
}
