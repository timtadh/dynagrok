package localize

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	discflo "github.com/timtadh/dynagrok/localize/discflo/cmd"
	"github.com/timtadh/dynagrok/localize/locavore"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/stat"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	main := NewLocalizeMain(c)
	st := stat.NewCommand(c)
	df := discflo.NewCommand(c)
	m := mine.NewCommand(c)
	locav := locavore.NewCommand(c)
	return cmd.Concat(
		main,
		cmd.Commands(map[string]cmd.Runnable{
			st.Name():    st,
			df.Name():    df,
			m.Name():     m,
			locav.Name(): locav,
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
				switch oa.Opt() {
				}
			}
			return args, nil
		})
}
