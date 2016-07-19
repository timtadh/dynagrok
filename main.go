package main

import (
	"fmt"
	"os"
)

import (
	"github.com/timtadh/getopt"
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/grok"
	"github.com/timtadh/dynagrok/instrument"
)

func main() {
	main := cmd.Concat(
		Main,
		cmd.Commands(map[string]cmd.Runnable{
			grok.Command.Name(): grok.Command,
			instrument.Command.Name(): instrument.Command,
		}),
	)
	args, xtra, err := main.Run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(err.ExitCode)
	}
	if xtra == nil {
		panic(fmt.Errorf("assertion error xtra == nil"))
	}
	xtras := xtra.([]interface{})
	if len(xtras) != 2 {
		panic(fmt.Errorf("assertion error len(xtras) != 2 , ==> %v", len(xtras)))
	}
	mainOut := xtras[0].(string)
	subOut := xtras[1].(string)
	fmt.Println(mainOut, subOut)
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected 1 package name got %v\n", args)
		os.Exit(1)
	}
	pkgName := args[0]
	fmt.Printf("analyze this: %v\n", pkgName)
}

var Main = cmd.Cmd(os.Args[0],
	`[options] <pkg>`,
	`
Option Flags
    -h,--help                         Show this message
    -p,--cpu-profile=<path>           Path to write the cpu-profile
`,
	"",
	[]string{},
	func(args []string, optargs []getopt.OptArg, _ ...interface{}) ([]string, interface{}, *cmd.Error) {
		cpuProfile := ""
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-p", "--cpu-profile":
				cpuProfile = oa.Arg()
			}
		}
		if cpuProfile != "" {
			cleanup, err := cmd.CPUProfile(cpuProfile)
			if err != nil {
				return nil, nil, err
			}
			defer cleanup()
		}
		fmt.Println("Done")
		return args, "main", nil
	},
)
