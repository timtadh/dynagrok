package instrument

import (
	"fmt"
	"path/filepath"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	return cmd.Cmd(
		"instrument",
		`[options] <pkg>`,
		`
Option Flags
    -h,--help                         Show this message
    -o,--output=<path>                Output file to create (defaults to pkg-name.instr)
    -w,--work=<path>                  Work directory to use (defaults to tempdir)
    --keep-work                       Keep the work directory
`,
		"o:w:",
		[]string{
			"output=",
			"work=",
			"keep-work",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			fmt.Println(c)
			output := ""
			keepWork := false
			work := ""
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-o", "--output":
					output = oa.Arg()
				case "-w", "--work":
					work = oa.Arg()
				case "-k", "--keep-work":
					keepWork = true
				}
			}
			if len(args) != 1 {
				return nil, cmd.Usage(r, 5, "Expected one package name got %v", args)
			}
			pkgName := args[0]
			if output == "" {
				output = fmt.Sprintf("%v.instr", filepath.Base(pkgName))
			}
			fmt.Println("instrumenting", pkgName)
			program, err := cmd.LoadPkg(c, pkgName)
			if err != nil {
				return nil, cmd.Usage(r, 6, err.Error())
			}
			err = Instrument(pkgName, program)
			if err != nil {
				return nil, cmd.Errorf(7, err.Error())
			}
			_, err = BuildBinary(c, keepWork, work, pkgName, output, program)
			if err != nil {
				return nil, cmd.Errorf(8, err.Error())
			}
			return nil, nil
		})
}
