package mine

import (
	"strconv"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

func NewBranchAndBoundParser(c *cmd.Config, o *Options) cmd.Runnable {
	return cmd.Cmd(
		"branch-and-bound",
		`[options]`,
		`
Option Flags
    -h,--help                         Show this message
    -k,--top-k=<int>                  Number of graphs to find
    --debug                           Turn on debug prints
`,
		"k:",
		[]string{
			"top-k=",
			"debug",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			debug := false
			topk := 10
			for _, oa := range optargs {
				switch oa.Opt() {
				case "--debug":
					debug = true
				case "-k", "--top-k":
					k, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					topk = k
				}
			}
			o.Miner = BranchAndBound(topk, debug).Mine
			o.MinerName += "branch-and-bound"
			return args, nil
		})
}
