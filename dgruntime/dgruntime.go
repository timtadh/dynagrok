package dgruntime


import (
	"fmt"
	"runtime"
	"sync"
)

var execMu sync.Mutex
var exec *Execution

type Execution struct {
	m sync.Mutex
	Goroutines []*Goroutine
}

type Goroutine struct {
	GoID   int64
	Closed bool
	Stack  []*Func
	Calls  map[Call]int
}

type Func struct {
	Name string
}

type Call struct {
	Caller string
	Callee string
}

func newExecution() *Execution {
	e := &Execution{}
	e.growGoroutines()
	return e
}

func newGoroutine(id int64) *Goroutine {
	g := &Goroutine{
		GoID: id,
		Stack: make([]*Func, 0, 10),
		Calls: make(map[Call]int),
	}
	g.Stack = append(g.Stack, &Func{
		Name: "<entry>",
	})
	return g
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
	Println(fmt.Sprintf("exit-goroutine %v: %v", g.GoID, g.Calls))
	g.Closed = true
	if g.GoID == 1 {
		shutdown(exec)
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
	exec.Goroutine(1).Exit()
}

func shutdown(e *Execution) {
	execMu.Lock()
	defer execMu.Unlock()
	if e == nil {
		return
	}
	for _, g := range e.Goroutines {
		if !g.Closed && len(g.Calls) > 0 {
			g.Exit()
		}
	}
}

func EnterFunc(name string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.Stack = append(g.Stack, &Func{
		Name: name,
	})
	g.Calls[Call{Caller: g.Stack[len(g.Stack)-2].Name, Callee: g.Stack[len(g.Stack)-1].Name}]++
	// Println(fmt.Sprintf("enter-func: %v %d", name, g.GoID))
}

func ExitFunc(name string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.Stack = g.Stack[:len(g.Stack)-1]
	// Println(fmt.Sprintf("exit-func: %v %d", name, g.GoID))
	if len(g.Stack) <= 1 {
		// g.Exit()
	}
}

func Println(data string) {
	execCheck()
	exec.m.Lock()
	defer exec.m.Unlock()
	fmt.Printf("goid %v:\t %v\n", runtime.GoID(), data)
}

