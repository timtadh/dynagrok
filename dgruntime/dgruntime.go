package dgruntime

import (
	"os"
	"syscall"
	"os/signal"
	"fmt"
	"runtime"
	"unsafe"
)

func init() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig:=<-sigs
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
	cur := BlkEntrance{In: fc.FuncPc, BasicBlockId: bbid}
	g.Flows[FlowEdge{Src: last, Targ: cur}]++
	fc.Last = cur
	g.Positions[cur] = pos
}

func EnterFunc(name, pos string) {
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
	cur := BlkEntrance{In: fpc, BasicBlockId: 0}
	g.Stack = append(g.Stack, &FuncCall{
		Name: name,
		FuncPc: fpc,
		Last: cur,
	})
	g.Flows[FlowEdge{Src: g.Stack[len(g.Stack)-2].Last, Targ: cur}]++
	g.Calls[Call{Caller: g.Stack[len(g.Stack)-2].FuncPc, Callee: fpc}]++
	g.Positions[cur] = pos
	// g.m.Unlock()
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
		g.Flows[FlowEdge{Src: fc.Last, Targ: g.Stack[len(g.Stack)-1].Last}]++
	}
	if f, has := g.Funcs[fc.FuncPc]; has {
		f.Update(fc)
	} else {
		g.Funcs[fc.FuncPc] = newFunction(fc)
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

