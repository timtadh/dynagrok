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
	Fields    []Field
	History   []string
	Reference uintptr
}

type Field struct {
	Type    ObjectType
	Struct  *Instance
	Pointer uintptr
	Slice   uintptr
	Other   interface{}
}

func newObjectType(n string, isPtr bool, f []ObjectType) *ObjectType {
	if n == "" {
		return &ObjectType{Name: "Unnamed Type", Pointer: isPtr, Fields: f}
	}
	return &ObjectType{Name: n, Pointer: isPtr, Fields: f}
}

func newInstance(tipe ObjectType, initValues []Field, data_ptr uintptr) *Instance {
	return &Instance{Interface: tipe, Fields: initValues, Reference: data_ptr, History: make([]string, 0)}
}

func newShallowInstance(tipe ObjectType, data_ptr uintptr) *Instance {
	return newInstance(tipe, nil, data_ptr)
}

func (o *Instance) addCall(method string) {
	o.History = append(o.History, method)
}

func (o *Instance) snap(pos string) {
	exec.Profile.Instances[pos] = o
}
func (o *Instance) String() string {
	return o.Serialize(0)
}

func (o *Instance) Serialize(depth int) string {
	space := ""
	for i := 0; i < depth; i++ {
		space += "\t"
	}
	str := fmt.Sprintf("%s{ Reference: %d\n", space, o.Reference)
	//if len(o.Fields) > 0 {
	str += space + "  Fields: \n"
	for _, f := range o.Fields {
		str += fmt.Sprintf("%s%v\n", space, f.Serialize(depth+1))
	}
	//}
	if len(o.History) > 0 {
		str += fmt.Sprintf("%s  Method Sequence: %s\n", space, o.History)
	}
	return str + fmt.Sprintf("%s } %s", space, o.Interface.Name)
}

func (f *Field) Kind() Kind {
	if f.Struct != nil {
		return Struct
	}
	if f.Pointer != 0 {
		return Pointer
	}
	if f.Slice != 0 {
		return Slice
	}
	return Other
}

func (f Field) String() string {
	return f.Serialize(0)
}

func (f Field) Serialize(depth int) string {
	space := ""
	for i := 0; i < depth; i++ {
		space += "\t"
	}
	switch f.Kind() {
	case Struct:
		return fmt.Sprintf("%s", f.Struct.Serialize(depth))
	case Pointer:
		return fmt.Sprintf("%s%d", space, f.Pointer)
	case Slice:
		return fmt.Sprintf("%s[]%d", space, f.Slice)
	case Other:
		return fmt.Sprintf("%s%v", space, f.Other)
	}
	panic("Trying to print an uninitialized field")
}

type Kind uint

const (
	Pointer Kind = iota
	Struct
	Slice
	Other
)
