package dgruntime


import (
	"fmt"
	"runtime"
	"unsafe"
)

var excludedPackages = map[string]bool{
	"fmt": true,
	"runtime": true,
	"sync": true,
	"strconv": true,
	"io": true,
	"os": true,
	"unsafe": true,
}

const MAXFLOW = 10

func ExcludedPkg(pkg string) bool {
	return excludedPackages[pkg]
}

func Shutdown() {
	execCheck()
	shutdown(exec)
}

func EnterBlk(bid int) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.m.Lock()
	defer g.m.Unlock()
	// fc := g.Stack[len(g.Stack)-1]
	// if len(fc.Flow) >= MAXFLOW {
	// 	return
	// }
	// fc.Flow = append(fc.Flow, BlkEntrance{bid, 0})
}

func Re_enterBlk(bid, at int) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.m.Lock()
	defer g.m.Unlock()
	//fc := g.Stack[len(g.Stack)-1]
	//if len(fc.Flow) >= MAXFLOW {
	//	return
	//}
	//fc.Flow = append(fc.Flow, BlkEntrance{bid, at})
}

func EnterFunc(name string) {
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
	g.Stack = append(g.Stack, &FuncCall{
		Name: name,
		FuncPc: fpc,
	})
	g.Calls[Call{Caller: g.Stack[len(g.Stack)-2].FuncPc, Callee: fpc}]++
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

