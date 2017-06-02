// This file defines methods which will be called by the instrumented program

package dgruntime

import (
	"dgruntime/dgtypes"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

func init() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("dynagrok got a sig", sig)
		Shutdown()
		panic(fmt.Errorf("dynagrok caught signal: %v", sig))
	}()
}

func Shutdown() {
	fmt.Println(runtime.Wacky())
	execCheck()
	shutdown(exec)
}

func ReportFailBool(fnName string, bbid int, pos string) bool {
	execCheck()
	exec.Fail(fnName, bbid, pos)
	return true
}

func ReportFailInt(fnName string, bbid int, pos string) int {
	execCheck()
	exec.Fail(fnName, bbid, pos)
	return 0
}

func ReportFailFloat(fnName string, bbid int, pos string) float64 {
	execCheck()
	exec.Fail(fnName, bbid, pos)
	return 0
}

func EnterBlkFromCond(bbid int, pos string) bool {
	EnterBlk(bbid, pos)
	return true
}

func EnterBlk(bbid int, pos string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.m.Lock()
	defer g.m.Unlock()
	fc := g.Stack[len(g.Stack)-1]
	last := fc.Last
	start := fc.LastTime
	cur := dgtypes.BlkEntrance{In: fc.FuncPc, BasicBlockId: bbid}
	fc.Last = cur
	fc.LastTime = time.Now()
	dur := fc.LastTime.Sub(start)
	g.Flows[dgtypes.FlowEdge{Src: last, Targ: cur}]++
	g.Positions[cur] = pos
	g.Durations[last] += dur
	//
	// Masri's Algorithm for dynamic control dependence
	//
	// W. Masri and A. Podgurski, “Algorithms and Tool Support for Dynamic
	// Information Flow Analysis,” Information and Software Technology. Feb.
	// 2009. https://doi.org/10.1016/j.infsof.2008.05.008
	if len(fc.CDStack) > 0 && bbid == fc.IPDom[fc.CDStack[len(fc.CDStack)-1]] {
		fc.CDStack = fc.CDStack[:len(fc.CDStack)-1] // pop the CDStack
	}
	if len(fc.CDStack) > 0 {
		// fmt.Printf("%v: dyn-cdp for %d is %d\n", fc.Name, bbid, fc.CDStack[len(fc.CDStack)-1])
		fc.DynCDP[bbid][fc.CDStack[len(fc.CDStack)-1]] = true
	} else {
		// fmt.Printf("%v: dyn-cdp for %d is null\n", fc.Name, bbid)
	}
	if len(fc.CFG[bbid]) > 1 {
		if len(fc.CDStack) > 0 && fc.IPDom[bbid] == fc.IPDom[fc.CDStack[len(fc.CDStack)-1]] {
			fc.CDStack = fc.CDStack[:len(fc.CDStack)-1] // pop the CDStack
		}
		fc.CDStack = append(fc.CDStack, bbid)
	}
}

func EnterFunc(name, pos string, cfg [][]int, ipdom []int) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	// g.m.Lock()
	if g.Closed {
		// g.m.Unlock()
		panic("enter func on closed Goroutine")
	}
	pc := runtime.GetCallerPC(unsafe.Pointer(&name))
	f := runtime.FuncForPC(pc)
	fpc := f.Entry()
	cur := dgtypes.BlkEntrance{In: fpc, BasicBlockId: 0}
	fc := &dgtypes.FuncCall{
		Name:     name,
		FuncPc:   fpc,
		Last:     cur,
		LastTime: time.Now(),
		CFG:      cfg,
		IPDom:    ipdom,
		CDStack:  append(make([]int, 0, len(ipdom)), 0),
		DynCDP:   make([]map[int]bool, len(cfg)),
	}
	g.Stack = append(g.Stack, fc)
	for i := range fc.DynCDP {
		fc.DynCDP[i] = make(map[int]bool)
	}
	g.Flows[dgtypes.FlowEdge{Src: g.Stack[len(g.Stack)-2].Last, Targ: cur}]++
	g.Calls[dgtypes.Call{Caller: g.Stack[len(g.Stack)-2].FuncPc, Callee: fpc}]++
	g.Positions[cur] = pos
	// g.m.Unlock()
}

func deriveProfile(items []interface{}) (dgtypes.ObjectProfile, []dgtypes.Type) {
	// profiles will be delivered as a struct {name string, val interface{}}

	values := make(dgtypes.ObjectProfile, 0, len(items))
	types := make([]dgtypes.Type, 0, len(items))
	for _, item := range items {
		if param, ok := item.(struct {
			Name string
			Val  interface{}
		}); ok {
			values = append(values, dgtypes.Param{Name: param.Name, Val: dgtypes.NewVal(param.Val)})
			types = append(types, dgtypes.NewType(param.Val))
		}
	}
	return values, types
}

func MethodInput(fnName string, pos string, inputs ...interface{}) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	values, types := deriveProfile(inputs)
	g.Inputs[fnName] = append(g.Inputs[fnName], values)
	for _, typ := range types {
		g.Types[typ.Name()] = typ
	}
}

func MethodOutput(fnName string, pos string, outputs ...interface{}) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	values, types := deriveProfile(outputs)
	g.Outputs[fnName] = append(g.Outputs[fnName], values)
	for _, typ := range types {
		g.Types[typ.Name()] = typ
	}
}

func ExitFunc(name string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	// g.m.Lock()
	if g.Closed {
		// g.m.Unlock()
		panic("enter func on closed Goroutine")
	}
	g.CallCount++
	fc := g.Stack[len(g.Stack)-1]
	g.Stack = g.Stack[:len(g.Stack)-1]
	// Println(fmt.Sprintf("exit %v %v", fc.Name, fc.Flow))
	if len(g.Stack) >= 1 {
		ret := g.Stack[len(g.Stack)-1]
		start := fc.LastTime
		now := time.Now()
		g.Flows[dgtypes.FlowEdge{Src: fc.Last, Targ: ret.Last}]++
		g.Durations[fc.Last] += now.Sub(start)
		ret.LastTime = now
	}
	if f, has := g.Funcs[fc.FuncPc]; has {
		f.Update(fc)
	} else {
		g.Funcs[fc.FuncPc] = dgtypes.NewFunction(fc)
	}
	if len(g.Stack) == 1 {
		// g.m.Unlock()
		g.Exit()
		return
	}
	// g.m.Unlock()
}

func Println(data string) {
	execCheck()
	exec.m.Lock()
	defer exec.m.Unlock()
	fmt.Printf("goid %v:\t %v\n", runtime.GoID(), data)
}
