// This file defines methods which will be called by the instrumented program

package dgruntime

import (
	"dgruntime/dgtypes"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
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
	cur := dgtypes.BlkEntrance{In: fc.FuncPc, BasicBlockId: bbid}
	g.Flows[dgtypes.FlowEdge{Src: last, Targ: cur}]++
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
	cur := dgtypes.BlkEntrance{In: fpc, BasicBlockId: 0}
	g.Stack = append(g.Stack, &dgtypes.FuncCall{
		Name:   name,
		FuncPc: fpc,
		Last:   cur,
	})
	g.Flows[dgtypes.FlowEdge{Src: g.Stack[len(g.Stack)-2].Last, Targ: cur}]++
	g.Calls[dgtypes.Call{Caller: g.Stack[len(g.Stack)-2].FuncPc, Callee: fpc}]++
	g.Positions[cur] = pos
	// g.m.Unlock()
}

// deriveFields is a helper method which takes an object reference and ObjectType and
// uses reflection to determine the fields of the object. It returns the results
// in a slice.
//func deriveFields(v reflect.Value) []dgtypes.Field {
//	fields := make([]dgtypes.Field, 0)
//	typeOfObj := v.Type()
//
//	for i := 0; i < v.NumField(); i++ {
//		f := v.Field(i)
//		name := typeOfObj.Field(i).Name
//		fieldType := typeOfObj.Field(i).Type
//		if f.CanInterface() { // f.Interface() will fail otherwise
//			ft := *getType(f.Interface())
//			switch f.Kind() {
//			case reflect.Slice:
//				if f.CanAddr() {
//					fields = append(fields, dgtypes.Field{Name: name, Exported: true,
//						Val: dgtypes.Value{Type: ft, Slice: f.UnsafeAddr()}})
//				} else {
//					fields = append(fields, dgtypes.Field{Name: name, Exported: true,
//						Val: dgtypes.Value{Type: ft, Slice: 1}})
//				}
//			case reflect.Ptr:
//				if f.CanAddr() {
//					fields = append(fields, dgtypes.Field{Name: name, Exported: true,
//						Val: dgtypes.Value{Type: ft, Pointer: f.UnsafeAddr()}})
//				} else {
//					fields = append(fields, dgtypes.Field{Name: name, Exported: true,
//						Val: dgtypes.Value{Type: ft, Pointer: 1}})
//				}
//			case reflect.Struct:
//				if f.CanAddr() {
//					fields = append(fields, dgtypes.Field{Name: name, Exported: true, Val: dgtypes.Value{
//						Type: ft, Struct: dgtypes.NewShallowStruct(ft, f.UnsafeAddr())}})
//				} else {
//					fields = append(fields, dgtypes.Field{Name: name, Exported: true,
//						Val: dgtypes.Value{Type: ft, Struct: dgtypes.NewShallowStruct(ft, 1)}})
//				}
//			default:
//				fields = append(fields, dgtypes.Field{Name: name, Exported: true, Val: dgtypes.Value{Type: ft, Other: f.Interface()}})
//			}
//		} else {
//			fields = append(fields, dgtypes.Field{Name: name, Exported: true,
//				Val: dgtypes.Value{Type: dgtypes.ObjectType{fieldType.Name(), false}}})
//			log.Printf("Could not access unexported %v field: %v", f.Kind(), name)
//		}
//	}
//	return fields
//}
//
//// getType uses reflection to find the type of the object at 'obj'.
//func getType(obj interface{}) *dgtypes.ObjectType {
//	// uses reflection to determine the typename
//	value := reflect.ValueOf(obj)
//	zero := reflect.Value{}
//	if value == zero {
//		return &dgtypes.ObjectType{}
//	}
//	tipe := value.Type()
//	if tipe.Kind() == reflect.Ptr {
//		if value.Elem() == zero {
//			return &dgtypes.ObjectType{}
//		}
//		return &dgtypes.ObjectType{value.Elem().Type().Name(), true}
//	} else if tipe.Kind() == reflect.Struct {
//		return &dgtypes.ObjectType{reflect.TypeOf(obj).Name(), false}
//	}
//	return &dgtypes.ObjectType{value.Type().String(), false}
//}
//
//func reflectValToValue(val reflect.Value, tp dgtypes.ObjectType) *dgtypes.Value {
//	typeOfObj := val.Type()
//	switch typeOfObj.Kind() {
//	case reflect.Struct:
//		fields := deriveFields(val)
//		return &dgtypes.Value{Type: tp, Struct: &dgtypes.StructT{Type: tp, Fields: fields}}
//	}
//	return nil
//}
//
//func deriveValue(obj interface{}) *dgtypes.Value {
//	t := getType(obj)
//
//	var v reflect.Value
//	if (*t).Pointer {
//		v = reflect.ValueOf(obj).Elem()
//	} else {
//		v = reflect.ValueOf(obj)
//	}
//
//	return reflectValToValue(v, *t)
//}

func deriveProfile(items []interface{}) dgtypes.ObjectProfile {
	values := make(dgtypes.ObjectProfile, 0, len(items))
	for _, item := range items {
		values = append(values, dgtypes.NewVal(item))
	}
	return values
}

func MethodInput(fnName string, pos string, inputs ...interface{}) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.Inputs[fnName] = append(g.Inputs[fnName], deriveProfile(inputs))
}

func MethodOutput(fnName string, pos string, outputs ...interface{}) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.Outputs[fnName] = append(g.Outputs[fnName], deriveProfile(outputs))
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
		g.Flows[dgtypes.FlowEdge{Src: fc.Last, Targ: g.Stack[len(g.Stack)-1].Last}]++
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
