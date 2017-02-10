package mutate

import (
	"path/filepath"
	"fmt"
	"strconv"
	"strings"
	"os"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/instrument"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	return cmd.Cmd(
	"mutate",
	`[options] <pkg>`,
	`
Option Flags
    -h,--help                         Show this message
    -o,--output=<path>                Output file to create (defaults to pkg-name.instr)
    -w,--work=<path>                  Work directory to use (defaults to tempdir)
    --keep-work                       Keep the work directory
    -r,--mutation-rate=<float>        Percentage of statements to mutate (defaults to .01)
    --instrument                      Also instrument the resulting program
    --only=<pkg>                      Only mutate the specified pkg (may be specified multiple
                                      times or with a comma separated list)
    -m,--mutation=<mut>               Only use the specified mutations (may be specified
                                      multiple times or with a comma separated list).
    --mutations                       List the available mutations
`,
	"o:w:r:m:",
	[]string{
		"output=",
		"work=",
		"keep-work",
		"mutation-rate=",
		"instrument",
		"only=",
		"mutation=",
		"mutations",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		output := ""
		keepWork := false
		work := ""
		mutate := .01
		addInstrumentation := false
		only := make(map[string]bool)
		allowedMuts := make(map[string]bool)
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-o", "--output":
				output = oa.Arg()
			case "-w", "--work":
				work = oa.Arg()
			case "-k", "--keep-work":
				keepWork = true
			case "-r", "--mutation-rate":
				f, err := strconv.ParseFloat(oa.Arg(), 64)
				if err != nil {
					return nil, cmd.Usage(r, 1, fmt.Sprintf(
						"%v takes a float. %v", oa.Opt(), err.Error()))
				}
				if f <= 0 || f > 1 {
					return nil, cmd.Usage(r, 1, fmt.Sprintf(
						"%v takes a float between 0 and 1, got: %v", oa.Opt(), f))
				}
				mutate = f
			case "--instrument":
				addInstrumentation = true
			case "--only":
				for _, pkg := range strings.Split(oa.Arg(), ",") {
					only[strings.TrimSpace(pkg)] = true
				}
			case "-m", "--mutation":
				for _, typ := range strings.Split(oa.Arg(), ",") {
					typ = strings.TrimSpace(typ)
					if _, has := MutationTypes[typ]; has {
						allowedMuts[typ] = true
					} else {
						return nil, cmd.Errorf(1, fmt.Sprintf(
							"mutation %v, given in `%v %v`, is support by dynagrok. (use --mutations for list)",
							typ, oa.Opt(), oa.Arg()))
					}
				}
			}
		}
		if len(args) != 1 {
			return nil, cmd.Usage(r, 5, "Expected one package name got %v", args)
		}
		pkgName := args[0]
		if output == "" {
			output = fmt.Sprintf("%v.instr", filepath.Base(pkgName))
		}
		fmt.Println("mutating", pkgName)
		program, err := cmd.LoadPkg(c, pkgName)
		if err != nil {
			return nil, cmd.Usage(r, 6, err.Error())
		}
		mutations, err := Mutate(mutate, only, allowedMuts, addInstrumentation, pkgName, program)
		if err != nil {
			return nil, cmd.Errorf(7, err.Error())
		}
		if addInstrumentation {
			err = instrument.Instrument(pkgName, program)
			if err != nil {
				return nil, cmd.Errorf(8, err.Error())
			}
		}
		// return nil, cmd.Errorf(1, "early exit for no build")
		work, err = instrument.BuildBinary(c, keepWork, work, pkgName, output, program)
		if err != nil {
			return nil, cmd.Errorf(9, err.Error())
		}
		if keepWork {
			f, err := os.Create(filepath.Join(work, "mutations"))
			if err != nil {
				return nil, nil
			}
			defer f.Close()
			for _, m := range mutations {
				_, err := f.Write(m.AsJson())
				if err != nil {
					return nil, cmd.Errorf(10, "error trying to serialize exported mutation: %v", err)
				}
				_, err = fmt.Fprintln(f)
				if err != nil {
					return nil, cmd.Errorf(10, "error trying to print a new line: %v", err)
				}
			}
		}
		return nil, nil
	})
}
