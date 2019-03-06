package instrument

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/getopt"
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
    --instrument-dataflow             Do dataflow instrumentation
    --keep-work                       Keep the work directory
`,
		"o:w:",
		[]string{
			"output=",
			"work=",
			"keep-work",
			"instrument-dataflow",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			fmt.Println(c)
			output := ""
			keepWork := false
			work := ""
			flags := make([]InstrumentOption, 0, 10)
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-o", "--output":
					output = oa.Arg()
				case "-w", "--work":
					work = oa.Arg()
				case "-k", "--keep-work":
					keepWork = true
				case "--instrument-dataflow":
					flags = append(flags, InstrumentDataflow)
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
			pkgName := args[0]
			if output == "" {
				output = fmt.Sprintf("%v.instr", filepath.Base(pkgName))
			}
			fmt.Println("instrumenting", pkgName)
			err := CD(work, func() error {
				program, err := cmd.LoadPkg(c, pkgName)
				if err != nil {
					return err
				}
				err = Instrument(pkgName, program, flags...)
				if err != nil {
					return err
				}
				_, err = BuildBinary(c, work, pkgName, output, program)
				return err
			})
			if err != nil {
				return nil, cmd.Errorf(7, err.Error())
			}
			return nil, nil
		})
}
