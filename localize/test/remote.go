package test


import (
	"os"
	"os/exec"
	"bytes"
	"syscall"
	"path/filepath"
	"time"
	"fmt"
	"context"
	"io/ioutil"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

var WIZ = "wizard"

type Remote struct {
	Config *cmd.Config
	Path string
	Timeout time.Duration
	MaxMem int // Maximum Resident Memory in Bytes
}

type RemoteOption func(r *Remote)

func Timeout(t time.Duration) RemoteOption {
	return func(r *Remote) {
		r.Timeout = t
	}
}

func MaxMemory(megabytes int) RemoteOption {
	return func(r *Remote) {
		r.MaxMem = megabytes * 10e7
	}
}

func Config(c *cmd.Config) RemoteOption {
	return func(r *Remote) {
		r.Config = c
	}
}

func NewRemote(path string, opts ...RemoteOption) (r *Remote, err error) {
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.Mode() & 0111 == 0 {
		return nil, errors.Errorf("File %v is not executable", path)
	}
	r = &Remote{
		Path: path,
		Timeout: 2 * time.Second,
		MaxMem: 5 * 10e7, // 50 MB
	}
	for _, opt := range opts {
		opt(r)
	}
	return r, nil
}

func (r *Remote) Env(dgprof string) []string {
	env := []string{
		fmt.Sprintf("PATH=%v", os.Getenv("PATH")),
		fmt.Sprintf("USER=%v", os.Getenv("USER")),
		fmt.Sprintf("HOME=%v", os.Getenv("HOME")),
		fmt.Sprintf("DGPROF=%v", dgprof),
	}
	if r.Config != nil {
		env = append(env, fmt.Sprintf("GOROOT=%v", r.Config.GOROOT))
		env = append(env, fmt.Sprintf("GOPATH=%v", r.Config.GOPATH))
	} else {
		if os.Getenv("GOROOT") != "" {
			env = append(env, fmt.Sprintf("GOROOT=%v", os.Getenv("GOROOT")))
		}
		if os.Getenv("GOPATH") != "" {
			env = append(env, fmt.Sprintf("GOPATH=%v", os.Getenv("GOPATH")))
		}
	}
	return env
}

func (r *Remote) Execute(args []string, stdin []byte) (stdout, stderr, profile, failures []byte, ok bool, err error) {
	_, name := filepath.Split(r.Path)
	dgprof, err := ioutil.TempDir("", fmt.Sprintf("dynagrok-dgprof-%v-", name))
	if err != nil {
		return nil, nil, nil, nil, false, err
	}
	defer os.RemoveAll(dgprof)

	var outbuf, errbuf bytes.Buffer
	inbuf := bytes.NewBuffer(stdin)
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()
	c := exec.CommandContext(ctx, r.Path, args...)
	c.Env = r.Env(dgprof)
	c.Stdin = inbuf
	c.Stdout = &outbuf
	c.Stderr = &errbuf

	err = c.Start()
	if err != nil {
		return nil, nil, nil, nil, false, err
	}
	var timeKilled bool
	var memKilled bool
	r.watchMemory(ctx, cancel, c.Process, &memKilled)
	err = c.Wait()
	cerr := ctx.Err()
	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			// skip
		default:
			return nil, nil, nil, nil, false, err
		}
	}
	if cerr != nil && cerr == context.DeadlineExceeded {
		timeKilled = true
	}
	ok = c.ProcessState.Success() && !timeKilled && !memKilled

	fgPath := filepath.Join(dgprof, "flow-graph.dot")
	if _, err := os.Stat(fgPath); err == nil {
		fg, err := os.Open(fgPath)
		if err != nil {
			return nil, nil, nil, nil, false, err
		}
		profile, err = ioutil.ReadAll(fg)
		if err != nil {
			return nil, nil, nil, nil, false, err
		}
	}
	failsPath := filepath.Join(dgprof, "failures")
	if _, err := os.Stat(failsPath); err == nil {
		fails, err := os.Open(failsPath)
		if err != nil {
			return nil, nil, nil, nil, false, err
		}
		failures, err = ioutil.ReadAll(fails)
		if err != nil {
			return nil, nil, nil, nil, false, err
		}
	}

	return outbuf.Bytes(), errbuf.Bytes(), profile, failures, ok, nil
}

func (r *Remote) watchMemory(ctx context.Context, cancel context.CancelFunc, p *os.Process, killed *bool) {
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mem, err := getMemoryUsage(p)
				if err != nil {
					errors.Logf("ERROR", "getMemoryUsage err: %v", err)
				} else if mem > r.MaxMem {
					*killed = true
					cancel()
					errors.Logf(
						"ERROR",
						"Canceled context, memory usage was too high %v > %v",
						mem, r.MaxMem)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// this is not at all portable
// gets the memory usage in bytes
func getMemoryUsage(p *os.Process) (int, error) {
	var ru syscall.Rusage
	err := syscall.Getrusage(p.Pid, &ru)
	if err != nil {
		return 0, err
	}
	return int(ru.Maxrss) * 1000, nil
}

