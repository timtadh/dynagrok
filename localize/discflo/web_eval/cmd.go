package web_eval

import (
	"fmt"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
)

func NewCommand(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
		"web-eval",
		`[options]`,
		`
Evaluate a fault localization method from ground truth

Option Flags
    -h,--help                         Show this message
    -f,--faults=<path>                Path to a fault file.
`,
		"f:",
		[]string{
			"faults=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			faultsPath := ""
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
				}
			}
			if faultsPath == "" {
				return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
			}
			faults, err := mine.LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			for _, f := range faults {
				fmt.Println(f)
			}
			dflo := func(s mine.ScoreFunc) func(lat *lattice.Lattice) discflo.Result {
				return func(lat *lattice.Lattice) discflo.Result {
					miner := mine.NewMiner(o.Miner, lat, s, o.Opts...)
					c, err := discflo.Localizer(o)(miner)
					if err != nil {
						panic(err)
					}
					return c.RankColors(miner)
				}
			}
			eval := func(name string, m func(*lattice.Lattice) discflo.Result) {
				r := m(o.Lattice)
				colors, P := MarkovChain(r)
				M := ExpectedHittingTimes(P)
				scores := make(map[int]float64)
				for color, states := range colors {
					min := 0
					for i, state := range states {
						hTime := M[0][state]
						if i == 0 || hTime < min {
							min = hTime
						}
					}
					scores[color] = min
				}
				for _, f := range faults {
					for color, score := range scores {
						b, fn, pos := o.Lattice.Info.Get(color)
						if fn == f.FnName && b == f.BasicBlockId {
							fmt.Printf(
								"    %v {\n\thitting time: %v,\n\tfn: %v (%d),\n\tpos: %v\n    }\n",
								name,
								score,
								fn, b, pos,
							)
							break
						}
					}
				}
			}
			if o.Score == nil {
				for name, score := range mine.Scores {
					eval("Discflo + "+name, dflo(score))
				}
			} else {
				eval("Discflo + "+o.ScoreName, dflo(o.Score))
			}
			return nil, nil
		})
}


func Group(results discflo.Result) []discflo.Result {
	groups := make([]discflo.Result, 0, 10)
	for _, r := range results {
		lg := len(groups)
		if lg > 0 && r.Score == groups[lg-1][0].Score {
			groups[lg-1] = append(groups[lg-1], r)
		} else {
			groups = append(groups, make(discflo.Result, 0, 10))
			groups[lg] = append(groups[lg], r)
		}
	}
	return groups
}

