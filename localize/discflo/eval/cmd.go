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
)

type ColorScore struct {
	Color int
	Score float64
}

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
			if o.Score == nil {
				for name, score := range mine.Scores {
					Eval(faults, o.Lattice, "Discflo + "+name, Discflo(o, o.Lattice, score))
					Eval(faults, o.Lattice, name, CBSFL(o, o.Lattice, score))
				}
			} else {
				Eval(faults, o.Lattice, "Discflo + "+o.ScoreName, Discflo(o, o.Lattice, o.Score))
				Eval(faults, o.Lattice, o.ScoreName, CBSFL(o, o.Lattice, o.Score))
			}
			return nil, nil
		})
}

func Discflo(o *discflo.Options, lat *lattice.Lattice, s mine.ScoreFunc) [][]ColorScore {
	miner := mine.NewMiner(o.Miner, lat, s, o.Opts...)
	c, err := discflo.Localizer(o)(miner)
	if err != nil {
		panic(err)
	}
	groups := make([][]ColorScore, 0, 10)
	for _, group := range c.RankColors(miner).ScoredLocations().Group() {
		colorGroup := make([]ColorScore, 0, len(group))
		for _, n := range group {
			colorGroup = append(colorGroup, ColorScore{n.Color, n.Score})
		}
		groups = append(groups, colorGroup)
	}
	return groups
}

func CBSFL(o *discflo.Options, lat *lattice.Lattice, s mine.ScoreFunc) [][]ColorScore {
	miner := mine.NewMiner(o.Miner, lat, s, o.Opts...)
	groups := make([][]ColorScore, 0, 10)
	for _, group := range mine.LocalizeNodes(miner.Score).Group() {
		colorGroup := make([]ColorScore, 0, len(group))
		for _, n := range group {
			colorGroup = append(colorGroup, ColorScore{n.Color, n.Score})
		}
		groups = append(groups, colorGroup)
	}
	return groups
}

func Eval(faults []*mine.Fault, lat *lattice.Lattice, name string, groups [][]ColorScore) (results mine.EvalResults) {
	for _, f := range faults {
		sum := 0
		for gid, group := range groups {
			for _, cs := range group {
				bbid, fnName, pos := lat.Info.Get(cs.Color)
				if fnName == f.FnName && bbid == f.BasicBlockId {
					fmt.Printf(
						"    %v {\n        rank: %v, gid: %v, group-size: %v\n        score: %v,\n        fn: %v (%d),\n        pos: %v\n    }\n",
						name,
						float64(sum)+float64(len(group))/2, gid, len(group),
						cs.Score,
						fnName,
						bbid,
						pos,
					)
					r := &mine.RankListEvalResult{
						MethodName:     "CBSFL",
						ScoreName:      name,
						RankScore:      float64(sum) + float64(len(group))/2,
						Suspiciousness: cs.Score,
						LocalizedFault: f,
						Loc: &mine.Location{
							Color:        cs.Color,
							BasicBlockId: bbid,
							FnName:       fnName,
							Position:     pos,
						},
					}
					results = append(results, r)
				}
			}
			sum += len(group)
		}
	}
	return results
}
