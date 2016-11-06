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

func InstanceDecl(name string, ptr *interface{}) {
	execCheck()
	// g := exec.Goroutine(runtime.GoID())
	/*
		_ = name
		fields := deriveFields(ptr)
		t := getType(name, ptr)
		o := newInstance(*t, fields, ptr)
		g.Instances[ptr] = o
			if t, has := g.Types[name]; has {
				o := newInstance(t, initVals)
				g.Instances[ptr] = o
			} else {
				panic("Type Undeclared")
			}
	*/
}

// deriveFields takes an object reference and returns
// a mapping of field typeNames to the corresponding ?concretization?
/*func deriveFields(ptr *interface{}) map[string]interface{} {
	s := reflect.ValueOf(ptr).Elem()
	fields := *new(map[string]interface{})
	typeOfT := s.Type()
	str, ok := (*ptr).(types.Struct)
	if ok == true {
		for i := 0; i < s.NumField(); i++ {
			f := s.Field(i)
			if f.CanSet() {
				fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
				fields[f.Type().Name()] = f.Interface()
			}

		}
	}
	fmt.Printf("Object: %s, Kind: %s, %t, %v", typeOfT, s.Kind().String(), ok, str)
	return fields
}
*/
func deriveFields(t *ObjectType, obj *interface{}) map[string]interface{} {
	var v reflect.Value
	if (*t).Pointer {
		v = reflect.ValueOf(*obj).Elem()
	} else {
		v = reflect.ValueOf(*obj)
	}

	fields := make(map[string]interface{})
	typeOfObj := v.Type()
	if typeOfObj.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.CanSet() {
				fields[typeOfObj.Field(i).Name] = f.Interface()
			}
		}
	}
	return fields
}

// getType gets the type of the object at 'obj'
func getType(obj interface{}) *ObjectType {
	// uses reflection to determine the typename
	tipe := reflect.ValueOf(obj).Type()
	if tipe.Kind() == reflect.Ptr {
		return newObjectType("*"+reflect.ValueOf(obj).Elem().Type().Name(), true, nil)
	}
	return newObjectType(reflect.ValueOf(obj).Type().Name(), false, nil)
}

// MethodCall takes the name of the call, the position,
// and a pointer to the object-receiver. It adds this call
// to the method-call sequence for this particular receiver,
// and uses the opportunity to take a snapshot of object state
func MethodCall(field string, tipe string, pos string, obj interface{}) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	o := *(*[2]uintptr)(unsafe.Pointer(&obj))
	data_ptr := o[1]
	if instance, has := g.Instances[data_ptr]; has {
		instance.addCall(field)
		instance.snap(pos)
	} else {
		t := getType(obj)
		fields := deriveFields(t, &obj)
		o := newInstance(*t, fields, data_ptr)
		g.Instances[data_ptr] = o
		o.addCall(field)
		o.snap(pos)
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
