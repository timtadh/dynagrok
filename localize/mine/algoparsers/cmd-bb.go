package algoparsers

import (
	"strconv"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/opts"
	"github.com/timtadh/getopt"
)

func NewBranchAndBoundParser(c *cmd.Config, o *opts.Options) cmd.Runnable {
	return cmd.Cmd(
		"branch-and-bound",
		`[options]`,
		`
Option Flags
    -h,--help                         Show this message
    -k,--top-k=<int>                  Number of graphs to find
    --maximal                         Mine only Maximal suspicious subgraphs
    --debug                           Turn on debug prints
`,
		"k:",
		[]string{
			"top-k=",
			"debug",
			"maximal",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			debug := false
			maximal := false
			topk := 10
			for _, oa := range optargs {
				switch oa.Opt() {
				case "--debug":
					debug = true
				case "--maximal":
					maximal = true
				case "-k", "--top-k":
					k, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					topk = k
				}
			}
			o.Miner = mine.BranchAndBound(topk, maximal, debug).Mine
			o.MinerName += "branch-and-bound"
			if maximal {
				o.MinerName += " (maximal)"
			}
			return args, nil
		})
}
