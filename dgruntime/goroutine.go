package dgruntime

import (
	"dgruntime/dgtypes"
	"sync"
	"time"
)

type Goroutine struct {
	m         sync.Mutex
	GoID      int64
	Closed    bool
	Inputs    map[string][]dgtypes.ObjectProfile
	Outputs   map[string][]dgtypes.ObjectProfile
	Types     map[string]dgtypes.Type
	Stack     []*dgtypes.FuncCall
	Calls     map[dgtypes.Call]int
	Flows     map[dgtypes.FlowEdge]int
	Funcs     map[uintptr]*dgtypes.Function
	Positions map[dgtypes.BlkEntrance]string
	Durations map[dgtypes.BlkEntrance]time.Duration
	CallCount int
}

func newGoroutine(id int64) *Goroutine {
	g := &Goroutine{
		Inputs:    make(map[string][]dgtypes.ObjectProfile),
		Outputs:   make(map[string][]dgtypes.ObjectProfile),
		Types:     make(map[string]dgtypes.Type),
		GoID:      id,
		Stack:     make([]*dgtypes.FuncCall, 0, 10),
		Calls:     make(map[dgtypes.Call]int),
		Funcs:     make(map[uintptr]*dgtypes.Function),
		Flows:     make(map[dgtypes.FlowEdge]int),
		Positions: make(map[dgtypes.BlkEntrance]string),
		Durations: make(map[dgtypes.BlkEntrance]time.Duration),
	}
	g.Stack = append(g.Stack, &dgtypes.FuncCall{
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
