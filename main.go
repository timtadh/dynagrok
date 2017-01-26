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
    -r,--go-root=<path>               go root
    -g,--go-path=<path>               go path
    -d,--dynagrok-path=<path>         dynagrok path
`,
	"p:r:g:d:",
	[]string{
		"cpu-profile=",
		"go-root=",
		"go-path=",
		"dynagrok-path=",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg, _ ...interface{}) ([]string, interface{}, *cmd.Error) {
		GOROOT := os.Getenv("GOROOT")
		GOPATH := os.Getenv("GOPATH")
		DGPATH := os.Getenv("DGPATH")
		cpuProfile := ""
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-p", "--cpu-profile":
				cpuProfile = oa.Arg()
			case "-g", "--go-path":
				GOPATH = oa.Arg()
			case "-d", "--dynagrok-path":
				DGPATH = oa.Arg()
			case "-r", "--go-root":
				GOROOT = oa.Arg()
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
			GOROOT: GOROOT,
			GOPATH: GOPATH,
			DGPATH: DGPATH,
		}
		return args, c, nil
	},
)
