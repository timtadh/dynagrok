package main

import (
	"os"
	"fmt"
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
	fmt.Println(os.Args)
	var config cmd.Config
	var cleanup func()
	main := NewMain(&config, &cleanup)
	grk := grok.NewCommand(&config)
	inst := instrument.NewCommand(&config)
	mut := mutate.NewCommand(&config)
	loc := localize.NewCommand(&config)
	cmd.Main(cmd.Concat(
		main,
		cmd.Commands(map[string]cmd.Runnable{
			grk.Name(): grk,
			inst.Name(): inst,
			mut.Name(): mut,
			loc.Name(): loc,
		}),
	), &cleanup)
}

func NewMain(c *cmd.Config, cleanup *func()) cmd.Runnable {
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
			clean, err := cmd.CPUProfile(cpuProfile)
			if err != nil {
				return nil, err
			}
			*cleanup = clean
		}
		*c = cmd.Config{
			GOROOT: GOROOT,
			GOPATH: GOPATH,
			DGPATH: DGPATH,
		}
		return args, nil
	})
}
