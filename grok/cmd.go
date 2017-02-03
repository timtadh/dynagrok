package grok

import (
	"fmt"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

var Command = cmd.Cmd(
	"grok",
	`[options] <pkg>`,
	`
Option Flags
    -h,--help                         Show this message
`,
	"",
	[]string{},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		// for _, oa := range optargs {
		// 	switch oa.Opt() {
		// 	}
		// }
		fmt.Println("grokked", args, optargs)
		return args, nil
	},
)

