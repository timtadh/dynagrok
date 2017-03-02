package discflo

import (
	"bytes"
	"os"
	"io/ioutil"
	"fmt"
	"strings"
	"strconv"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/test"
)

type Options struct {
	Lattice   *lattice.Lattice
	Remote    *test.Remote
	Oracle    *test.Remote
	Tests     []*test.Testcase
	Score     Score
	ScoreName string
	Walks     int
	Minimize  bool
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
    -s,--score=<score>                Suspiciousness score to use
    --scores                          List of available suspiciousness scores
    -w,--walks=<int>                  Number of walks to perform (default: 100)
    --minimize-tests                  Do the test case minimization
    --failure-oracle=<path>           A failure oracle to filter out graphs with
                                      non-failing minimized tests.
`,
	"s:b:t:w:",
	[]string{
		"binary=",
		"test=",
		"score=",
		"scores",
		"walks=",
		"minimize-tests",
		"failure-oracle=",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		o.Walks = 100
		var testBits [][]byte
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-b", "--binary":
				r, err := test.NewRemote(oa.Arg())
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				o.Remote = r
			case "--failure-oracle":
				r, err := test.NewRemote(oa.Arg())
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				o.Oracle = r
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
			case "-s", "--score":
				name := oa.Arg()
				if n, has := scoreAbbrvs[oa.Arg()]; has {
					name = n
				}
				if m, has := Scores[name]; has {
					fmt.Println("using method", name)
					o.Score = m
					o.ScoreName = name
				} else {
					return nil, cmd.Errorf(1, "Localization method '%v' is not supported. (use --methods to get a list)", oa.Arg())
				}
			case "--scores":
				fmt.Println("\nNames of Suspicousness Scores (and Abbrevations):")
				for name, abbrvs := range scoreNames {
					fmt.Printf("  - %v : [%v]\n", name, strings.Join(abbrvs, ", "))
				}
				return nil, cmd.Errorf(0, "")
			case "-w","--walks":
				w, err := strconv.Atoi(oa.Arg())
				if err != nil {
					return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
				}
				o.Walks = w
			case "--minimize-tests":
				o.Minimize = true
			}
		}
		if len(args) < 1 {
			return nil, cmd.Usage(r, 2, "Expected an argument for successful test profiles got: [%v]", strings.Join(args, ", "))
		}
		if len(testBits) < 1 {
			return nil, cmd.Usage(r, 2, "Expected at least one test. (see -t)")
		}
		if o.Remote == nil {
			return nil, cmd.Usage(r, 2, "You must supply a binary (see -b)")
		}
		var fails bytes.Buffer
		tests := make([]*test.Testcase, 0, len(testBits))
		count := 0
		for i, bits := range testBits {
			t := test.Test(o.Remote, bits)
			err := t.Execute()
			if err != nil {
				return nil, cmd.Usage(r, 2, "Could not execute the test %d. err: %v", i, err)
			}
			if !t.Usable() {
				count++
				if count < 10 {
					continue
				}
				return nil, cmd.Usage(r, 2, "Can't use test %d", i)
			}
			_, err = fails.Write(t.Profile())
			if err != nil {
				return nil, cmd.Usage(r, 2, "Could not construct buffer for profiles. test %d err: %v", i, err)
			}
			tests = append(tests, t)
		}
		o.Tests = tests
		oksPath := args[0]
		oks, okClose, err := cmd.Input(oksPath)
		if err != nil {
			return nil, cmd.Usage(r, 2, "Could not open ok profiles, %v. err: %v", oksPath, err)
		}
		defer okClose()
		o.Lattice, err = lattice.LoadFrom(&fails, oks)
		if err != nil {
			return nil, cmd.Err(3, err)
		}
		return args[1:], nil
	})
}

func NewRunner(c *cmd.Config, o *Options) cmd.Runnable {
	return cmd.BareCmd(
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		if o.Score == nil {
			return nil, cmd.Usage(r, 2, "You must supply a score (see -s or --scores)")
		}
		result, err := o.Localize()
		if err != nil {
			return nil, cmd.Err(3, err)
		}
		fmt.Println(result)
		fmt.Println(result.StatResult())
		return nil, nil
	})
}

func (o *Options) Localize() (Result, error) {
	return o.LocalizeWithScore(o.Score)
}

func (o *Options) LocalizeWithScore(s Score) (Result, error) {
	var tests []*test.Testcase
	if o.Minimize {
		tests = o.Tests
	}
	return Localize(o.Walks, tests, o.Oracle, s, o.Lattice)
}
