package main

import (
	"os"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/grok"
	"github.com/timtadh/dynagrok/instrument"
)

func main() {
	cmd.Main(cmd.Concat(
		Main,
		cmd.Commands(map[string]cmd.Runnable{
			grok.Command.Name(): grok.Command,
			instrument.Command.Name(): instrument.Command,
		}),
	))
}

var Main = cmd.Cmd(os.Args[0],
	`[options] <pkg>`,
	`
Option Flags
    -h,--help                         Show this message
    -p,--cpu-profile=<path>           Path to write the cpu-profile
    -g,--go-path=<path>               go path
    -d,--dynagrok-path=<path>         dynagrok path
    --root                            instrument a root binary (cmd/compile)
`,
	"p:g:d:",
	[]string{
		"cpu-profile=",
		"go-path=",
		"dynagrok-path=",
		"root",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg, _ ...interface{}) ([]string, interface{}, *cmd.Error) {
		GOPATH := os.Getenv("GOPATH")
		DGPATH := os.Getenv("DGPATH")
		ROOT := false
		cpuProfile := ""
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-p", "--cpu-profile":
				cpuProfile = oa.Arg()
			case "-g", "--go-path":
				GOPATH = oa.Arg()
			case "-d", "--dynagrok-path":
				DGPATH = oa.Arg()
			case "--root":
				ROOT = true
			}
		}
		if cpuProfile != "" {
			cleanup, err := cmd.CPUProfile(cpuProfile)
			if err != nil {
				return nil, nil, err
			}
			defer cleanup()
		}
		c := &cmd.Config{
			GOPATH: GOPATH,
			DGPATH: DGPATH,
			ROOT: ROOT,
		}
		return args, c, nil
	},
)
