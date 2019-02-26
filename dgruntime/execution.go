package dgruntime

import (
	"dgruntime/dgtypes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Execution struct {
	m          sync.Mutex
	Goroutines []*Goroutine
	Profile    *dgtypes.Profile
	OutputDir  string
	mergeCh    chan *Goroutine
	async      sync.WaitGroup
	fails      []*Failure
	failed     map[string]bool
}

type Failure struct {
	Position     string
	FnName       string
	BasicBlockId int
	StatementId  int
}

func (f *Failure) String() string {
	return fmt.Sprintf(`{"Position":%v, "FnName":%v, "BasicBlockId":%d, "StatementId":%d}`,
		strconv.Quote(f.Position), strconv.Quote(f.FnName), f.BasicBlockId, f.StatementId)
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
		Profile:   dgtypes.NewProfile(),
		OutputDir: outputDir,
		mergeCh:   make(chan *Goroutine, 15),
		failed:    make(map[string]bool),
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

func (e *Execution) Fail(fnName string, bbid, sid int, pos string) {
	e.m.Lock()
	if !e.failed[pos] {
		e.failed[pos] = true
		e.fails = append(e.fails, &Failure{
			FnName:       fnName,
			BasicBlockId: bbid,
			StatementId:  sid,
			Position:     pos,
		})
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
	e.mergeCh <- g
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
	for be, dur := range g.Durations {
		e.Profile.Durations[be] += dur
	}
	for funcName, instances := range g.Inputs {
		e.Profile.Inputs[funcName] = append(e.Profile.Inputs[funcName], instances...)
	}
	for funcName, instances := range g.Outputs {
		e.Profile.Outputs[funcName] = append(e.Profile.Outputs[funcName], instances...)
	}
	for typeName, typ := range g.Types {
		e.Profile.Types[typeName] = typ
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
		fnPath := pjoin(e.OutputDir, "functions.json")
		fmt.Println("writing functions to:", fnPath)
		fn, err := os.Create(fnPath)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		defer fn.Close()
		err = e.Profile.WriteFunctions(fn)
		if err != nil {
			panic(err)
		}

		dotPath := pjoin(e.OutputDir, "flow-graph.dot")
		fmt.Println("writing flow-graph to:", dotPath)
		dot, err := os.Create(dotPath)
		if err != nil {
			panic(err)
		}
		defer dot.Close()
		e.Profile.WriteDotty(dot)

		txtPath := pjoin(e.OutputDir, "flow-graph.txt")
		fmt.Println("writing flow-graph to:", txtPath)
		txt, err := os.Create(txtPath)
		if err != nil {
			panic(err)
		}
		defer txt.Close()
		e.Profile.WriteSimple(txt)
	}

	if len(e.Profile.Inputs) > 0 {
		files := []string{"object-profiles.json"}
		writeOut(e, files[0], e.Profile.SerializeProfs)
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

func writeOut(e *Execution, filename string, serializeFunc func(io.Writer)) {
	filePath := pjoin(e.OutputDir, filename)
	fmt.Println("writing to:", filePath)
	fout, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer fout.Close()
	serializeFunc(fout)
}
