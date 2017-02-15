package discflo

import (
	"bytes"
	"os"
	"io/ioutil"
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/test"
)


func NewCommand(c *cmd.Config) cmd.Runnable {
	return cmd.Concat(cmd.Cmd(
	"disc-flo",
	`[options]`,
	`
Use DISCriminative subgraph Fault LOcalization (disc-flo) to localize faults
from failing and passing runs.

<succeeding-profiles> should be a directory (or file) containing flow-graphs
                      from successful executions of an instrumented copy of the
                      program under test (PUT).

Option Flags
    -h,--help                         Show this message
    -b,--binary=<path>                The binary to test. It should be
                                      instrumented.
                                      (see: dynagrok instrument -h)
    -t,--test=<path>                  Failing test case to minimize. (May be
                                      specified multiple times or with a comma
                                      separated list).
    -m,--method=<method>              Statistical method to use
    --methods                         List localization methods available
`,
	"m:b:t:",
	[]string{
		"binary=",
		"test=",
		"method=",
		"methods",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		var remote *test.Remote
        var testBits [][]byte
		var method Score
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-b", "--binary":
				r, err := test.NewRemote(oa.Arg())
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				remote = r
            case "-t", "--test":
				for _, path := range strings.Split(oa.Arg(), ",") {
					fmt.Println("test", path)
					if f, err := os.Open(path); err != nil {
						return nil, cmd.Errorf(1, "Could not open test %v, err: %v", path, err)
					} else {
						bits, err := ioutil.ReadAll(f)
						f.Close()
						if err != nil {
							return nil, cmd.Errorf(1, "Could not read test %v, err: %v", path, err)
						}
						testBits = append(testBits, bits)
					}
				}
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
		if len(args) < 1 {
			return nil, cmd.Usage(r, 2, "Expected an argument for successful test profiles got: [%v]", strings.Join(args, ", "))
		}
		if method == nil {
			return nil, cmd.Usage(r, 2, "You must supply a method (see -m or --methods)")
		}
		if remote == nil {
			return nil, cmd.Usage(r, 2, "You must supply a binary (see -b)")
		}
		var fails bytes.Buffer
		tests := make([]*test.Testcase, 0, len(testBits))
		for i, bits := range testBits {
			t := test.Test(remote, bits)
			err := t.Execute()
			if err != nil {
				return nil, cmd.Usage(r, 2, "Could not execute the test %d. err: %v", i, err)
			}
			if !t.Usable() {
				return nil, cmd.Usage(r, 2, "Can't use test %d", i)
			}
			_, err = fails.Write(t.Profile())
			if err != nil {
				return nil, cmd.Usage(r, 2, "Could not construct buffer for profiles. test %d err: %v", i, err)
			}
			tests = append(tests, t)
		}
		oksPath := args[0]
		oks, okClose, err := cmd.Input(oksPath)
		if err != nil {
			return nil, cmd.Usage(r, 2, "Could not open ok profiles, %v. err: %v", oksPath, err)
		}
		defer okClose()
		lat, err := lattice.LoadFrom(&fails, oks)
		if err != nil {
			return nil, cmd.Err(3, err)
		}
		err = Localize(tests, method, lat)
		if err != nil {
			return nil, cmd.Err(3, err)
		}
		fmt.Println()
		return nil, nil
	}))
}
