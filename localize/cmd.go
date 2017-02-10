package localize

import (
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/getopt"
	"github.com/timtadh/sfp/types/digraph/digraph"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	return cmd.Cmd(
	"localize",
	`[options] <failing-profiles> <succeeding-profiles>`,
	`

<failing-profiles> should be a directory (or file) containing flow-graphs from
                   failed executions of an instrumented copy of the program
                   under test (PUT).

<succeeding-profiles> should be a directory (or file) containing flow-graphs from
                      successful executions of an instrumented copy of the
                      program under test (PUT).

Option Flags
    -h,--help                         Show this message
    -o,--output=<path>                Output file to create (defaults to localized.json)
    -m,--method=<method>              Statistical method to use
                                      (defaults to: pr-fail-given-line)
    --methods                         List localization methods available
`,
	"o:w:m:",
	[]string{
		"output=",
		"method=",
		"methods",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		output := "localized.json"
		method := "pr-fail-given-line"
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-o", "--output":
				output = oa.Arg()
			case "-m", "--method":
				method = oa.Arg()
			case "--methods":
				for k, _ := range Methods {
					fmt.Println(k)
				}
				return nil, nil
			}
		}
		if len(args) != 2 {
			return nil, cmd.Usage(r, 2, "Expected exactly 2 arguments for successful/failing test profiles got: [%v]", strings.Join(args, ", "))
		}
		if _, has := Methods[method]; !has {
			return nil, cmd.Usage(r, 2, "Specified localization method '%v' is not available (see --methods).", method)
		}
		fmt.Println("method", method)
		fmt.Println("output", output)
		labels := digraph.NewLabels()
		positions := make(map[int]string)
		failFile, failClose, err := cmd.Input(args[0])
		if err != nil {
			return nil, cmd.Errorf(2, "Could not read profiles from failed executions: %v\n%v", args[0], err)
		}
		defer failClose()
		fail, err := LoadDot(positions, labels, failFile)
		if err != nil {
			return nil, cmd.Errorf(2, "Could not load profiles from failed executions: %v\n%v", args[0], err)
		}
		okFile, okClose, err := cmd.Input(args[1])
		if err != nil {
			return nil, cmd.Errorf(2, "Could not read profiles from successful executions: %v\n%v", args[0], err)
		}
		defer okClose()
		ok, err := LoadDot(positions, labels, okFile)
		if err != nil {
			return nil, cmd.Errorf(2, "Could not load profiles from successful executions: %v\n%v", args[0], err)
		}
		fmt.Println(Methods[method](fail, ok))
		return nil, nil
	})
}
