package instrument

import (
	"os"
	"os/exec"
	"io/ioutil"
	"path/filepath"
	"fmt"
	"go/printer"
	"go/build"
	"go/types"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/loader"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

// var config = printer.Config{Mode: printer.UseSpaces | printer.TabIndent | printer.SourcePos, Tabwidth: 8}
var config = printer.Config{Tabwidth: 8}


type binaryBuilder struct {
	config *cmd.Config
	buildContext *build.Context
	program *loader.Program
	entry string
	work string
	output string
}


func BuildBinary(c *cmd.Config, keepWork bool, work, entryPkgName, output string, program *loader.Program) (err error) {
	if work == "" {
		work, err = ioutil.TempDir("", fmt.Sprintf("dynagrok-build-%v-", filepath.Base(entryPkgName)))
		if err != nil {
			return err
		}
	}
	if !keepWork {
		defer os.RemoveAll(work)
	}
	errors.Logf("INFO", "work-dir %v", work)
	b := &binaryBuilder{
		config: c,
		buildContext: cmd.BuildContext(c),
		program: program,
		entry: entryPkgName,
		work: work,
		output: output,
	}
	return b.Build()
}


func (b *binaryBuilder) basePaths() paths {
	basePaths := make([]string, 0, 10)
	basePaths = append(basePaths, b.buildContext.GOROOT)
	paths := strings.Split(b.buildContext.GOPATH, ":")
	for _, path := range paths {
		if path != "" {
			basePaths = append(basePaths, path)
		}
	}
	return basePaths
}

type paths []string

func (ps paths) TrimPrefix(s string) string {
	for _, path := range ps {
		if strings.HasPrefix(s, path) {
			return strings.TrimPrefix(strings.TrimPrefix(s, path), "/")
		}
	}
	return s
}

func (b *binaryBuilder) Build() error {
	basePaths := b.basePaths()
	for pkgType, pkgInfo := range b.program.AllPackages {
		if err := b.createDir(pkgType); err != nil {
			return err
		}
		for _, f := range pkgInfo.Files {
			to := filepath.Join(b.work, basePaths.TrimPrefix(b.program.Fset.File(f.Pos()).Name()))
			fout, err := os.Create(to)
			if err != nil {
				return err
			}
			err = config.Fprint(fout, b.program.Fset, f)
			fout.Close()
			if err != nil {
				return errors.Errorf("Could not serialize tree at %v tree %v error: %v", to, f, err)
			}
		}
	}
	return b.goBuild()
}

func (b *binaryBuilder) goBuild() error {
	c := exec.Command("go", "build", "-o", b.output, b.entry)
	c.Env = append(c.Env, fmt.Sprintf("GOPATH=%v", b.work))
	output, err := c.CombinedOutput()
	fmt.Fprintln(os.Stderr, c.Path, strings.Join(c.Args[1:], " "))
	fmt.Fprintln(os.Stderr, string(output))
	return err
}

func (b *binaryBuilder) createDir(pkg *types.Package) error {
	path := filepath.Join(b.work, "src", pkg.Path())
	return os.MkdirAll(path, os.ModeDir|os.ModeTemporary|0775)
}

