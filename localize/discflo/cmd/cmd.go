package cmd

import (
	"fmt"
	"strconv"
	"time"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/discflo/web"
	"github.com/timtadh/dynagrok/localize/mine"
	minecmd "github.com/timtadh/dynagrok/localize/mine/cmd"
	"github.com/timtadh/dynagrok/localize/test"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	var o discflo.Options
	web := web.NewCommand(c, &o)
	return cmd.Concat(
		cmd.Annotate(
			cmd.Join(
				"disc-flo",
				minecmd.NewOptionParser(c, &o.Options),
				NewOptionParser(c, &o),
			),
			"disc-flo",
			"", "[options]",
			"\nOptions", minecmd.Notes,
		),
		minecmd.NewAlgorithmParser(c, &o.Options),
		cmd.Commands(map[string]cmd.Runnable{
			"":         NewRunner(c, &o),
			web.Name(): web,
		}),
	)
}

func NewOptionParser(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
		"",
		``,
		`
--minimize-tests=<int>            Use test case minimization to minimize the
                                  failing tests.
--failure-oracle=<path>           A failure oracle to filter out graphs with
                                  non-failing minimized tests.
--db-scan-epsilon=<float>         Distance epsilon to use in DBScan
--debug=<int>                     Debug level >= 0
`,
		"",
		[]string{
			"minimize-tests=",
			"failure-oracle=",
			"db-scan-epsilon=",
			"debug=",
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
					m, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected a int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					o.DiscfloOpts = append(o.DiscfloOpts, discflo.Tests(o.Failing), discflo.Minimize(m))
				case "--db-scan-epsilon":
					e, err := strconv.ParseFloat(oa.Arg(), 64)
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected an float (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					o.DiscfloOpts = append(o.DiscfloOpts, discflo.DbScanEpsilon(e))
				case "--debug":
					d, err := strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "Could not parse arg to `%v` expected a int (got %v). err: %v", oa.Opt(), oa.Arg(), err)
					}
					o.DiscfloOpts = append(o.DiscfloOpts, discflo.DebugLevel(d))
				}
			}
			if oracle != nil {
				fex, err := test.SingleInputExecutor(o.BinArgs, oracle)
				if err != nil {
					return nil, cmd.Err(2, err)
				}
				o.DiscfloOpts = append(o.DiscfloOpts, discflo.Oracle(fex))
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
			fmt.Println(result.ScoredLocations())
			return nil, nil
		})
}
