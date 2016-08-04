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
	Profile    *Profile
	OutputPath string
}

type Profile struct {
	Funcs     map[string]*Function
	Calls     map[Call]int
	CallCount int
}

type Goroutine struct {
	m         sync.Mutex
	GoID      int64
	Closed    bool
	Stack     []*FuncCall
	Calls     map[Call]int
	Funcs     map[string]*Function
	CallCount int
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
		Profile: &Profile{
			Calls: make(map[Call]int),
			Funcs: make(map[string]*Function),
		},
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

func (e *Execution) Merge(g *Goroutine) {
	e.m.Lock()
	defer e.m.Unlock()
	if !g.Closed {
		return
	}
	e.Profile.CallCount += g.CallCount
	for _, fn := range g.Funcs {
		if x, has := e.Profile.Funcs[fn.Name]; has {
			x.Merge(fn)
		} else {
			e.Profile.Funcs[fn.Name] = fn
		}
	}
	for call, count := range g.Calls {
		e.Profile.Calls[call] += count
	}
}

func (g *Goroutine) Exit() {
	g.m.Lock()
	defer g.m.Unlock()
	g.Closed = true
	exec.Merge(g)
}

func (p *Profile) Serialize(fout io.Writer) {
	strlist := func(list []string) string {
		str := "["
		for i, item := range list {
			str += item
			if i+1 < len(list) {
				str += ", "
			}
		}
		str += "]"
		return strconv.Quote(str)
	}
	intlist := func(list []uintptr) string {
		items := make([]string, 0, len(list))
		for _, i := range list {
			items = append(items, fmt.Sprintf("%v", i))
		}
		return strlist(items)
	}
	max := func(a, b float64) float64 {
		if a > b {
			return a
		}
		return b
	}
	round := func(a float64) int {
		return int(a + .5)
	}
	nextfid := 1
	fids := make(map[string]int)
	fids["<entry>"] = 0
	fmt.Fprintf(fout, "digraph {\n",)
	fmt.Fprintf(fout, "0 [label=\"entry\", shape=rect];\n")
	for _, f := range p.Funcs {
		fid := nextfid
		nextfid++
		fids[f.Name] = fid
		fmt.Fprintf(fout, "%d [label=%v, shape=rect, calls=%d, runtime_names=%v, entry_pcs=%v, fontsize=%d];\n",
			fid, strconv.Quote(f.Name), f.Calls, strlist(f.RuntimeNames),
			intlist(f.FuncPcs),
			round(96*max(.15, float64(f.Calls)/float64(p.CallCount))),
		)
	}
	for call, count := range p.Calls {
		fmt.Fprintf(fout, "%v -> %v [calls=%d, weight=%f];\n",
			fids[call.Caller], fids[call.Callee],
			count, float64(count)/float64(p.CallCount))
	}
	fmt.Fprintln(fout, "}\n\n")
}

func (f *Function) Merge(b *Function) {
	f.Calls += b.Calls
	for _, bName := range b.RuntimeNames {
		hasName := false
		for _, name := range f.RuntimeNames {
			if name == bName {
				hasName = true
				break
			}
		}
		if !hasName {
			f.RuntimeNames = append(f.RuntimeNames, bName)
		}
	}
	for _, bCallPc := range b.CallPcs {
		hasCallPc := false
		for _, pc := range f.CallPcs {
			if pc == bCallPc {
				hasCallPc = true
				break
			}
		}
		if !hasCallPc {
			f.CallPcs = append(f.CallPcs, bCallPc)
		}
	}
	for _, bFuncPc := range b.FuncPcs {
		hasFuncPc := false
		for _, pc := range f.FuncPcs {
			if pc == bFuncPc {
				hasFuncPc = true
				break
			}
		}
		if !hasFuncPc {
			f.FuncPcs = append(f.FuncPcs, bFuncPc)
		}
	}
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
	for _, g := range e.Goroutines {
		g.m.Lock()
		if !g.Closed && len(g.Calls) > 0 {
			g.m.Unlock()
			g.Exit()
		}
	}
	e.m.Lock()
	defer e.m.Unlock()
	fout, err := os.Create(e.OutputPath)
	if err != nil {
		panic(err)
	}
	e.Profile.Serialize(fout)
	fout.Close()
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
	g.CallCount++
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

