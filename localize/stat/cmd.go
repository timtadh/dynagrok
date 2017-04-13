package stat

import (
	"fmt"
	"os"
	"strings"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/mine"
)

type Options struct {
	FailsPath  string
	OksPath    string
	Score      mine.ScoreFunc
	ScoreName  string
	OutputPath string
}

func NewCommand(c *cmd.Config) cmd.Runnable {
	var o Options
	return cmd.Concat(
		NewOptionParser(c, &o),
		NewRunner(c, &o),
	)
}

func NewOptionParser(c *cmd.Config, o *Options) cmd.Runnable {
	return cmd.Cmd(
		"stat",
		`[options] <failing-profiles> <succeeding-profiles>`,
		`
Use a statistical fault localization method to find a ordered list of suggested
locations for the root cause of a bug

<failing-profiles> should be a directory (or file) containing flow-graphs from
                   failed executions of an instrumented copy of the program
                   under test (PUT).

<succeeding-profiles> should be a directory (or file) containing flow-graphs
                      from successful executions of an instrumented copy of the
                      program under test (PUT).

Option Flags
    -h,--help                         Show this message
    -o,--output=<path>                Output file to create
                                      (defaults to standard output)
    -s,--score=<score>              Statistical method to use
    --scores                         List localization methods available
`,
		"o:w:m:",
		[]string{
			"output=",
			"method=",
			"methods",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-o", "--output":
					o.OutputPath = oa.Arg()
				case "--scores":
					fmt.Println("\nNames of Suspicousness Scores (and Abbrevations):")
					for name, abbrvs := range mine.ScoreNames {
						fmt.Printf("  - %v : [%v]\n", name, strings.Join(abbrvs, ", "))
					}
					return nil, cmd.Errorf(0, "")
				case "-s", "--score":
					name := oa.Arg()
					if n, has := mine.ScoreAbbrvs[oa.Arg()]; has {
						name = n
					}
					if m, has := mine.Scores[name]; has {
						fmt.Println("using method", name)
						o.Score = m
						o.ScoreName = name
					} else {
						return nil, cmd.Errorf(1, "Localization method '%v' is not supported. (use --methods to get a list)", oa.Arg())
					}
				}
			}
			if len(args) < 2 {
				return nil, cmd.Usage(r, 2, "Expected 2 arguments for successful/failing test profiles got: [%v]", strings.Join(args, ", "))
			}
			o.FailsPath = args[0]
			o.OksPath = args[1]
			return args[2:], nil
		})
}

func NewRunner(c *cmd.Config, o *Options) cmd.Runnable {
	return cmd.BareCmd(
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			if o.Score == nil {
				return nil, cmd.Errorf(2, "Expected a localization method (flag -s)")
			}
			ouf := os.Stdout
			if o.OutputPath != "" {
				var err error
				ouf, err = os.Create(o.OutputPath)
				if err != nil {
					return nil, cmd.Errorf(1, "Could not create output file: %v, error: %v", o.OutputPath, err)
				}
				defer ouf.Close()
			}
			l, err := lattice.Load(o.FailsPath, o.OksPath)
			if err != nil {
				return nil, cmd.Err(2, err)
			}
			miner := mine.NewMiner(nil, l, o.Score)
			fmt.Fprintln(ouf, mine.LocalizeNodes(miner.Score))
			return args, nil
		})
}
