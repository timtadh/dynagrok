package eval

import (
	"fmt"
	"strconv"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/eval"
	"github.com/timtadh/dynagrok/localize/mine"
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
    -m,--max=<int>                    Maximum number of states in the chain
    -j,--jump-prs=<float64>           Probability of taking jumps in chains which have them
`,
		"f:m:j:",
		[]string{
			"faults=",
			"max=",
			"jump-prs=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			max := 1000000
			faultsPath := ""
			jumpPrs := []float64{}
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
				case "-m", "--max":
					var err error
					max, err = strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "For flag %v expected an int got %v. err: %v", oa.Opt, oa.Arg(), err)
					}
				case "-j", "--jump-pr":
					jumpPr, err := strconv.ParseFloat(oa.Arg(), 64)
					if err != nil {
						return nil, cmd.Errorf(1, "For flag %v expected a float got %v. err: %v", oa.Opt, oa.Arg(), err)
					}
					if jumpPr < 0 || jumpPr >= 1 {
						return nil, cmd.Errorf(1, "For flag %v expected a float between 0-1. got %v", oa.Opt, oa.Arg())
					}
					jumpPrs = append(jumpPrs, jumpPr)
				}
			}
			if len(jumpPrs) <= 0 {
				jumpPrs = append(jumpPrs, (1. / 10.))
			}
			if faultsPath == "" {
				return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
			}
			faults, err := mine.LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			fmt.Println("max", max)
			for _, f := range faults {
				fmt.Println(f)
			}
			for _, jumpPr := range jumpPrs {
				results, err := eval.Evaluate(faults, o, o.Score, "RankList", "DISCFLO", o.ScoreName, "", max, jumpPr)
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				fmt.Println(results)
			}
			// if o.Score == nil {
			// 	for name, score := range mine.Scores {
			// 		eval.Eval(faults, o.Lattice, "Discflo + "+name, eval.Discflo(o, o.Lattice, score))
			// 		eval.Eval(faults, o.Lattice, name, eval.CBSFL(o, o.Lattice, score))
			// 		colors, P, err := DiscfloMarkovChain(jumpPr, max, o, score)
			// 		if err != nil {
			// 			return nil, cmd.Err(1, err)
			// 		}
			// 		mine.MarkovEval(faults, o.Lattice, "discflo + "+name, colors, P)
			// 		m := mine.NewMiner(o.Miner, o.Lattice, score, o.Opts...)
			// 		colors, P = mine.DsgMarkovChain(max, m)
			// 		mine.MarkovEval(faults, o.Lattice, "mine-dsg + "+name, colors, P)
			// 		colors, P = mine.RankListMarkovChain(max, m)
			// 		mine.MarkovEval(faults, o.Lattice, name, colors, P)
			// 		colors, P = mine.SpacialJumps(jumpPr, max, m)
			// 		mine.MarkovEval(faults, o.Lattice, "spacial jumps + "+name, colors, P)
			// 		colors, P = mine.BehavioralJumps(jumpPr, max, m)
			// 		mine.MarkovEval(faults, o.Lattice, "behavioral jumps + "+name, colors, P)
			// 		colors, P = mine.BehavioralAndSpacialJumps(jumpPr, max, m)
			// 		mine.MarkovEval(faults, o.Lattice, "behavioral and spacial jumps + "+name, colors, P)
			// 	}
			// } else {
			// 	eval.Eval(faults, o.Lattice, "Discflo + "+o.ScoreName, eval.Discflo(o, o.Lattice, o.Score))
			// 	eval.Eval(faults, o.Lattice, o.ScoreName, eval.CBSFL(o, o.Lattice, o.Score))
			// 	colors, P, err := DiscfloMarkovChain(jumpPr, max, o, o.Score)
			// 	if err != nil {
			// 		return nil, cmd.Err(1, err)
			// 	}
			// 	mine.MarkovEval(faults, o.Lattice, "discflo + "+o.ScoreName, colors, P)
			// 	m := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
			// 	colors, P = mine.DsgMarkovChain(max, m)
			// 	mine.MarkovEval(faults, o.Lattice, "mine-dsg + "+o.ScoreName, colors, P)
			// 	colors, P = mine.RankListMarkovChain(max, m)
			// 	mine.MarkovEval(faults, o.Lattice, o.ScoreName, colors, P)
			// 	colors, P = mine.SpacialJumps(jumpPr, max, m)
			// 	mine.MarkovEval(faults, o.Lattice, "spacial jumps + "+o.ScoreName, colors, P)
			// 	colors, P = mine.BehavioralJumps(jumpPr, max, m)
			// 	mine.MarkovEval(faults, o.Lattice, "behavioral jumps + "+o.ScoreName, colors, P)
			// 	colors, P = mine.BehavioralAndSpacialJumps(jumpPr, max, m)
			// 	mine.MarkovEval(faults, o.Lattice, "behavioral and spacial jumps + "+o.ScoreName, colors, P)
			// }
			return nil, nil
		})
}
