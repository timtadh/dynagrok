package test


import (
	"os"
	"os/exec"
	"syscall"
	"bytes"
	"path/filepath"
	"time"
	"fmt"
	"context"
	"io/ioutil"
	"strings"
	"strconv"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

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

func MaxMegabytes(megabytes int) RemoteOption {
	return func(r *Remote) {
		r.MaxMem = megabytes * 1000000
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
		MaxMem: 50000000, // 50 MB
	}
	for _, opt := range opts {
		opt(r)
	}
	return r, nil
}

func (r *Remote) Reconfig(opts ...RemoteOption) {
	for _, opt := range opts {
		opt(r)
	}
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
	ctx, cancel := context.WithCancel(context.Background())
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
	r.watch(ctx, cancel, c, &timeKilled, &memKilled)
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
		errors.Logf("ERROR", "Killed, too much time used")
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

func (r *Remote) watch(ctx context.Context, cancel context.CancelFunc, c *exec.Cmd, timeKilled, memKilled *bool) {
	kill := func() {
		if c.ProcessState != nil {
			return
		}
		err := c.Process.Signal(syscall.SIGINT)
		if err != nil && c.ProcessState != nil {
			cancel()
		}
		time.AfterFunc(100 * time.Nanosecond, func() {
			if c.ProcessState != nil {
				cancel()
			}
		})
	}
	go func() {
		timer := time.NewTimer(r.Timeout)
		defer func() {
			if timer.Stop() {
				<-timer.C
			}
		}()
		ticker := time.NewTicker(50 * time.Nanosecond)
		defer ticker.Stop()
		for {
			select {
			case <-timer.C:
				errors.Logf("ERROR", "Killed, time limit exceeded")
				kill()
				return
			case <-ticker.C:
				mem, err := getMemoryUsage(c.Process)
				fmt.Println("mem", mem, r.MaxMem)
				if err != nil {
					errors.Logf("ERROR", "getMemoryUsage err: %v", err)
				} else if mem > r.MaxMem {
					*memKilled = true
					errors.Logf(
						"ERROR",
						"Killed, memory usage was too high %v > %v",
						mem, r.MaxMem)
					kill()
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
	pid := fmt.Sprintf("%d", p.Pid)
	statmPath := filepath.Join("/proc", pid, "statm")
	var pages int
	if f, err := os.Open(statmPath); os.IsNotExist(err) {
		return 0, nil
	} else if err != nil {
		return 0, err
	} else {
		bits, err := ioutil.ReadAll(f)
		if err != nil {
			return 0, err
		}
		statm := string(bits)
		split := strings.Split(statm, " ")
		if len(split) < 2 {
			return 0, errors.Errorf("statm in unexpected format: %v", statm)
		}
		pages, err = strconv.Atoi(split[1])
		if err != nil {
			return 0, errors.Errorf("could not parse rss: %v", err)
		}
	}
	return pages * os.Getpagesize(), nil
}

