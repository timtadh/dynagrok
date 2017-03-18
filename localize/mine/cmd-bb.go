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
`,
		"k:",
		[]string{
			"top-k=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			topk := 10
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-k", "--top-k":
					k, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					topk = k
				}
			}
			o.Miner = BranchAndBound(topk).Mine
			return args, nil
		})
}
