package dgruntime

import (
	"fmt"
	"os"
	"sync"
	"runtime"
	"strings"
)

type Execution struct {
	m sync.Mutex
	Goroutines []*Goroutine
	Profile    *Profile
	OutputDir  string
	mergeCh    chan *Goroutine
	async      sync.WaitGroup
	fails      []string
	failed     map[string]bool
}

var execMu sync.Mutex
var exec *Execution

func execCheck() {
	if exec == nil {
		execMu.Lock()
		if exec == nil {
			exec = newExecution()
		}
		runtime.SetFinalizer(exec, shutdown)
		execMu.Unlock()
	}
}

func pjoin(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}

func newExecution() *Execution {
	outputDir := "/tmp/dynagrok-profile"
	if os.Getenv("DGPROF") != "" {
		outputDir = os.Getenv("DGPROF")
	}
	if err := os.MkdirAll(outputDir, os.ModeDir|0775); err != nil {
		panic(fmt.Errorf("dynagrok's dgruntime could not make directory %v", outputDir))
	}
	e := &Execution{
		Profile: &Profile{
			Calls: make(map[Call]int),
			Funcs: make(map[uintptr]*Function),
			Flows: make(map[FlowEdge]int),
			Positions: make(map[BlkEntrance]string),
		},
		OutputDir: outputDir,
		mergeCh: make(chan *Goroutine, 15),
		failed: make(map[string]bool),
	}
	e.growGoroutines()
	go func() {
		e.async.Add(1)
		for g := range e.mergeCh {
			e.merge(g)
		}
		e.async.Done()
	}()
	return e
}

func (e *Execution) Goroutine(id int64) *Goroutine {
	for id >= int64(len(e.Goroutines)) {
		e.m.Lock()
		for id >= int64(len(e.Goroutines)) {
			e.growGoroutines()
		}
		e.m.Unlock()
	}
	if e.Goroutines[id] == nil {
		e.m.Lock()
		if e.Goroutines[id] == nil {
			// Println(fmt.Sprintf("new goroutine %d", id))
			e.Goroutines[id] = newGoroutine(id)
		}
		e.m.Unlock()
	}
	return e.Goroutines[id]
}

func (e *Execution) Fail(pos string) {
	e.m.Lock()
	if !e.failed[pos] {
		e.failed[pos] = true
		e.fails = append(e.fails, pos)
	}
	e.m.Unlock()
}

func (e *Execution) growGoroutines() {
	n := make([]*Goroutine, (len(e.Goroutines)+1)*2)
	copy(n, e.Goroutines)
	// for i := len(e.Goroutines); i < len(n); i++ {
	// 	n[i] = newGoroutine(int64(i))
	// }
	e.Goroutines = n
}

func (e *Execution) Merge(g *Goroutine) {
	e.mergeCh<-g
}

func (e *Execution) merge(g *Goroutine) {
	e.m.Lock()
	defer e.m.Unlock()
	if !g.Closed {
		return
	}
	e.Profile.CallCount += g.CallCount
	for _, fn := range g.Funcs {
		if x, has := e.Profile.Funcs[fn.FuncPc]; has {
			x.Merge(fn)
		} else {
			e.Profile.Funcs[fn.FuncPc] = fn
		}
	}
	for call, count := range g.Calls {
		e.Profile.Calls[call] += count
	}
	for edge, count := range g.Flows {
		e.Profile.Flows[edge] += count
	}
	for be, pos := range g.Positions {
		e.Profile.Positions[be] = pos
	}
}

func shutdown(e *Execution) {
	fmt.Println("starting shut down")
	execMu.Lock()
	defer execMu.Unlock()
	if e == nil {
		return
	}
	for _, g := range e.Goroutines {
		if g == nil {
			continue
		}
		g.m.Lock()
		if !g.Closed && len(g.Calls) > 0 {
			g.m.Unlock()
			g.Exit()
		}
	}
	close(e.mergeCh)
	e.async.Wait()
	e.m.Lock()
	defer e.m.Unlock()
	if !e.Profile.Empty() {
		graphPath := pjoin(e.OutputDir, "flow-graph.dot")
		fmt.Println("writing flow-graph to:", graphPath)
		fout, err := os.Create(graphPath)
		if err != nil {
			panic(err)
		}
		defer fout.Close()
		e.Profile.Serialize(fout)
	}
	if len(e.fails) > 0 {
		failPath := pjoin(e.OutputDir, "failures")
		fmt.Printf("The program registered %v failures\n", len(e.fails))
		fmt.Println("writing failures to:", failPath)
		fout, err := os.Create(failPath)
		if err != nil {
			panic(err)
		}
		defer fout.Close()
		for _, f := range e.fails {
			fmt.Printf("fail: %v\n", f)
			_, err := fmt.Fprintln(fout, f)
			if err != nil {
				panic(err)
			}
		}
	}
	fmt.Println("done shutting down")
}
