package cmd

import (
	"os"
	"fmt"
)

import ()

// diverges
func Main(argv []string, r Runnable) {
	args, _, err := r.Run(argv)
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

