package dgruntime


import (
	"fmt"
	"runtime"
	"sync"
	"io"
	"os"
	"strconv"
)

var excludedPackages = map[string]bool{
	"fmt": true,
	"runtime": true,
	"sync": true,
	"strconv": true,
	"io": true,
	"os": true,
}

func ExcludedPkg(pkg string) bool {
	return excludedPackages[pkg]
}

var execMu sync.Mutex
var exec *Execution

type Execution struct {
	m sync.Mutex
	Goroutines []*Goroutine
	OutputPath string
}

type Goroutine struct {
	m sync.Mutex
	GoID   int64
	Closed bool
	Stack  []*FuncCall
	Calls  map[Call]int
	Funcs  map[string]*Function
}

type Function struct {
	Name string
	RuntimeNames []string
	FuncPcs []uintptr
	CallPcs []uintptr
	Calls int
}

type FuncCall struct {
	Name, RuntimeName string
	FuncPc, CallPc uintptr
}

type Call struct {
	Caller string
	Callee string
}

func newExecution() *Execution {
	output := "/tmp/dynagrok-profile.dot"
	if os.Getenv("DGPROF") != "" {
		output = os.Getenv("DGPROF")
	}
	e := &Execution{
		OutputPath: output,
	}
	e.growGoroutines()
	return e
}

func newGoroutine(id int64) *Goroutine {
	g := &Goroutine{
		GoID: id,
		Stack: make([]*FuncCall, 0, 10),
		Calls: make(map[Call]int),
		Funcs: make(map[string]*Function),
	}
	g.Stack = append(g.Stack, &FuncCall{
		Name: "<entry>",
	})
	return g
}

func newFunction(fc *FuncCall) *Function {
	f := &Function {
		Name: fc.Name,
		RuntimeNames: make([]string, 0, 10),
		FuncPcs: make([]uintptr, 0, 10),
		CallPcs: make([]uintptr, 0, 10),
	}
	f.Update(fc)
	return f
}

func (e *Execution) Goroutine(id int64) *Goroutine {
	for id >= int64(len(e.Goroutines)) {
		e.m.Lock()
		for id >= int64(len(e.Goroutines)) {
			e.growGoroutines()
		}
		e.m.Unlock()
	}
	return e.Goroutines[id]
}

func (e *Execution) growGoroutines() {
	n := make([]*Goroutine, (len(e.Goroutines)+1)*2)
	copy(n, e.Goroutines)
	for i := len(e.Goroutines); i < len(n); i++ {
		n[i] = newGoroutine(int64(i))
	}
	e.Goroutines = n
}

func (g *Goroutine) Exit() {
	g.m.Lock()
	g.Closed = true
	g.m.Unlock()
}

func (g *Goroutine) Serialize(fout io.Writer) {
	g.m.Lock()
	defer g.m.Unlock()
	nextfid := 1
	fids := make(map[string]int)
	fids["<entry>"] = 0
	fmt.Fprintf(fout, "digraph \"g-%d\" {\n", g.GoID)
	fmt.Fprintf(fout, "0 [label=\"goroutine %d\", shape=rect];\n", g.GoID)
	for _, f := range g.Funcs {
		fid := nextfid
		nextfid++
		fids[f.Name] = fid
		fmt.Fprintf(fout, "%d [label=%v, shape=rect, calls=%d, runtime_name=%v, entry_pc=%d];\n",
			fid, strconv.Quote(f.Name), f.Calls, strconv.Quote(f.RuntimeNames[0]), f.FuncPcs[0])
	}
	for call, count := range g.Calls {
		fmt.Fprintf(fout, "%v -> %v [calls=%d];\n",
			fids[call.Caller], fids[call.Callee], count)
	}
	fmt.Fprintln(fout, "}\n\n")
}

func (f *Function) Update(fc *FuncCall) {
	f.Calls++
	hasName := false
	for _, name := range f.RuntimeNames {
		if name == fc.RuntimeName {
			hasName = true
			break
		}
	}
	if !hasName {
		f.RuntimeNames = append(f.RuntimeNames, fc.RuntimeName)
	}
	hasCallPc := false
	for _, pc := range f.CallPcs {
		if pc == fc.CallPc {
			hasCallPc = true
			break
		}
	}
	if !hasCallPc {
		f.CallPcs = append(f.CallPcs, fc.CallPc)
	}
	hasFuncPc := false
	for _, pc := range f.FuncPcs {
		if pc == fc.FuncPc {
			hasFuncPc = true
			break
		}
	}
	if !hasFuncPc {
		f.FuncPcs = append(f.FuncPcs, fc.FuncPc)
	}
}

func execCheck() {
	if exec == nil {
		execMu.Lock()
		for exec == nil {
			exec = newExecution()
		}
		runtime.SetFinalizer(exec, shutdown)
		execMu.Unlock()
	}
}

func Shutdown() {
	execCheck()
	shutdown(exec)
}

func shutdown(e *Execution) {
	execMu.Lock()
	defer execMu.Unlock()
	if e == nil {
		return
	}
	e.m.Lock()
	defer e.m.Unlock()
	fout, err := os.Create(e.OutputPath)
	if err != nil {
		panic(err)
	}
	defer fout.Close()
	for _, g := range e.Goroutines {
		if !g.Closed && len(g.Calls) > 0 {
			g.Exit()
		}
	}
	for _, g := range e.Goroutines {
		if g.Closed && len(g.Calls) > 0 {
			g.Serialize(fout)
		}
	}
	fmt.Println("done shutting down")
}

func EnterFunc(name string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.m.Lock()
	defer g.m.Unlock()
	if g.Closed {
		panic("enter func on closed Goroutine")
	}
	var callers [2]uintptr
	n := runtime.Callers(2, callers[:])
	if n <= 0 {
		panic("could not get stack frame")
	}
	f := runtime.FuncForPC(callers[0])
	g.Stack = append(g.Stack, &FuncCall{
		Name: name,
		RuntimeName: f.Name(),
		CallPc: callers[0]-1,
		FuncPc: f.Entry(),
	})
	g.Calls[Call{Caller: g.Stack[len(g.Stack)-2].Name, Callee: g.Stack[len(g.Stack)-1].Name}]++
	// Println(fmt.Sprintf("enter-func: %v %d", name, g.GoID))
}

func ExitFunc(name string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.m.Lock()
	defer g.m.Unlock()
	if g.Closed {
		panic("enter func on closed Goroutine")
	}
	fc := g.Stack[len(g.Stack)-1]
	g.Stack = g.Stack[:len(g.Stack)-1]
	if f, has := g.Funcs[fc.Name]; has {
		f.Update(fc)
	} else {
		g.Funcs[fc.Name] = newFunction(fc)
	}
}

func Println(data string) {
	execCheck()
	exec.m.Lock()
	defer exec.m.Unlock()
	fmt.Printf("goid %v:\t %v\n", runtime.GoID(), data)
}

