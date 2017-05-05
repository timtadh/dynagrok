package eval

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/eval"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/getopt"
)

func NewCommand(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
		"eval",
		`[options]`,
		`
Evaluate a fault localization method from ground truth

Option Flags
    -h,--help                         Show this message
    -o,--output=<path>                Place to write CSV of evaluation
    -f,--faults=<path>                Path to a fault file.
    --max=<int>                       Maximum number of states in the chain
    -j,--jump-prs=<float64>           Probability of taking jumps in chains which have them
    -m,--method=<method>
    -e,--eval-method=<eval-method>

Methods
    DISCFLO
    SBBFL
    CBSFL

Eval Methods
    RankList
    Markov
`,
		"f:o:j:m:e:",
		[]string{
			"faults=",
			"output=",
			"max=",
			"jump-prs=",
			"method=",
			"eval-method=",
			"minimize-tests=",
			"failure-oracle=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			outputPath := ""
			methods := make([]string, 0, 10)
			evalMethods := make([]string, 0, 10)
			max := 100
			faultsPath := ""
			jumpPrs := []float64{}
			var oracle *test.Remote
			var opts []discflo.DiscfloOption
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
				case "-o", "--output":
					outputPath = oa.Arg()
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
					opts = append(opts, discflo.Tests(o.Failing), discflo.Minimize(m))
				case "-m", "--method":
					for _, part := range strings.Split(oa.Arg(), ",") {
						methods = append(methods, strings.TrimSpace(part))
					}
				case "-e", "--eval-method":
					for _, part := range strings.Split(oa.Arg(), ",") {
						evalMethods = append(evalMethods, strings.TrimSpace(part))
					}
				case "--max":
					var err error
					max, err = strconv.Atoi(oa.Arg())
					if err != nil {
						return nil, cmd.Errorf(1, "For flag %v expected an int got %v. err: %v", oa.Opt, oa.Arg(), err)
					}
				case "-j", "--jump-prs":
					for _, part := range strings.Split(oa.Arg(), ",") {
						jumpPr, err := strconv.ParseFloat(part, 64)
						if err != nil {
							return nil, cmd.Errorf(1, "For flag %v expected a float got %v. err: %v", oa.Opt, oa.Arg(), err)
						}
						if jumpPr < 0 || jumpPr >= 1 {
							return nil, cmd.Errorf(1, "For flag %v expected a float between 0-1. got %v", oa.Opt, oa.Arg())
						}
						jumpPrs = append(jumpPrs, jumpPr)
					}
				}
			}
			if len(jumpPrs) <= 0 {
				jumpPrs = append(jumpPrs, (1. / 10.))
			}
			if len(methods) <= 0 {
				methods = append(methods, "DISCFLO", "SBBFL", "CBSFL")
			}
			if len(evalMethods) <= 0 {
				evalMethods = append(evalMethods, "RankList", "Markov")
			}
			if faultsPath == "" {
				return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
			}
			faults, err := mine.LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			fmt.Println("max", max)
			for _, f := range faults {
				fmt.Println(f)
			}
			evaluate := func(evalMethod, method string, o *discflo.Options) (eval.EvalResults, error) {
				if evalMethod == "Markov" {
					results := make(eval.EvalResults, 0, 10)
					for _, chain := range eval.Chains[method] {
						r, err := eval.Evaluate(faults, o, o.Score, evalMethod, method, o.ScoreName, chain, max, jumpPrs)
						if err != nil {
							return nil, err
						}
						results = append(results, r...)
					}
					return results, nil
				} else if method == "SBBFL" {
					return nil, nil
				}
				return eval.Evaluate(faults, o, o.Score, evalMethod, method, o.ScoreName, "", max, jumpPrs)
			}
			results := make(eval.EvalResults, 0, 10)
			fmt.Println("methods", methods)
			for _, evalMethod := range evalMethods {
				for _, method := range methods {
					if method == "DISCFLO" && oracle != nil && len(opts) > 0 {
						o := o.Copy()
						fex, err := test.SingleInputExecutor(o.BinArgs, oracle)
						if err != nil {
							return nil, cmd.Err(2, err)
						}
						opts = append(opts, discflo.Oracle(fex))
						o.DiscfloOpts = append(o.DiscfloOpts, opts...)
						r, err := evaluate(evalMethod, method+" + FP-Filter", o)
						if err != nil {
							return nil, cmd.Err(1, err)
						}
						results = append(results, r...)
					}
					r, err := evaluate(evalMethod, method, o)
					if err != nil {
						return nil, cmd.Err(1, err)
					}
					results = append(results, r...)
				}
			}
			var output io.Writer = os.Stdout
			if outputPath != "" {
				f, err := os.Create(outputPath)
				if err != nil {
					return nil, cmd.Err(1, err)
				}
				defer f.Close()
				output = f
			}
			fmt.Fprintln(output, results)
			return nil, nil
		})
}
