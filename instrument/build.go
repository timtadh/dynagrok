package instrument

import (
	"os"
	"os/exec"
	"io"
	"io/ioutil"
	"path/filepath"
	"fmt"
	"go/ast"
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
	"github.com/timtadh/dynagrok/dgruntime"
)

// var config = printer.Config{Mode: printer.UseSpaces | printer.TabIndent | printer.SourcePos, Tabwidth: 8}
var config = printer.Config{Tabwidth: 8}


type binaryBuilder struct {
	config *cmd.Config
	buildContext *build.Context
	program *loader.Program
	entry string
	_work, root, path string
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
		_work: work,
		root: filepath.Join(work, "goroot"),
		path: filepath.Join(work, "gopath"),
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

func (ps paths) PrefixedBy(s string) string {
	for _, path := range ps {
		if strings.HasPrefix(s, path) {
			return path
		}
	}
	panic("unreachable")
}

func (b *binaryBuilder) Build() error {
	err := b.copyDir(
		filepath.Join(b.config.GOROOT),
		filepath.Join(b.root),
	)
	if err != nil {
		return err
	}
	err = b.copyDir(
		filepath.Join(b.config.DGPATH, "dgruntime"),
		filepath.Join(b.root, "src", "dgruntime"),
	)
	if err != nil {
		return err
	}
	err = b.copyDir(
		filepath.Join(b.config.DGPATH, "src", "runtime"),
		filepath.Join(b.root, "src", "runtime"),
	)
	if err != nil {
		return err
	}
	basePaths := b.basePaths()
	for pkgType, pkgInfo := range b.program.AllPackages {
		root, err := b.createDir(basePaths, pkgType, pkgInfo.Files)
		errors.Logf("DEBUG", "pkgInfo %v %v", pkgInfo, root)
		if err != nil {
			return err
		}
		if len(pkgInfo.BuildPackage.CgoFiles) > 0 {
			continue
		}
		if dgruntime.ExcludedPkg(pkgInfo.Pkg.Path()) {
			continue
		}
		for _, f := range pkgInfo.Files {
			to := filepath.Join(root, basePaths.TrimPrefix(b.program.Fset.File(f.Pos()).Name()))
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
	err = b.goInstallRuntime()
	if err != nil {
		return err
	}
	return b.goBuild()
}

func (b *binaryBuilder) goEnv() []string {
	env := make([]string, 0, len(os.Environ()))
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "GOPATH=") {
			continue
		}
		if strings.HasPrefix(item, "GOROOT=") {
			continue
		}
		env = append(env, item)
	}
	env = append(env, fmt.Sprintf("GOROOT=%v", b.root))
	env = append(env, fmt.Sprintf("GOPATH=%v", b.path))
	return env
}

func (b *binaryBuilder) goInstallRuntime() error {
	goProg := filepath.Join(b.root, "bin", "go")
	c := exec.Command(goProg, "build", "-i", b.entry)
	c.Env = b.goEnv()
	fmt.Fprintln(os.Stderr, c.Path, strings.Join(c.Args[1:], " "))
	output, err := c.CombinedOutput()
	fmt.Fprintln(os.Stderr, string(output))
	return err
}
func (b *binaryBuilder) goBuild() error {
	goProg := filepath.Join(b.root, "bin", "go")
	c := exec.Command(goProg, "build", "-o", b.output, b.entry)
	c.Env = b.goEnv()
	fmt.Fprintln(os.Stderr, c.Path, strings.Join(c.Args[1:], " "))
	output, err := c.CombinedOutput()
	fmt.Fprintln(os.Stderr, string(output))
	return err
}

func (b *binaryBuilder) createDir(basePaths paths, pkg *types.Package, pkgFiles []*ast.File) (root string, err error) {
	var src string
	for _, path := range basePaths {
		if _, err := os.Stat(filepath.Join(path, "src", pkg.Path())); err == nil {
			src = path
			break
		}
	}
	srcDir, err := os.Open(filepath.Join(src, "src", pkg.Path()))
	if err != nil {
		return "", err
	}
	files, err := srcDir.Readdir(0)
	srcDir.Close()
	if err != nil {
		return "", err
	}
	root = b.path
	if filepath.Clean(src) == filepath.Clean(b.buildContext.GOROOT) {
		root = b.root
	}
	err = os.MkdirAll(filepath.Join(root, "src", pkg.Path()), os.ModeDir|os.ModeTemporary|0775)
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		from, err := os.Open(filepath.Join(src, "src", pkg.Path(), name))
		if err != nil {
			return "", err
		}
		to, err := os.Create(filepath.Join(root, "src", pkg.Path(), name))
		if err != nil {
			from.Close()
			return "", err
		}
		_, err = io.Copy(to, from)
		from.Close()
		to.Close()
		if err != nil {
			return "", err
		}
	}
	return root, nil
}

func (b *binaryBuilder) copyDir(src, targ string) error {
	err := os.MkdirAll(targ, os.ModeDir|os.ModeTemporary|0775)
	if err != nil {
		return err
	}
	srcDir, err := os.Open(src)
	if err != nil {
		return err
	}
	files, err := srcDir.Readdir(0)
	srcDir.Close()
	if err != nil {
		return err
	}
	for _, f := range files {
		name := f.Name()
		if f.IsDir() {
			err := b.copyDir(filepath.Join(src, name), filepath.Join(targ, name))
			if err != nil {
				return nil
			}
		} else {
			stat, err := os.Stat(filepath.Join(src, name))
			if err != nil {
				return err
			}
			from, err := os.Open(filepath.Join(src, name))
			if err != nil {
				return err
			}
			to, err := os.Create(filepath.Join(targ, name))
			if err != nil {
				from.Close()
				return err
			}
			_, err = io.Copy(to, from)
			from.Close()
			to.Close()
			if err != nil {
				return err
			}
			err = os.Chmod(filepath.Join(targ, name), stat.Mode())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

