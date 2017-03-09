package localize

import ()

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/stat"
	"github.com/timtadh/dynagrok/localize/eval"
	"github.com/timtadh/dynagrok/localize/discflo"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	main := NewLocalizeMain(c)
	st := stat.NewCommand(c)
	ev := eval.NewCommand(c)
	df := discflo.NewCommand(c)
	return cmd.Concat(
		main,
		cmd.Commands(map[string]cmd.Runnable{
			st.Name(): st,
			ev.Name(): ev,
			df.Name(): df,
		}),
	)
}

func NewLocalizeMain(c *cmd.Config) cmd.Runnable {
	return cmd.Cmd("localize",
	`[options]`,
	`
Option Flags
    -h,--help                         Show this message
`,
	"",
	[]string{},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		for _, oa := range optargs {
			switch oa.Opt() {}
		}
		return args, nil
	})
}

