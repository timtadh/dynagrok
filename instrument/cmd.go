package instrument

import (
	"os"
	"io/ioutil"
	"path/filepath"
	"fmt"
)

import (
	"github.com/timtadh/getopt"
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/loader"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

var Command = cmd.Cmd(
	"instrument",
	`[options] <pkg>`,
	`
Option Flags
    -h,--help                         Show this message
`,
	"o:",
	[]string{
		"output=",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg, xtra ...interface{}) ([]string, interface{}, *cmd.Error) {
		c := xtra[0].(*cmd.Config)
		output := ""
		for _, oa := range optargs {
			switch oa.Opt() {
			case "-o", "--output":
				output = oa.Arg()
			}
		}
		if len(args) != 1 {
			return nil, nil, cmd.Usage(r, 5, "Expected one package name got %v", args)
		}
		pkgName := args[0]
		if output == "" {
			output = fmt.Sprintf("%v.instr", filepath.Base(pkgName))
		}
		fmt.Println("instrumenting", pkgName)
		program, err := cmd.LoadPkg(c, pkgName)
		if err != nil {
			return nil, nil, cmd.Usage(r, 6, err.Error())
		}
		err = Instrument(pkgName, program)
		if err != nil {
			return nil, nil, cmd.Errorf(7, err.Error())
		}
		err = BuildBinary(pkgName, output, program)
		if err != nil {
			return nil, nil, cmd.Errorf(8, err.Error())
		}
		return nil, nil, nil
	},
)

func BuildBinary(entryPkgName, output string, program *loader.Program) (err error) {
	dir, err := ioutil.TempDir("", fmt.Sprintf("dynagrok-build-%v-", filepath.Base(entryPkgName)))
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	return nil
}

func Instrument(entryPkgName string, program *loader.Program) (err error) {
	entry := program.Package(entryPkgName)
	if entry == nil {
		return errors.Errorf("The entry package was not found in the loaded program")
	}
	if entry.Pkg.Name() != "main" {
		return errors.Errorf("The entry package was not main")
	}
	return nil
}

