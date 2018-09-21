package cmd

import (
	"fmt"
	"go/build"
	"os"
)

import (
	"golang.org/x/tools/go/loader"
)

// diverges
func Main(r Runnable, cleans ...*func()) {
	args, err := r.Run(os.Args[1:])
	for _, c := range cleans {
		if c != nil && *c != nil {
			(*c)()
		}
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(err.ExitCode)
	}
	if len(args) != 0 {
		fmt.Printf("expected 0 args left got %v\n", args)
		os.Exit(1)
	}
	os.Exit(0)
}

func BuildContext(c *Config) *build.Context {
	b := &build.Default
	b.GOROOT = c.GOROOT
	b.GOPATH = c.GOPATH
	return b
}

func LoadPkg(c *Config, pkg string) (*loader.Program, error) {
	var conf loader.Config
	conf.Build = BuildContext(c)
	conf.Build.CgoEnabled = true
	conf.Import(pkg)
	return conf.Load()
}
