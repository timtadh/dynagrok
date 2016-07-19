package cmd

import (
	"os"
	"fmt"
	"os/signal"
	"runtime/pprof"
	"syscall"
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig:=<-sigs
		pprof.StopCPUProfile()
		f.Close()
		panic(fmt.Errorf("caught signal: %v", sig))
	}()
	cleanup := func() {
		pprof.StopCPUProfile()
		f.Close()
	}
	return cleanup, nil
}

