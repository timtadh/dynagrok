package dgruntime

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

var excludedPackages = map[string]bool{
	"fmt":     true,
	"runtime": true,
	"sync":    true,
	"strconv": true,
	"io":      true,
	"os":      true,
	"unsafe":  true,
}

const MAXFLOW = 10

func ExcludedPkg(pkg string) bool {
	return excludedPackages[pkg]
}

func Shutdown() {
	execCheck()
	shutdown(exec)
}

func EnterBlk(bid int, pos string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.m.Lock()
	defer g.m.Unlock()
	fc := g.Stack[len(g.Stack)-1]
	last := fc.Last
	cur := BlkEntrance{In: fc.FuncPc, BlkId: bid, At: 0}
	g.Flows[FlowEdge{Src: last, Targ: cur}]++
	fc.Last = cur
	g.Positions[cur] = pos
}

func Re_enterBlk(bid, at int, pos string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.m.Lock()
	defer g.m.Unlock()
	fc := g.Stack[len(g.Stack)-1]
	last := fc.Last
	cur := BlkEntrance{In: fc.FuncPc, BlkId: bid, At: at}
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
	cur := BlkEntrance{In: fpc, BlkId: 0, At: 0}
	g.Stack = append(g.Stack, &FuncCall{
		Name:   name,
		FuncPc: fpc,
		Last:   cur,
	})
	g.Flows[FlowEdge{Src: g.Stack[len(g.Stack)-2].Last, Targ: cur}]++
	g.Calls[Call{Caller: g.Stack[len(g.Stack)-2].FuncPc, Callee: fpc}]++
	g.Positions[cur] = pos
	// g.m.Unlock()
}

// StructDecl may not be usable because you can't insert
// instrumentation code within a struct declaration
func StructDecl(name string, fieldTypes []string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	// TODO: Lookup objecttpes from fieldType strings
	t := newObjectType(name, nil)
	g.Types[name] = *t
	for _, field := range fieldTypes {
		if ft, has := g.Types[field]; has {
			t.Fields = append(t.Fields, ft)
		}
	}
}

func getType(name string, ptr uintptr) *ObjectType {
	return newObjectType(reflect.ValueOf(ptr).Elem().Type().Name(), nil)
}

func InstanceDecl(name string, ptr uintptr) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	_ = name
	fields := deriveFields(ptr)
	t := getType(name, ptr)
	o := newInstance(*t, fields, ptr)
	g.Instances[ptr] = o
	/*
		if t, has := g.Types[name]; has {
			o := newInstance(t, initVals)
			g.Instances[ptr] = o
		} else {
			panic("Type Undeclared")
		}
	*/
}

func deriveFields(ptr uintptr) map[string]interface{} {
	s := reflect.ValueOf(ptr).Elem()
	fields := *new(map[string]interface{})
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
		fields[f.Type().Name()] = f.Interface()
	}
	return fields
}

func MethodCall(field string, pos string, ptr uintptr) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	if instance, has := g.Instances[ptr]; has {
		instance.addCall(field)
		instance.snap(pos)
	} else {
		panic("Undeclared object")
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
