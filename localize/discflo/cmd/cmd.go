package cmd

import (
	"fmt"
	"time"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/discflo/web"
	"github.com/timtadh/dynagrok/localize/eval"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/test"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	var o discflo.Options
	var wo walkOpts
	bb := NewBranchAndBoundParser(c, &o, &wo)
	sleap := NewSLeapParser(c, &o, &wo)
	leap := NewLeapParser(c, &o, &wo)
	urw := NewURWParser(c, &o, &wo)
	swrw := NewSWRWParser(c, &o, &wo)
	walks := NewWalksParser(c, &o, &wo)
	topColors := NewWalkTopColorsParser(c, &o, &wo)
	walkTypes := cmd.Commands(map[string]cmd.Runnable{
		walks.Name():     walks,
		topColors.Name(): topColors,
	})
	evaluate := eval.NewCommand(c, &o)
	web := web.NewCommand(c, &o)
	return cmd.Concat(
		cmd.Annotate(
			cmd.Join(
				"disc-flo",
				mine.NewOptionParser(c, &o.Options),
				NewOptionParser(c, &o),
			),
			"disc-flo",
			"", "[options]",
			"\nOptions", mine.Notes,
		),
		cmd.Commands(map[string]cmd.Runnable{
			bb.Name():    bb,
			sleap.Name(): sleap,
			leap.Name(): leap,
			urw.Name():   cmd.Concat(urw, walkTypes),
			swrw.Name():  cmd.Concat(swrw, walkTypes),
		}),
		cmd.Commands(map[string]cmd.Runnable{
			"":              NewRunner(c, &o),
			evaluate.Name(): evaluate,
			web.Name():      web,
		}),
	)
	// return cmd.Concat(
	// 	NewOptionParser(c, &o),
	// 	cmd.Commands(map[string]cmd.Runnable {
	// 		"": NewRunner(c, &o),
	// 		// "web": web.NewCommand(c, &o),
	// 	}),
	// )
}

func NewOptionParser(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
		"",
		``,
		`
--minimize-tests                  Use test case minimization to minimize the
                                  failing tests.
--failure-oracle=<path>           A failure oracle to filter out graphs with
                                  non-failing minimized tests.
`,
		"",
		[]string{
			"minimize-tests",
			"failure-oracle=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			var oracle *test.Remote
			for _, oa := range optargs {
				switch oa.Opt() {
				case "--failure-oracle":
					r, err := test.NewRemote(oa.Arg(), test.Timeout(10*time.Second), test.Config(c))
					if err != nil {
						return nil, cmd.Err(1, err)
					}
					oracle = r
				case "--minimize-tests":
					o.Minimize = true
				}
			}
			if oracle != nil {
				fex, err := test.SingleInputExecutor(o.BinArgs, oracle)
				if err != nil {
					return nil, cmd.Err(2, err)
				}
				o.Oracle = fex
			}
			return args, nil
		})
}

func NewRunner(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.BareCmd(
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			miner := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
			if o.Score == nil {
				return nil, cmd.Usage(r, 2, "You must supply a score (see -s or --scores)")
			}
			clusters, err := discflo.Localizer(o)(miner)
			if err != nil {
				return nil, cmd.Err(3, err)
			}
			result := clusters.RankColors(miner)
			fmt.Println(result)
			fmt.Println(result.StatResult())
			return nil, nil
		})
}
