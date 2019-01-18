package objectstate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/instrument"
	"github.com/timtadh/getopt"
)

func NewCommand(c *cmd.Config) cmd.Runnable {
	return cmd.Cmd(
		"objectstate",
		`[options] <pkg>`,
		`
Option Flags
    -h,--help                         Show this message
    -o,--output=<path>                Output file to create (defaults to pkg-name.instr)
    -w,--work=<path>                  Work directory to use (defaults to tempdir)
	-m,--method=<method-name>         Name of a specific method to profile
    --keep-work                       Keep the work directory
`,
		"o:w:m:",
		[]string{
			"output=",
			"work=",
			"method=",
			"keep-work",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			fmt.Println(c)
			output := ""
			keepWork := false
			work := ""
			method := ""
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-o", "--output":
					output = oa.Arg()
				case "-w", "--work":
					work = oa.Arg()
				case "-m", "--method":
					method = oa.Arg()
				case "-k", "--keep-work":
					keepWork = true
				}
			}
			if work == "" {
				tmpdir, err := ioutil.TempDir("", fmt.Sprintf("dynagrok-work-"))
				if err != nil {
					return nil, cmd.Errorf(4, "could not make tmp dir for working: %v", err)
				}
				work = tmpdir
			} else {
				err := os.MkdirAll(work, os.ModeDir|0775)
				if err != nil {
					return nil, cmd.Errorf(4, "could not make work dir: %v", err)
				}
			}
			if !keepWork {
				defer os.RemoveAll(work)
			}
			if len(args) != 1 {
				return nil, cmd.Usage(r, 5, "Expected one package name got %v", args)
			}
			if work == "" {
				tmpdir, err := ioutil.TempDir("", fmt.Sprintf("dynagrok-work-"))
				if err != nil {
					return nil, cmd.Errorf(4, "could not make tmp dir for working: %v", err)
				}
				work = tmpdir
			} else {
				err := os.MkdirAll(work, os.ModeDir|0775)
				if err != nil {
					return nil, cmd.Errorf(4, "could not make work dir: %v", err)
				}
			}
			if !keepWork {
				defer os.RemoveAll(work)
			}
			pkgName := args[0]
			if output == "" {
				output = fmt.Sprintf("%v.instr", filepath.Base(pkgName))
			}
			program, err := cmd.LoadPkg(c, pkgName)
			if err != nil {
				return nil, cmd.Usage(r, 6, err.Error())
			}
			fmt.Println("instrumenting for object-state", pkgName)
			err = Instrument(pkgName, method, program)
			if err != nil {
				return nil, cmd.Errorf(7, err.Error())
			}
			fmt.Println("instrumenting", pkgName)
			err = instrument.Instrument(pkgName, program)
			if err != nil {
				return nil, cmd.Errorf(8, err.Error())
			}
			_, err = instrument.BuildBinary(c, work, pkgName, output, program)
			if err != nil {
				return nil, cmd.Errorf(9, err.Error())
			}
			return nil, nil
		})
}
