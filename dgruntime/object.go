// Defines types that serve as our runtime interpretation of objects, types, and
// fields.

package dgruntime

import (
	"encoding/binary"
	"fmt"
	"hash"
)

const Depth = 7

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

type Kind uint

const (
	Invalid Kind = iota
	Bool
	Uint
	Int
	//Array
	Func
	//Interface
	Map
	Pointer
	Slice
	String
	Struct
	Other
	Zero // nil value
)

type Field struct {
	Name     string
	Type     ObjectType
	Exported bool
	Func     bool
	Map      bool
	Pointer  uintptr
	Slice    uintptr
	Struct   *Instance
	// TODO distinguish types that are 'uncomparable': slices, maps, and
	// functions
	Other interface{}
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
	if f.Other != nil {
		switch f.Other.(type) {
		case uint:
			return Uint
		case int:
			return Int
		case bool:
			return Bool
		case string:
			return String
		default:
			return Other
		}
	}
	return Invalid
}

func (f Field) Hash(h hash.Hash) {
	f.LeveledHash(h, Depth)
}

func (f Field) LeveledHash(h hash.Hash, n int) {
	h.Write([]byte(f.Name))
	h.Write([]byte(f.Type.Name))
	switch f.Kind() {
	case Bool:
		if test, ok := f.Other.(bool); ok {
			if test {
				h.Write([]byte("1"))
			} else {
				h.Write([]byte("0"))
			}
		}
	case Int, Uint:
		binary.Write(h, binary.BigEndian, f.Other)
	case String:
		h.Write([]byte(f.Other.(string)))
	case Func:
		panic("not implemented")
	case Map:
		panic("not implemented")
	case Pointer:
		panic("not implemented")
	case Slice:
		panic("not implemented")
	case Struct:
		f.Struct.LeveledHash(h, n)
	}
}

func (i *Instance) LeveledHash(h hash.Hash, n int) {
	for _, f := range i.Fields {
		f.LeveledHash(h, n-1)
	}
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

// Takes a snapshot of the object state
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

// Converts an instance into a snap.Object (the object type specifically defined
// as an interface between dynagrok and other applications). Then, calls that
// object's serialization method.
func (o *Instance) Serialize(pos string) string {
	obj := NewObject(o.Interface.Name, 0, pos, o.getExportedFields(), o.History)
	return SerializeObject(obj)
}

// PrettyString() with nesting // TODO: rename
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

// A compact one-line representation of a field
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
	case Other, Int, Uint, String, Bool:
		if f.Other != nil {
			return fmt.Sprintf("{%s(%s): %v}", f.Name, f.Type.Name, f.Other)
		} else {
			return fmt.Sprintf("{%s: %s}", f.Name, f.Type.Name)
		}
	case Zero:
		return fmt.Sprintf("{%s: nil}", f.Name)
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
