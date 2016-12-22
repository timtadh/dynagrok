package dgruntime

import (
	"fmt"
)

type ObjectType struct {
	Name    string
	Pointer bool
}

type Instance struct {
	Interface ObjectType
	Fields    []Field
	History   []string
	Reference uintptr
}

type Field struct {
	Name     string
	Type     ObjectType
	Exported bool
	Struct   *Instance
	Pointer  uintptr
	Slice    uintptr
	Map      bool
	Func     bool
	// TODO distinguish types that are 'uncomparable': slices, maps, and
	// functions
	Other interface{}
}

func newObjectType(n string, isPtr bool) *ObjectType {
	if n == "" {
		return &ObjectType{Name: "Unnamed Type", Pointer: isPtr}
	}
	return &ObjectType{Name: n, Pointer: isPtr}
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
	exec.Profile.Instances[pos] = append(exec.Profile.Instances[pos], *o)
	if len(exec.Profile.Instances[pos]) > 50 {
		exec.Profile.Instances[pos] = exec.Profile.Instances[pos][1:]
	}
}
func (o *Instance) String() string {
	return o.Serialize("")
}

func (o *Instance) PrettyString() string {
	return o.PrettySerialize(0)
}

func (o *Instance) getExportedFields() []Field {
	var exported []Field = make([]Field, 0)
	for _, f := range o.Fields {
		if f.Exported {
			exported = append(exported, f)
		}
	}
	return exported
}

func (o *Instance) Serialize(pos string) string {
	obj := NewObject(o.Interface.Name, 0, pos, o.getExportedFields(), o.History)
	return SerializeObject(obj)
}

func (o *Instance) PrettySerialize(depth int) string {
	space := ""
	for i := 0; i < depth; i++ {
		space += "\t"
	}
	str := fmt.Sprintf("%s%s: { Reference: %d\n", space, o.Interface.Name, o.Reference)
	str += space + "  Fields: \n"
	for _, f := range o.Fields {
		str += fmt.Sprintf("%s%v\n", space, f.PrettySerialize(depth+1))
	}
	if len(o.Fields) == 0 {
		str += space + "\t<no fields>" + "\n"
	}
	if len(o.History) > 0 {
		str += fmt.Sprintf("%s  Method Sequence: %s\n", space, o.History)
	}
	return str + fmt.Sprintf("%s }", space)
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
	if f.Map != false {
		return Map
	}
	if f.Func != false {
		return Func
	}
	return Other
}

func (f Field) CompactString() string {
	switch f.Kind() {
	case Struct:
		return fmt.Sprintf("{%s: %s}", f.Name, f.Type.Name)
	case Pointer:
		return fmt.Sprintf("{%s: %s}", f.Name, f.Type.Name)
	case Slice:
		return fmt.Sprintf("{%s: %s}", f.Name, f.Type.Name)
	case Map:
		return fmt.Sprintf("{%s: %s}", f.Name, f.Type.Name)
	case Func:
		return fmt.Sprintf("{%s: %s}", f.Name, f.Type.Name)
	case Other:
		if f.Other != nil {
			return fmt.Sprintf("{%s(%s): %v}", f.Name, f.Type.Name, f.Other)
		} else {
			return fmt.Sprintf("{%s: %s}", f.Name, f.Type.Name)
		}
	}
	return "{invalid field}"
}

func (f Field) PrettyString() string {
	return f.PrettySerialize(0)
}

func (f Field) PrettySerialize(depth int) string {
	space := ""
	for i := 0; i < depth; i++ {
		space += "\t"
	}
	switch f.Kind() {
	case Struct:
		return fmt.Sprintf("%s(%s): %s", f.Name, f.Type.Name, f.Struct.PrettySerialize(depth))
	case Pointer:
		return fmt.Sprintf("%s%s(%s): %d", space, f.Name, f.Type.Name, f.Pointer)
	case Slice:
		return fmt.Sprintf("%s%s(%s): %d", space, f.Name, f.Type.Name, f.Slice)
	case Map:
		return fmt.Sprintf("%s%s(%s): %s", space, f.Name, f.Type.Name, "Map")
	case Func:
		return fmt.Sprintf("%s%s(%s): %s", space, f.Name, f.Type.Name, "Func")
	case Other:
		return fmt.Sprintf("%s%s(%s): %v", space, f.Name, f.Type.Name, f.Other)
	}
	panic("Trying to print an uninitialized field")
}

type Kind uint

const (
	Pointer Kind = iota
	Struct
	Slice
	Map
	Func
	Other
)
