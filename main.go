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
	"github.com/timtadh/dynagrok/mutate"
	"github.com/timtadh/dynagrok/localize"
)

func main() {
	var config cmd.Config
	main := NewMain(&config)
	inst := instrument.NewCommand(&config)
	mut := mutate.NewCommand(&config)
	loc := localize.NewCommand(&config)
	cmd.Main(cmd.Concat(
		main,
		cmd.Commands(map[string]cmd.Runnable{
			grok.Command.Name(): grok.Command,
			inst.Name(): inst,
			mut.Name(): mut,
			loc.Name(): loc,
		}),
	))
}

func NewMain(c *cmd.Config) cmd.Runnable {
	return cmd.Cmd(os.Args[0],
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
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
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
				return nil, err
			}
			defer cleanup()
		}
		*c = cmd.Config{
			GOROOT: GOROOT,
			GOPATH: GOPATH,
			DGPATH: DGPATH,
		}
		return args, nil
	})
}
