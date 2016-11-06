package dgruntime

import "fmt"

type ObjectType struct {
	Name    string
	Pointer bool
	Fields  []ObjectType
	Methods []string
}

type Instance struct {
	Interface ObjectType
	Fields    map[string]interface{}
	History   []string
	Reference uintptr
}

func newObjectType(n string, isPtr bool, f []ObjectType) *ObjectType {
	if n == "" {
		return &ObjectType{Name: "Unnamed Type", Pointer: isPtr, Fields: f}
	}
	return &ObjectType{Name: n, Pointer: isPtr, Fields: f}
}

func newInstance(tipe ObjectType, initValues map[string]interface{}, data_ptr uintptr) *Instance {
	return &Instance{Interface: tipe, Fields: initValues, Reference: data_ptr, History: make([]string, 3)}
}

func (o *Instance) addCall(method string) {
	o.History = append(o.History, method)
}

func (o *Instance) snap(pos string) {
	println("snapping")
	exec.Profile.Instances[pos] = o
}

func (o *Instance) String() string {
	return fmt.Sprintf("{ Reference: %d\n Fields: %v\n Method Sequence: %s\n} %s", o.Reference, o.Fields, o.History, o.Interface.Name)
}
