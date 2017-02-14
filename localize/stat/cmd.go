package stat

import (
	"os"
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/getopt"
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

type Options struct {
	FailsPath  string
	OksPath    string
	Method     Method
	MethodName string
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
    -m,--method=<method>              Statistical method to use
    --methods                         List localization methods available
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
			case "-m", "--method":
				if m, has := Methods[oa.Arg()]; has {
					fmt.Println("using method", oa.Arg())
					o.Method = m
					o.MethodName = oa.Arg()
				} else {
					return nil, cmd.Errorf(1, "Localization method '%v' is not supported. (use --methods to get a list)", oa.Arg())
				}
			case "--methods":
				fmt.Println("Statisical Localization Methods:")
				for k, _ := range Methods {
					fmt.Println("  -", k)
				}
				return nil, nil
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
		if o.Method == nil {
			return nil, cmd.Errorf(2, "Expected a localization method (flag -m)")
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
		fail, ok, err := Load(o.FailsPath, o.OksPath)
		if err != nil {
			return nil, cmd.Err(2, err)
		}
		fmt.Fprintln(ouf, o.Method(fail, ok))
		return args, nil
	})
}

func Load(failPath, okPath string) (fail, ok *Digraph, err error) {
	labels := digraph.NewLabels()
	positions := make(map[int]string)
	fnNames := make(map[int]string)
	bbids := make(map[int]int)
	failFile, failClose, err := cmd.Input(failPath)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not read profiles from failed executions: %v\n%v", failPath, err)
	}
	defer failClose()
	fail, err = LoadDot(positions, fnNames, bbids, labels, failFile)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not load profiles from failed executions: %v\n%v", failPath, err)
	}
	okFile, okClose, err := cmd.Input(okPath)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not read profiles from successful executions: %v\n%v", okPath, err)
	}
	defer okClose()
	ok, err = LoadDot(positions, fnNames, bbids, labels, okFile)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not load profiles from successful executions: %v\n%v", okPath, err)
	}
	return fail, ok, nil
}
