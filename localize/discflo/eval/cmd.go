package eval

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
	"github.com/timtadh/dynagrok/localize/stat"
)

func NewCommand(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
		"eval",
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
			eval := func(name string, method stat.Method) {
				localized := Group(method(o.Lattice))
				for _, f := range faults {
					sum := 0
					for _, g := range localized {
						for _, l := range g {
							if l.FnName == f.FnName && l.BasicBlockId == f.BasicBlockId {
								fmt.Printf(
									"    %v {\n\trank: %v,\n\tscore: %v,\n\tfn: %v (%d),\n\tpos: %v\n    }\n",
									name,
									float64(sum)+float64(len(g))/2,
									l.Score,
									l.FnName,
									l.BasicBlockId,
									l.Position,
								)
							}
						}
						sum += len(g)
					}
				}
			}
			dflo := func(s mine.ScoreFunc) stat.Method {
				return func(lat *lattice.Lattice) stat.Result {
					miner := mine.NewMiner(o.Miner, lat, s, o.Opts...)
					c, err := discflo.Localizer(o)(miner)
					if err != nil {
						panic(err)
					}
					return c.RankColors(miner).StatResult()
				}
			}
			if o.Score == nil {
				for name, score := range mine.Scores {
					eval("Discflo + "+name, dflo(score))
					eval(name, func(s mine.ScoreFunc) stat.Method {
						return func(lat *lattice.Lattice) stat.Result {
							miner := mine.NewMiner(o.Miner, lat, s, o.Opts...)
							return mine.LocalizeNodes(miner.Score)
						}
					}(score))
				}
			} else {
				eval("Discflo + "+o.ScoreName, dflo(o.Score))
				eval(o.ScoreName, func(s mine.ScoreFunc) stat.Method {
					return func(lat *lattice.Lattice) stat.Result {
						miner := mine.NewMiner(o.Miner, lat, s, o.Opts...)
						return mine.LocalizeNodes(miner.Score)
					}
				}(o.Score))
			}
			return nil, nil
		})
}

func Group(results stat.Result) []stat.Result {
	groups := make([]stat.Result, 0, 10)
	for _, r := range results {
		lg := len(groups)
		if lg > 0 && r.Score == groups[lg-1][0].Score {
			groups[lg-1] = append(groups[lg-1], r)
		} else {
			groups = append(groups, make(stat.Result, 0, 10))
			groups[lg] = append(groups[lg], r)
		}
	}
	return groups
}
