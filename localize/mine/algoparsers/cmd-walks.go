package algoparsers

import (
	"fmt"
	"strconv"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/opts"
	"github.com/timtadh/getopt"
)

type WalkOpts struct {
	walker mine.Walker
}

func NewWalksParser(c *cmd.Config, o *opts.Options, wo *WalkOpts) cmd.Runnable {
	return cmd.Cmd(
		"k-walks",
		`[options]`,
		`
Option Flags
    -h,--help                         Show this message
    -w,-k,--walks=<int>                  Number of walks to take
`,
		"w:k:",
		[]string{
			"walks=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			walks := 10
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-w", "-k", "--walks":
					w, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					walks = w
				}
			}
			o.Miner = mine.Walking(wo.walker, walks)
			o.MinerName += fmt.Sprintf(" k-walks %v", walks)
			return args, nil
		})
}

func NewWalkTopColorsParser(c *cmd.Config, o *opts.Options, wo *WalkOpts) cmd.Runnable {
	return cmd.Cmd(
		"walk-top-colors",
		`[options]`,
		`
Option Flags
    -h,--help                         Show this message
    -p,--percent-of-colors<float>     Percent of top colors to walk from
    -w,--walks-per-color=<int>        Number of walks to take per color
    -m,--min-groups-walked=<int>      Minimum number of groups of colors to walk from
    -s,--skip-seen-colors             Skip taking walks from colors which have already
                                      been found from some other color.
    --debug=<int>                     Debug level >= 0
`,
		"p:w:m:s",
		[]string{
			"percent-of-colors=",
			"walks-per-color=",
			"min-groups-walked=",
			"skip-seen-colors",
			"debug=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			opts := make([]mine.TopColorOpt, 0, 10)
			walks := 0
			percent := 0.0
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-p", "--percent-of-colors":
					p, err := strconv.ParseFloat(oa.Arg(), 64)
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					opts = append(opts, mine.PercentOfColors(p))
					percent = p
				case "-w", "--walks-per-color":
					w, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					opts = append(opts, mine.WalksPerColor(w))
					walks = w
				case "-m", "--min-groups-walked":
					m, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					opts = append(opts, mine.MinGroupsWalked(m))
				case "-s", "--skip-seen-colors":
					opts = append(opts, mine.SkipSeenColors())
				case "--debug":
					d, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected a int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					opts = append(opts, mine.WTCDebugLevel(d))
				}
			}
			// o.Miner = WalkingTopColors(wo.walker, opts...)
			o.Miner = mine.ParTopColors(wo.walker, opts...)
			o.MinerName += fmt.Sprintf(" walk-top-colors %v:%v", percent, walks)
			return args, nil
		})
}

func NewURWParser(c *cmd.Config, o *opts.Options, wo *WalkOpts) cmd.Runnable {
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
			o.MinerName += "urw"
			return args, nil
		})
}

func NewSWRWParser(c *cmd.Config, o *opts.Options, wo *WalkOpts) cmd.Runnable {
	return cmd.Cmd(
		"swrw",
		`[options]`,
		`
Option Flags
    -h,--help                         Show this message
    --sample-non-maximal              Allow sampling of non-maximal subgraphs
`,
		"",
		[]string{
			"sample-non-maximal",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			opts := make([]mine.SWRWOpt, 0, 10)
			for _, oa := range optargs {
				switch oa.Opt() {
				case "--sample-non-maximal":
					opts = append(opts, mine.SWRWSampleNonMax)
				}
			}
			wo.walker = mine.ScoreWeightedRandomWalk(opts...)
			o.MinerName += "swrw"
			return args, nil
		})
}
