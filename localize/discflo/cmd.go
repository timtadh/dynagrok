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
	"github.com/timtadh/dynagrok/localize/lattice"
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
    --methods                         List localization methods available
`,
	"m:",
	[]string{
		"method=",
		"methods",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		var method Score
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-m", "--method":
				name := oa.Arg()
				if n, has := scoreAbbrvs[oa.Arg()]; has {
					name = n
				}
				if m, has := Scores[name]; has {
					fmt.Println("using method", name)
					method = m
				} else {
					return nil, cmd.Errorf(1, "Localization method '%v' is not supported. (use --methods to get a list)", oa.Arg())
				}
			case "--methods":
				fmt.Println("Graphs Scoring Method Names (and Abbrevations):")
				for name, abbrvs := range scoreNames {
					fmt.Printf("  - %v : [%v]\n", name, strings.Join(abbrvs, ", "))
				}
				return nil, nil
			}
		}
		if len(args) < 2 {
			return nil, cmd.Usage(r, 2, "Expected 2 arguments for successful/failing test profiles got: [%v]", strings.Join(args, ", "))
		}
		if method == nil {
			return nil, cmd.Usage(r, 2, "You must supply a method (see -m or --methods)")
		}
		failsPath := args[0]
		oksPath := args[1]
		lat, err := lattice.Load(failsPath, oksPath)
		if err != nil {
			return nil, cmd.Err(3, err)
		}
		Localize(method, lat)
		fmt.Println()
		return nil, nil
	}))
}
