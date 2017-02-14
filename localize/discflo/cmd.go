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
`,
	"",
	[]string{},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		for _, oa := range optargs {
			switch oa.Opt() {
			}
		}
		if len(args) < 2 {
			return nil, cmd.Usage(r, 2, "Expected 2 arguments for successful/failing test profiles got: [%v]", strings.Join(args, ", "))
		}
		failsPath := args[0]
		oksPath := args[1]
		lat, err := lattice.Load(failsPath, oksPath)
		if err != nil {
			return nil, cmd.Err(3, err)
		}
		Localize(lat)
		fmt.Println()
		return nil, nil
	}))
}
