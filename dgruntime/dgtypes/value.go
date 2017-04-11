// Defines types that serve as our runtime interpretation of objects, types, and
// fields.

package dgtypes

import (
	"encoding/binary"
	"fmt"
	"hash"
	"math"
	"reflect"
)

const Depth = 7

type Kind uint

const (
	Invalid Kind = iota
	Bool
	Uint
	Int
	Int8
	UInt8
	Int16
	UInt16
	Int32
	UInt32
	Int64
	UInt64
	UInt
	UIntptr
	Array
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	Other
	Zero // nil value
)

type Reference interface {
	IsNil() bool
}

type Value interface {
	Kind() Kind
	LevelHash(hash.Hash, int)
	Value() interface{}
	String() string
	TypeName() string
	Dissimilar(Value) float64
}

// {{{ IntValue
type IntValue struct {
	kind     Kind
	Val      uint64
	JSONType string
}

const intType = "IntValue"

func IntVal(i interface{}) *IntValue {
	switch x := i.(type) {
	case int8:
		return &IntValue{kind: Int8, Val: uint64(x), JSONType: intType}
	case uint8:
		return &IntValue{kind: UInt8, Val: uint64(x), JSONType: intType}
	case int16:
		return &IntValue{kind: Int16, Val: uint64(x), JSONType: intType}
	case uint16:
		return &IntValue{kind: UInt16, Val: uint64(x), JSONType: intType}
	case int32:
		return &IntValue{kind: Int32, Val: uint64(x), JSONType: intType}
	case uint32:
		return &IntValue{kind: UInt32, Val: uint64(x), JSONType: intType}
	case int64:
		return &IntValue{kind: Int64, Val: uint64(x), JSONType: intType}
	case uint64:
		return &IntValue{kind: UInt64, Val: uint64(x), JSONType: intType}
	case int:
		return &IntValue{kind: Int, Val: uint64(x), JSONType: intType}
	case uint:
		return &IntValue{kind: UInt, Val: uint64(x), JSONType: intType}
	case uintptr:
		return &IntValue{kind: UIntptr, Val: uint64(x), JSONType: intType}
	default:
		panic(fmt.Errorf("%v should have been an int got %T", i, i))
	}
}

func (i *IntValue) Kind() Kind {
	return i.kind
}

func (i *IntValue) LevelHash(h hash.Hash, n int) {
	if n <= 0 {
		return
	}
	binary.Write(h, binary.BigEndian, i.Val)
}

func (i *IntValue) Value() interface{} {
	switch i.kind {
	case Int8:
		return int8(i.Val)
	case UInt8:
		return uint8(i.Val)
	case Int16:
		return int16(i.Val)
	case UInt16:
		return uint16(i.Val)
	case Int32:
		return int32(i.Val)
	case UInt32:
		return uint32(i.Val)
	case Int64:
		return int64(i.Val)
	case UInt64:
		return uint64(i.Val)
	case Int:
		return int(i.Val)
	case UInt:
		return uint(i.Val)
	case UIntptr:
		return uintptr(i.Val)
	default:
		panic(fmt.Errorf("%v should have been an int got %T", i, i))
	}
}

func (i *IntValue) String() string {
	return fmt.Sprintf("%d", i.Val)
}

func (i *IntValue) TypeName() string {
	return "int"
}

func (i *IntValue) Dissimilar(o Value) float64 {
	if other, ok := o.(*IntValue); ok {
		score := math.Abs(math.Abs(float64(i.Val-other.Val)) / (math.Abs(float64(math.MaxUint64))))
		//score := math.Abs(math.Log(math.Abs(float64(i.Val-other.Val))) - math.Log(math.Abs(float64(math.MaxUint64))))
		//fmt.Printf("Dis(%v, %v) = %v", i, other, score)
		return score
	} else {
		panic("Dissimilar shoud be called on type int")
	}
}

// }}}

// {{{ StringValue
type StringValue struct {
	Val      string
	JSONType string
}

func StringVal(i interface{}) *StringValue {
	if x, ok := i.(string); ok {
		var s StringValue = StringValue{Val: x, JSONType: "StringValue"}
		return &s
	} else {
		panic(fmt.Errorf("%v should have been a string got %T", i, i))
	}
}

func (s *StringValue) Kind() Kind {
	return String
}

func (s *StringValue) Value() interface{} {
	return s.Val
}

func (s *StringValue) String() string {
	return s.Val
}

func (s *StringValue) LevelHash(h hash.Hash, i int) {
	h.Write([]byte(s.String()))
}

func (s *StringValue) TypeName() string {
	return "string"
}

func (s *StringValue) Dissimilar(o Value) float64 {
	score := 0.0
	if other, ok := o.(*StringValue); ok {
		length := len(s.Val)
		if len(other.Val) > len(s.Val) {
			length = len(other.Val)
		}
		// TODO Perform Hamming distance
		for i, l := range s.Val {
			if i == len(other.Val) {
				break
			}
			if l != []rune(other.Val)[i] {
				score += 1 / float64(length)
			}
		}
		score += math.Abs(float64(len(s.Val)-len(other.Val))) / float64(length)
		return score
	} else {
		panic("Dissimilar should be called on type string")
	}
}

// }}}

// {{{ BoolValue
type BoolValue struct {
	Val      bool
	JSONType string
}

func BoolVal(i interface{}) *BoolValue {
	if x, ok := i.(bool); ok {
		var b BoolValue = BoolValue{Val: x, JSONType: "BoolValue"}
		return &b
	} else {
		panic(fmt.Errorf("%v should have been a bool got %T", i, i))
	}
}

func (b *BoolValue) Kind() Kind {
	return Bool
}

func (b *BoolValue) LevelHash(h hash.Hash, i int) {
	x := bool(b.Val)
	if x {
		h.Write([]byte("1"))
	} else {
		h.Write([]byte("0"))
	}
}

func (b *BoolValue) Value() interface{} {
	return b.Val
}

func (b *BoolValue) String() string {
	x := bool(b.Val)
	return fmt.Sprintf("%v", x)
}

func (b *BoolValue) TypeName() string {
	return "bool"
}

func (b *BoolValue) Dissimilar(v Value) float64 {
	if other, ok := v.(*BoolValue); ok {
		switch {
		case bool(b.Val) && bool(other.Val):
			return 0.0
		case bool(b.Val) != bool(other.Val):
			return 1.0
		case !bool(b.Val) && !bool(other.Val):
			return 0.0
		}
	}
	panic("Should have been type bool")
}

// }}}

// {{{ StructValue
type StructValue struct {
	TypName  string
	Fields   []Field
	val      interface{}
	JSONType string
}

func StructVal(i interface{}) *StructValue {
	v := reflect.ValueOf(i)
	fields := make([]Field, 0)
	vType := v.Type()

	if vType.Kind() != reflect.Struct {
		panic(fmt.Errorf("%v should have been Struct, was %s", i, vType.Name()))
	}

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		name := vType.Field(i).Name
		if f.CanInterface() { // f.Interface() will fail otherwise
			fields = append(fields, Field{Name: name, exported: true, Val: NewVal(f.Interface())})
		} else {
			fields = append(fields, Field{Name: name, exported: false,
				Val: nil})
		}
	}
	return &StructValue{TypName: vType.Name(), Fields: fields, val: i, JSONType: "StructValue"}
}

func (s *StructValue) LevelHash(h hash.Hash, n int) {
	for _, f := range s.Fields {
		f.Val.LevelHash(h, n-1)
	}
}

func (s *StructValue) Kind() Kind {
	return Struct
}

func (s *StructValue) Value() interface{} {
	return s.val
}

func (s *StructValue) String() string {
	if len(s.Fields) == 0 {
		return fmt.Sprintf("struct %v{}", s.TypeName())
	}
	str := fmt.Sprintf("struct %v {%s", s.TypeName(), s.Fields[0].String())
	for i, f := range s.Fields {
		if i == 0 {
			continue
		}
		str = fmt.Sprintf("%s, %s", str, f.String())
	}
	str = fmt.Sprintf("%s}", str)
	return str
}

func (s *StructValue) TypeName() string {
	return s.TypName
}

func (s *StructValue) Dissimilar(v Value) float64 {
	score := 0.0
	if other, ok := v.(*StructValue); ok {
		if s.TypName != other.TypName {
			panic("Cannot compute similarity between structs of different type")
		}
		for i := range s.Fields {
			if sf, ok := s.Fields[i].Val.(Reference); ok {
				of, _ := other.Fields[i].Val.(Reference)
				if sf.IsNil() && !of.IsNil() ||
					!sf.IsNil() && of.IsNil() {
					score += 1 / float64(len(s.Fields))
				}
			} else {
				score += s.Fields[i].Val.Dissimilar(other.Fields[i].Val) / float64(len(s.Fields))
			}
		}
		return score
	}
	panic("Should have been type struct")
}

// }}}

type Field struct {
	Name     string
	Val      Value
	exported bool
}

func (f Field) String() string {
	return fmt.Sprintf("%v: %v", f.Name, f.Val)
}

// {{{ ReferenceValue
type ReferenceValue struct {
	val      interface{}
	Typename string
	Elem     Value
	kind     Kind
	JSONType string
}

func ReferenceVal(i interface{}) *ReferenceValue {
	val := reflect.ValueOf(i)
	elem := NewVal(val.Elem().Interface())
	switch val.Kind() {
	case reflect.Ptr:
		return &ReferenceValue{val: i, Elem: elem, kind: Pointer, Typename: "*" + elem.TypeName(), JSONType: "ReferenceValue"}
	case reflect.Interface:
		// TODO determine the typename for interfaces
		return &ReferenceValue{val: i, Elem: elem, kind: Interface, Typename: "*" + elem.TypeName(), JSONType: "ReferenceValue"}
	default:
		panic(fmt.Errorf("%v should be a reference, is %T", i, i))
	}
	panic("This code should be unreachable")
}

func (r *ReferenceValue) Kind() Kind {
	return r.kind
}

func (r *ReferenceValue) LevelHash(h hash.Hash, i int) {
	r.LevelHash(h, i)
}

func (r *ReferenceValue) Value() interface{} {
	return r.val
}

func (r *ReferenceValue) String() string {
	return r.Elem.String()
}

func (r *ReferenceValue) TypeName() string {
	return r.Typename
}

func (r *ReferenceValue) IsNil() bool {
	return r.val == nil
}

func (r *ReferenceValue) Dissimilar(v Value) float64 {
	if other, ok := v.(*ReferenceValue); ok {
		return r.Elem.Dissimilar(other.Elem)
	}
	panic("Should have been ReferenceValue")
}

// }}}

// {{{ ArrayValue
type ArrayValue struct {
	ElemType string
	Val      []Value
	val      interface{}
	size     int
	JSONType string
}

func ArrayVal(i interface{}) *ArrayValue {
	x := reflect.ValueOf(i)
	vals := make([]Value, x.Len())
	for k := range vals {
		vals[k] = NewVal(x.Index(k).Interface())
	}
	if x.Len() > 0 {
		t := vals[0].TypeName()
		return &ArrayValue{Val: vals, val: i, size: x.Len(), ElemType: t, JSONType: "ArrayValue"}
	} else {
		return &ArrayValue{Val: vals, val: i, size: x.Len(), JSONType: "ArrayValue"}
	}
}

func (a *ArrayValue) Kind() Kind {
	return Array
}

func (a *ArrayValue) LevelHash(h hash.Hash, i int) {
	for _, v := range a.Val {
		v.LevelHash(h, i)
	}
}

func (a *ArrayValue) Value() interface{} {
	return a.Val
}

func (a *ArrayValue) String() string {
	str := "{"
	if len(a.Val) > 0 {
		str = fmt.Sprintf("%v%v", str, a.Val[0].String())
	}
	for i, v := range a.Val {
		if i == 0 {
			continue
		}
		str = fmt.Sprintf("%v, %v", str, v.String())
	}
	str = fmt.Sprintf("%v}", str)
	return str
}

func (a *ArrayValue) IsNil() bool {
	return a.val == nil
}

func (a *ArrayValue) TypeName() string {
	return fmt.Sprintf("[]%v", a.ElemType)
}

func (a *ArrayValue) Dissimilar(v Value) float64 {
	score := 0.0
	if other, ok := v.(*ArrayValue); ok {
		length := len(a.Val)
		if len(other.Val) > len(a.Val) {
			length = len(other.Val)
		}
		// TODO Perform Hamming distance
		for i, l := range a.Val {
			if i == len(other.Val) {
				break
			}
			if l != other.Val[i] {
				score += 1 / float64(length)
			}
		}
		score += math.Abs(float64(len(a.Val)-len(other.Val))) / float64(length)
		return score
	}
	panic("Should have been array or slice type")
}

// }}}

// {{{ FuncValue
type FuncValue struct {
	name     string
	inTypes  []Type
	outTypes []Type
}

// }}}

func NewVal(i interface{}) Value {
	val := reflect.ValueOf(i)

	vType := val.Type()
	switch vType.Kind() {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return IntVal(val.Interface())
	case reflect.Bool:
		return BoolVal(val.Interface())
	case reflect.Ptr, reflect.Interface:
		return ReferenceVal(val.Interface())
	case reflect.Struct:
		return StructVal(val.Interface())
	case reflect.Array, reflect.Slice:
		return ArrayVal(val.Interface())
	case reflect.String:
		return StringVal(val.Interface())
	default:
		//panic(fmt.Errorf("%v has unidentified Kind %v", i, vType.Kind()))
		return nil
	}
	panic("This statement should be unreachable")
}
