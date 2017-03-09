package dgtypes

import (
	"bytes"
	"encoding/json"
	"fmt"
)

/*
 * FuncProfile
 */
func (p FuncProfile) Serialize() string {
	b := new(bytes.Buffer)
	e := json.NewEncoder(b)
	e.Encode(p)
	return b.String()
}

func UnserializeFunc(str string) FuncProfile {
	var prof FuncProfile
	json.Unmarshal([]byte(str), &prof)
	return prof
}

/*
 * Instance
 */

func (o *StructT) PrettyString() string {
	return o.PrettySerialize(0)
}

// PrettyString() with nesting
func (o *StructT) PrettySerialize(depth int) string {
	space := ""
	for i := 0; i < depth; i++ {
		space += "\t"
	}
	str := fmt.Sprintf("%s%s: { Reference: %d\n", space, o.Type.Name, o.Reference)
	str += space + "  Fields: \n"
	for _, f := range o.Fields {
		str += fmt.Sprintf("%s%v\n", space, f.Val.PrettySerialize(depth+1))
	}
	if len(o.Fields) == 0 {
		str += space + "\t<no fields>" + "\n"
	}
	return str + fmt.Sprintf("%s }", space)
}

/*
 * Field
 */
// A compact one-line representation of a field
func (f Value) CompactString() string {
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

func (f Value) PrettyString() string {
	return f.PrettySerialize(0)
}

func (f Value) PrettySerialize(depth int) string {
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
