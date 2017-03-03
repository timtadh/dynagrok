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
	Flows     map[FlowEdge]int
	Funcs     map[uintptr]*Function
	Inputs    map[string][]ObjectProfile
	Outputs   map[string][]ObjectProfile
	Positions map[BlkEntrance]string
	CallCount int
}

type Call struct {
	Caller uintptr
	Callee uintptr
}

func newGoroutine(id int64) *Goroutine {
	g := &Goroutine{
		GoID:      id,
		Stack:     make([]*FuncCall, 0, 10),
		Calls:     make(map[Call]int),
		Funcs:     make(map[uintptr]*Function),
		Flows:     make(map[FlowEdge]int),
		Inputs:    make(map[string][]ObjectProfile),
		Outputs:   make(map[string][]ObjectProfile),
		Positions: make(map[BlkEntrance]string),
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
	exec.m.Lock()
	exec.Goroutines[g.GoID] = nil
	exec.m.Unlock()
	exec.Merge(g)
	// Println(fmt.Sprintf("exit goroutine %d", g.GoID))
}
