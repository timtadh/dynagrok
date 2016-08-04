package dgruntime

import (
	"sync"
)

type Goroutine struct {
	m         sync.Mutex
	GoID      int64
	Closed    bool
	Stack     []*FuncCall
	Calls     map[Call]int
	Funcs     map[string]*Function
	CallCount int
}

type Call struct {
	Caller string
	Callee string
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

func (g *Goroutine) Exit() {
	g.m.Lock()
	defer g.m.Unlock()
	g.Closed = true
	exec.Merge(g)
}


