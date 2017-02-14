package eval

import (
	"fmt"
	"bufio"
	"bytes"
	"encoding/json"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/stat"
)

type Options struct {
	stat.Options
	FailuresPath string
}

func NewCommand(c *cmd.Config) cmd.Runnable {
	var o Options
	optsParser := stat.NewOptionParser(c, &o.Options)
	return cmd.Concat(cmd.Cmd(
	"eval",
	`[options]`,
	`
Evaluate a fault localization method from ground truth

Option Flags
    -h,--help                         Show this message
    -f,--failures=<path>              Path to a failures file.
`,
	optsParser.ShortOpts() + "f:",
	append(optsParser.LongOpts(),
		"failures=",
	),
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		failures := ""
		consumed := make(map[int]bool)
		for i, oa := range optargs {
			switch oa.Opt() {
			case "-f", "--failures":
				failures = oa.Arg()
				consumed[i] = true
			}
		}
		if failures == "" {
			return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the failures")
		}
		o.FailuresPath = failures
		outargs := make([]string, 0, len(optargs) + len(args))
		for i, oa := range optargs {
			if !consumed[i] {
				outargs = append(outargs, oa.Opt())
				if oa.Arg() != "" {
					outargs = append(outargs, oa.Arg())
				}
			}
		}
		outargs = append(outargs, args...)
		return outargs, nil
	}),
	optsParser,
	cmd.BareCmd(
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		failures, err := LoadFailures(o.FailuresPath)
		if err != nil {
			return nil, cmd.Err(1, err)
		}
		lat, err := stat.Load(o.FailsPath, o.OksPath)
		if err != nil {
			return nil, cmd.Err(1, err)
		}
		eval := func(f *Failure, name string, m stat.Method) {
			localized := Group(m(lat))
			sum := 0
			for _, g := range localized {
				for _, l := range g {
					if l.FnName == f.FnName && l.BasicBlockId == f.BasicBlockId {
						fmt.Printf(
							"    %v {\n\trank: %v,\n\tscore: %v,\n\tfn: %v (%d),\n\tpos: %v\n    }\n",
							name,
							float64(sum) + float64(len(g))/2,
							l.Score,
							l.FnName,
							l.BasicBlockId,
							l.Position,
						)
					}
				}
				sum += len(g)
			}
		}
		for _, f := range failures {
			fmt.Println(f)
			if o.Method == nil {
				for name, method := range stat.Methods {
					eval(f, name, method)
				}
			} else {
				eval(f, o.MethodName, o.Method)
			}
		}
		return nil, nil
	}))
}

func Group(results stat.Result) []stat.Result {
	groups := make([]stat.Result, 0, 10)
	for _, r := range results {
		lg := len(groups)
		if lg > 0 && r.Score == groups[lg-1][0].Score {
			groups[lg-1] = append(groups[lg-1], r)
		} else {
			groups = append(groups, make(stat.Result, 0, 10))
			groups[lg] = append(groups[lg], r)
		}
	}
	return groups
}

type Failure struct {
	Position string
	FnName   string
	BasicBlockId int
}

func (f *Failure) String() string {
	return fmt.Sprintf(`Failure {
    Position: %v,
    FnName: %v,
    BasicBlockId: %d,
}`, f.Position, f.FnName, f.BasicBlockId)
}

func LoadFailure(bits []byte) (*Failure, error) {
	var f Failure
	err := json.Unmarshal(bits, &f)
	if err != nil{
		return nil, err
	}
	return &f, nil
}

func LoadFailures(path string) ([]*Failure, error) {
	fin, failClose, err := cmd.Input(path)
	if err != nil {
		return nil, fmt.Errorf("Could not read the list of failures: %v\n%v", path, err)
	}
	defer failClose()
	seen := make(map[string]bool)
	failures := make([]*Failure, 0, 10)
	s := bufio.NewScanner(fin)
	for s.Scan() {
		line := bytes.TrimSpace(s.Bytes())
		if len(line) == 0 {
			continue
		}
		f, err := LoadFailure(line)
		if err != nil {
			return nil, fmt.Errorf("Could not load failure: `%v`\nerror: %v", string(line), err)
		}
		if !seen[f.Position] {
			seen[f.Position] = true
			failures = append(failures, f)
		}
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("Could not read the failures file: %v, error: %v", path, err)
	}
	return failures, nil
}
