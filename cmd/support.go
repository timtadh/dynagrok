package cmd

import (
	"os"
	"fmt"
	"go/build"
)

import (
	"golang.org/x/tools/go/loader"
)

// diverges
func Main(r Runnable) {
	args, _, err := r.Run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(err.ExitCode)
	}
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "expected 0 args left got %v\n", args)
		os.Exit(1)
	}
	os.Exit(0)
}


func LoadPkg(c *Config, pkg string) (*loader.Program, error) {
	b := &build.Default
	b.GOPATH = c.GOPATH
	var conf loader.Config
	conf.Import(pkg)
	return conf.Load()
}

