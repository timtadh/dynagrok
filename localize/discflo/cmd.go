package discflo

import (
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/stat"
)


func NewCommand(c *cmd.Config) cmd.Runnable {
	return cmd.Concat(cmd.Cmd(
	"disc-flo",
	`[options]`,
	`
Use DISCriminative subgraph Fault LOcalization (disc-flo) to localize faults
from failing and passing runs.

<failing-profiles> should be a directory (or file) containing flow-graphs from
                   failed executions of an instrumented copy of the program
                   under test (PUT).

<succeeding-profiles> should be a directory (or file) containing flow-graphs
                      from successful executions of an instrumented copy of the
                      program under test (PUT).

Option Flags
    -h,--help                         Show this message
    -m,--method=<method>              Statistical method to use
    --methods                         List available methods
`,
	"",
	[]string{},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		var method stat.Method
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-m", "--method":
				if m, has := stat.Methods[oa.Arg()]; has {
					fmt.Println("using method", oa.Arg())
					method = m
				} else {
					return nil, cmd.Errorf(1, "Localization method '%v' is not supported. (use --methods to get a list)", oa.Arg())
				}
			case "--methods":
				fmt.Println("Statisical Localization Methods:")
				for k, _ := range stat.Methods {
					fmt.Println("  -", k)
				}
				return nil, nil
			}
		}
		if len(args) < 2 {
			return nil, cmd.Usage(r, 2, "Expected 2 arguments for successful/failing test profiles got: [%v]", strings.Join(args, ", "))
		}
		failsPath := args[0]
		oksPath := args[1]
		fmt.Println(failsPath, oksPath, method)
		return nil, nil
	}))
}
