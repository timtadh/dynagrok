// This file defines methods which will be called by the instrumented program

package dgruntime

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
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

func ReportFailBool(pos string) bool {
	execCheck()
	exec.Fail(pos)
	return true
}

func ReportFailInt(pos string) int {
	execCheck()
	exec.Fail(pos)
	return 0
}

func ReportFailFloat(pos string) float64 {
	execCheck()
	exec.Fail(pos)
	return 0
}

// EnterBlk denotes an entry to a syntactic block (that's { ... })
func EnterBlk(bid int, pos string) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	g.m.Lock()
	defer g.m.Unlock()
	fc := g.Stack[len(g.Stack)-1]
	last := fc.Last
	cur := BlkEntrance{In: fc.FuncPc, BasicBlockId: bid}
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
		Name:   name,
		FuncPc: fpc,
		Last:   cur,
	})
	g.Flows[FlowEdge{Src: g.Stack[len(g.Stack)-2].Last, Targ: cur}]++
	g.Calls[Call{Caller: g.Stack[len(g.Stack)-2].FuncPc, Callee: fpc}]++
	g.Positions[cur] = pos
	// g.m.Unlock()
}

// deriveFields is a helper method which takes an object reference and ObjectType and
// uses reflection to determine the fields of the object. It returns the results
// in a slice.
func deriveFields(t *ObjectType, obj *interface{}) []Value {
	var v reflect.Value
	if (*t).Pointer {
		v = reflect.ValueOf(*obj).Elem()
	} else {
		v = reflect.ValueOf(*obj)
	}

	fields := make([]Value, 0)
	typeOfObj := v.Type()
	if typeOfObj.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			name := typeOfObj.Field(i).Name
			fieldType := typeOfObj.Field(i).Type
			if f.CanInterface() { // f.Interface() will fail otherwise
				ft := *getType(f.Interface())
				if f.Kind() == reflect.Slice {
					if f.CanAddr() {
						fields = append(fields, Value{Exported: true, Name: name, Type: ft, Slice: f.UnsafeAddr()})
					} else {
						fields = append(fields, Value{Exported: true, Name: name, Type: ft, Slice: 1})
					}
				} else if f.Kind() == reflect.Ptr {
					if f.CanAddr() {
						fields = append(fields, Value{Exported: true, Name: name, Type: ft, Pointer: f.UnsafeAddr()})
					} else {
						fields = append(fields, Value{Exported: true, Name: name, Type: ft, Pointer: 1})

					}
				} else if f.Kind() == reflect.Struct {
					if f.CanAddr() {
						fields = append(fields, Value{Exported: true, Name: name, Type: ft, Struct: newShallowInstance(ft, f.UnsafeAddr())})
					} else {
						fields = append(fields, Value{Exported: true, Name: name, Type: ft, Struct: newShallowInstance(ft, 1)})
					}
				} else {
					fields = append(fields, Value{Exported: true, Name: name, Type: ft, Other: f.Interface()})
				}
			} else {
				fields = append(fields, Value{Exported: false, Name: name, Type: ObjectType{fieldType.Name(), false}})
				log.Printf("Could not access unexported %v field: %v", f.Kind(), name)
			}
		}
	}
	return fields
}

// getType uses reflection to find the type of the object at 'obj'.
func getType(obj interface{}) *ObjectType {
	// uses reflection to determine the typename
	value := reflect.ValueOf(obj)
	zero := reflect.Value{}
	if value == zero {
		return &ObjectType{}
	}
	tipe := value.Type()
	if tipe.Kind() == reflect.Ptr {
		if value.Elem() == zero {
			return &ObjectType{}
		}
		return &ObjectType{value.Elem().Type().Name(), true}
	} else if tipe.Kind() == reflect.Struct {
		return &ObjectType{reflect.TypeOf(obj).Name(), false}
	}
	return &ObjectType{value.Type().String(), false}
}

//// Takes a snapshot of the object state
//// TODO replace reference to exec.Profile with reference to Goroutine
//func (o *Instance) recordInput(pos string, g Goroutine) {
//	exec.Profile.Inputs[pos] = append(exec.Profile.Inputs[pos], *o)
//	if len(exec.Profile.Inputs[pos]) > 50 {
//		exec.Profile.Inputs[pos] = exec.Profile.Inputs[pos][1:]
//	}
//}
//
//// Takes a snapshot of the object state
//// TODO replace reference to exec.Profile with reference to Goroutine
//func (o *Instance) recordOutput(pos string, g Goroutine) {
//	exec.Profile.Outputs[pos] = append(exec.Profile.Outputs[pos], *o)
//	if len(exec.Profile.Outputs[pos]) > 50 {
//		exec.Profile.Outputs[pos] = exec.Profile.Outputs[pos][1:]
//	}
//}

func MethodInput(fnName string, pos string, inputs ...interface{}) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	var inputProfile []StructT = make([]StructT, len(inputs))
	for i := range inputs {
		o := *(*[2]uintptr)(unsafe.Pointer(&inputs[i]))
		data_ptr := o[1]

		t := getType(inputs[i])
		fields := deriveFields(t, &inputs[i])
		inst := StructT{*t, fields, data_ptr}
		inputProfile[i] = inst
	}
	g.Inputs[fnName] = append(g.Inputs[fnName], inputProfile)
}

func MethodOutput(fnName string, pos string, outputs ...interface{}) {
	execCheck()
	g := exec.Goroutine(runtime.GoID())
	var outputProfile []StructT = make([]StructT, len(outputs))
	for i := range outputs {
		o := *(*[2]uintptr)(unsafe.Pointer(&outputs[i]))
		data_ptr := o[1]

		t := getType(outputs[i])
		fields := deriveFields(t, &outputs[i])
		inst := StructT{*t, fields, data_ptr}
		outputProfile[i] = inst
	}
	g.Outputs[fnName] = append(g.Inputs[fnName], outputProfile)
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
