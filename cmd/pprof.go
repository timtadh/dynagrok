package cmd

import (
	"os"
	"fmt"
	"os/signal"
	"runtime/pprof"
	"syscall"
)

import (
	"github.com/timtadh/data-structures/errors"
)

func CPUProfile(output string) (func(), *Error) {
	f, err := os.Create(output)
	if err != nil {
		return nil, &Error{err, -2}
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		return nil, &Error{err, -2}
	}
	errors.Logf("DEBUG", "started cpu profile:  %v", output)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	cleanup := func() {
		errors.Logf("DEBUG", "closing cpu profile")
		pprof.StopCPUProfile()
		err := f.Close()
		errors.Logf("DEBUG", "closed cpu profile, err: %v", err)
	}
	go func() {
		sig:=<-sigs
		cleanup()
		panic(fmt.Errorf("caught signal: %v", sig))
	}()
	return cleanup, nil
}

