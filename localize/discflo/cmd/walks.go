package cmd

import (
	"strconv"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/mine"
)

type walkOpts struct {
	walker mine.Walker
}

func NewWalksParser(c *cmd.Config, o *discflo.Options, wo *walkOpts) cmd.Runnable {
	return cmd.Cmd(
	"k-walks",
	`[options]`,
	`
Option Flags
    -h,--help                         Show this message
    -w,--walks=<int>                  Number of walks to take
`,
	"w:",
	[]string{
		"walks=",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		walks := 10
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-w", "--walks":
				w, err := strconv.Atoi(oa.Arg())
				if err != nil {
					return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
				}
				walks = w
			}
		}
		o.Miner = mine.Walking(wo.walker, walks)
		return args, nil
	})
}

func NewWalkTopColorsParser(c *cmd.Config, o *discflo.Options, wo *walkOpts) cmd.Runnable {
	return cmd.Cmd(
	"walk-top-colors",
	`[options]`,
	`
Option Flags
    -h,--help                         Show this message
    -p,--percent-of-colors<float>     Percent of top colors to walk from
    -w,--walks-per-color=<int>        Number of walks to take per color
    -m,--min-groups-walked=<int>      Minimum number of groups of colors to walk from
`,
	"p:w:m:",
	[]string{
		"percent-of-colors=",
		"walks-per-color=",
		"min-groups-walked=",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		opts := make([]mine.TopColorOpt, 0, 10)
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-p", "--percent-of-colors":
				p, err := strconv.ParseFloat(oa.Arg(), 64)
				if err != nil {
					return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
				}
				opts = append(opts, mine.PercentOfColors(p))
			case "-w", "--walks-per-color":
				w, err := strconv.Atoi(oa.Arg())
				if err != nil {
					return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
				}
				opts = append(opts, mine.WalksPerColor(w))
			case "-m", "--min-groups-walked":
				m, err := strconv.Atoi(oa.Arg())
				if err != nil {
					return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
				}
				opts = append(opts, mine.MinGroupsWalked(m))
			}
		}
		o.Miner = mine.WalkingTopColors(wo.walker, opts...)
		return args, nil
	})
}

func NewURWParser(c *cmd.Config, o *discflo.Options, wo *walkOpts) cmd.Runnable {
	return cmd.Cmd(
	"urw",
	`[options]`,
	`
Option Flags
    -h,--help                         Show this message
`,
	"",
	[]string{},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		wo.walker = mine.UnweightedRandomWalk()
		return args, nil
	})
}

func NewSWRWParser(c *cmd.Config, o *discflo.Options, wo *walkOpts) cmd.Runnable {
	return cmd.Cmd(
	"swrw",
	`[options]`,
	`
Option Flags
    -h,--help                         Show this message
`,
	"",
	[]string{},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		wo.walker = mine.ScoreWeightedRandomWalk()
		return args, nil
	})
}
