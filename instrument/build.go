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
	"crypto/sha1"
	"bytes"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/loader"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/dgruntime/excludes"
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
	_, err := os.Stat(filepath.Join(b.root, "src"))
	if err != nil && os.IsNotExist(err) {
		err = b.copyDir(
			filepath.Join(b.config.GOROOT),
			filepath.Join(b.root),
			func(path string) bool {
				return path == "bin" || path == "pkg" || path == ".git"
			},
		)
		if err != nil {
			return err
		}
		err = b.dropVersion()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	err = b.copyDir(
		filepath.Join(b.config.DGPATH, "src", "runtime"),
		filepath.Join(b.root, "src", "runtime"),
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
	err = b.rebuildGo()
	if err != nil {
		return err
	}
	basePaths := b.basePaths()
	anyStdlib := false
	for pkgType, pkgInfo := range b.program.AllPackages {
		if len(pkgInfo.BuildPackage.CgoFiles) > 0 {
			continue
		}
		if excludes.ExcludedPkg(pkgInfo.Pkg.Path()) {
			continue
		}
		stdlib, root, err := b.createDir(basePaths, pkgType, pkgInfo.Files)
		if err != nil {
			return err
		}
		if stdlib {
			anyStdlib = true
		}
		errors.Logf("DEBUG", "%v -> %v", pkgInfo, root)
		for _, f := range pkgInfo.Files {
			to := filepath.Join(root, basePaths.TrimPrefix(b.program.Fset.File(f.Pos()).Name()))
			errors.Logf("DEBUG", "%v -> %v", b.program.Fset.File(f.Pos()).Name(), to)
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
	return b.goBuild(anyStdlib)
}

func (b *binaryBuilder) noEnv() []string {
	return []string{
		fmt.Sprintf("PATH=%v", os.Getenv("PATH")),
		fmt.Sprintf("USER=%v", os.Getenv("USER")),
		fmt.Sprintf("HOME=%v", os.Getenv("HOME")),
	}
}

func (b *binaryBuilder) goEnv() []string {
	env := b.noEnv()
	env = append(env, fmt.Sprintf("GOROOT=%v", b.root))
	env = append(env, fmt.Sprintf("GOPATH=%v", b.path))
	return env
}

func (b *binaryBuilder) getWorkingRoot() (string, error) {
	goBin := filepath.Join(b.root, "bin", "go")
	_, err := os.Stat(goBin)
	if err != nil && os.IsNotExist(err) {
		return "<none>", nil
	} else if err != nil {
		return "", err
	}
	c := exec.Command(goBin, "env", "GOROOT")
	c.Env = b.noEnv()
	fmt.Fprintf(os.Stderr, "%v %v\n", c.Path, strings.Join(c.Args[1:], " "))
	if output, err := c.CombinedOutput(); err != nil {
		return "", err
	} else {
		return strings.TrimSpace(string(output)), nil
	}
}

func (b *binaryBuilder) cd(path string, do func() error) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "cd %v\n", path)
	err = os.Chdir(path)
	if err != nil {
		return err
	}
	defer func() {
		fmt.Fprintf(os.Stderr, "cd %v\n", cwd)
		os.Chdir(cwd)
	}()
	return do()
}

func (b *binaryBuilder) rebuildGo() error {
	if goroot, err := b.getWorkingRoot(); err != nil {
		return err
	} else if goroot != b.root {
		errors.Logf("INFO", "go env GOROOT -> %v \n\t\t\t\t\t\t\t\t\t\t\t\t\t\twanted %v", goroot, b.root)
		err := b.cd(filepath.Join(b.root, "src"), func() error {
			c := exec.Command("bash", "make.bash")
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Env = b.goEnv()
			fmt.Fprintf(os.Stderr, "GOROOT=%v GOPATH=%v %v %v\n", b.root, b.path, c.Path, strings.Join(c.Args[1:], " "))
			return c.Run()
		})
		if err != nil {
			return err
		}
	} else {
		return nil
	}
	if goroot, err := b.getWorkingRoot(); err != nil {
		return err
	} else if goroot != b.root {
		return errors.Errorf("%v env GOROOT -> %v != %v", filepath.Join(b.root,"bin","go"), goroot, b.root)
	}
	return nil
}

func (b *binaryBuilder) version() (string, error) {
	goBin := filepath.Join(b.config.GOROOT, "bin", "go")
	c := exec.Command(goBin, "version")
	c.Env = b.noEnv()
	fmt.Fprintf(os.Stderr, "%v %v\n", c.Path, strings.Join(c.Args[1:], " "))
	if output, err := c.CombinedOutput(); err != nil {
		return "", err
	} else {
		version := strings.TrimSpace(string(output))
		parts := strings.Split(version, " ")
		if len(parts) < 3 {
			return "", errors.Errorf("unexpected output from `go version` -> %v", version)
		}
		return parts[2], nil
	}
}

func (b *binaryBuilder) dropVersion() error {
	version, err := b.version()
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(b.root, "VERSION"))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(version))
	return err
}

func (b *binaryBuilder) goBuild(stdlib bool) error {
	goBin := filepath.Join(b.root, "bin", "go")
	if stdlib {
		c := exec.Command(goBin, "install", "-v", "std")
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = b.goEnv()
		fmt.Fprintln(os.Stderr, strings.Join(c.Env, " "), c.Path, strings.Join(c.Args[1:], " "))
		err := c.Run()
		if err != nil {
			return err
		}
	}
	{
		// ignore os.Remove errors because it is a best effort thing
		os.Remove(filepath.Join(b.root, "pkg", "linux_amd64", "dgruntime.a"))
		os.Remove(filepath.Join(b.root, "pkg", "linux_amd64", "dgruntime", "excludes.a"))
		c := exec.Command(goBin, "install", "-v", "dgruntime")
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = b.goEnv()
		fmt.Fprintln(os.Stderr, strings.Join(c.Env, " "), c.Path, strings.Join(c.Args[1:], " "))
		err := c.Run()
		if err != nil {
			return err
		}
	}
	c := exec.Command(goBin, "build", "-o", b.output, b.entry)
	c.Env = b.goEnv()
	fmt.Fprintln(os.Stderr, c.Path, strings.Join(c.Args[1:], " "))
	output, err := c.CombinedOutput()
	fmt.Fprintln(os.Stderr, string(output))
	return err
}

func (b *binaryBuilder) createDir(basePaths paths, pkg *types.Package, pkgFiles []*ast.File) (stdlib bool, root string, err error) {
	var src string
	for _, path := range basePaths {
		if _, err := os.Stat(filepath.Join(path, "src", pkg.Path())); err == nil {
			src = path
			break
		}
	}
	srcDir, err := os.Open(filepath.Join(src, "src", pkg.Path()))
	if err != nil {
		return false, "", err
	}
	files, err := srcDir.Readdir(0)
	srcDir.Close()
	if err != nil {
		return false, "", err
	}
	root = b.path
	if filepath.Clean(src) == filepath.Clean(b.buildContext.GOROOT) {
		root = b.root
		stdlib = true
	}
	err = os.MkdirAll(filepath.Join(root, "src", pkg.Path()), os.ModeDir|os.ModeTemporary|0775)
	if err != nil {
		return false, "", err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		from, err := os.Open(filepath.Join(src, "src", pkg.Path(), name))
		if err != nil {
			return false, "", err
		}
		to, err := os.Create(filepath.Join(root, "src", pkg.Path(), name))
		if err != nil {
			from.Close()
			return false, "", err
		}
		_, err = io.Copy(to, from)
		from.Close()
		to.Close()
		if err != nil {
			return false, "", err
		}
	}
	return stdlib, root, nil
}

func (b *binaryBuilder) copyDir(src, targ string, skips ...func(string)bool) error {
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
outer:
	for _, f := range files {
		name := f.Name()
		for _, skip := range skips {
			if skip(name) {
				errors.Logf("INFO", "skipping %v", filepath.Join(src, name))
				continue outer
			}
		}
		if f.IsDir() {
			err := b.copyDir(filepath.Join(src, name), filepath.Join(targ, name))
			if err != nil {
				return nil
			}
		} else {
			err := b.copyFile(filepath.Join(src, name), filepath.Join(targ, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *binaryBuilder) copyFile(src, targ string) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	targStat, err := os.Stat(targ)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if targStat != nil {
		from, err := os.Open(src)
		if err != nil {
			return err
		}
		fromBytes, err := ioutil.ReadAll(from)
		if err != nil {
			from.Close()
			return err
		}
		from.Close()
		to, err := os.Open(targ)
		if err != nil {
			return err
		}
		toBytes, err := ioutil.ReadAll(to)
		if err != nil {
			to.Close()
			return err
		}
		to.Close()
		fromSha := sha1.New().Sum(fromBytes)
		toSha := sha1.New().Sum(toBytes)
		if bytes.Equal(fromSha, toSha) {
			// errors.Logf("DEBUG", "skip %v -> %v", src, targ)
			return os.Chmod(targ, srcStat.Mode())
		}
	}
	errors.Logf("DEBUG", "copy %v -> %v", src, targ)
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	to, err := os.Create(targ)
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
	return os.Chmod(targ, srcStat.Mode())
}

