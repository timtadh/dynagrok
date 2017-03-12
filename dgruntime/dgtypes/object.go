// Defines types that serve as our runtime interpretation of objects, types, and
// fields.

package dgtypes

import (
	"fmt"
	"encoding/binary"
	"hash"
)

const Depth = 7

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

type ObjectProfile []*Value

type FuncProfile struct {
	FuncName string
	In       []ObjectProfile
	Out      []ObjectProfile
}

func (fp FuncProfile) Vector() []float64 {
	return []float64{0}
}

type ObjectType struct {
	Name    string
	Pointer bool
}

type StructT struct {
	Type      ObjectType
	Fields    []Field
	Reference uintptr
}

func (i *StructT) LeveledHash(h hash.Hash, n int) {
	for _, f := range i.Fields {
		switch f.Val.Kind() {
		case Struct:
			f.Val.Struct.LeveledHash(h, n-1)
		}
	}
}

func NewShallowStruct(tipe ObjectType, data_ptr uintptr) *StructT {
	return &StructT{tipe, nil, data_ptr}
}

func (o *StructT) getExportedFields() []Field {
	var exported []Field = make([]Field, 0)
	for _, f := range o.Fields {
		if f.Exported {
			exported = append(exported, f)
		}
	}
	return exported
}

type Field struct {
	Name     string
	Val      Value
	Exported bool
}

// TODO(KOBY): refactor this value into an interface.
//
// type Value interface {
// 	Kind() Kind
// 	LevelHash(hash.Hash, int)
// 	Value() interface{}
// 	String() string
// }
//
// type IntValue struct {
// 	kind Kind
// 	val uint64
// }
// 
// func IntValue(i interface{}) *IntValue {
// 	switch x := i.(type) {
// 	case int8:
// 		return &IntValue{kind: Int8, val: uint64(x)}
// 	case uint8:
// 		return &IntValue{kind: UInt8, val: uint64(x)}
// 	case int16:
// 		return &IntValue{kind: Int16, val: uint64(x)}
// 	case uint16:
// 		return &IntValue{kind: UInt16, val: uint64(x)}
// 	case int32:
// 		return &IntValue{kind: Int32, val: uint64(x)}
// 	case uint32:
// 		return &IntValue{kind: UInt32, val: uint64(x)}
// 	case int64:
// 		return &IntValue{kind: Int64, val: uint64(x)}
// 	case uint64:
// 		return &IntValue{kind: UInt64, val: uint64(x)}
// 	case int:
// 		return &IntValue{kind: Int, val: uint64(x)}
// 	case uint:
// 		return &IntValue{kind: UInt, val: uint64(x)}
// 	case uintptr:
// 		return &IntValue{kind: UIntptr, val: uint64(x)}
// 	default:
// 		panic(fmt.Errorf("i %v should have been an int got %T", i, i)
// 	}
// }
// 
// func (i *IntValue) Kind() Kind {
// 	return i.kind
// }
// 
// func (i *IntValue) LeveledHash(h hash.Hash, n int) {
// 	if n <= 0 {
// 		return
// 	}
// 	binary.Write(h, binary.BigEndian, i.val)
// }
// 
// func (i *IntValue) Value() interface{} {
// 	switch i.kind {
// 	case Unt8:
// 		return int8(i.val)
// 	case Uint8:
// 		return uint8(i.val)
// 	case Int16:
// 		return int16(i.val)
// 	case Uint16:
// 		return uint16(i.val)
// 	case Int32:
// 		return int32(i.val)
// 	case Uint32:
// 		return uint32(i.val)
// 	case Int64:
// 		return int64(i.val)
// 	case Uint64:
// 		return uint64(i.val)
// 	case Int:
// 		return int(i.val)
// 	case Uint:
// 		return uint(i.val)
// 	case Uintptr:
// 		return uintptr(i.val)
// 	default:
// 		panic(fmt.Errorf("i %v should have been an int got %T", i, i)
// 	}
// }
// 
// func (i *IntValue) String() string {
// 	return fmt.Sprintf("%v", i.Value())
// }

type Value struct {
	Type    ObjectType
	Name    string
	Func    bool
	Map     bool
	Pointer uintptr
	Slice   uintptr
	Struct  *StructT
	// TODO distinguish types that are 'incomparable': slices, maps, and
	// functions
	Other interface{}
}

func (f *Value) Kind() Kind {
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

func (f Value) Hash(h hash.Hash) {
	//f.LeveledHash(h, Depth)
}

func (f Value) LeveledHash(h hash.Hash, n int) {
	if n == 0 {
		fmt.Printf("Depth level reached while hashing")
		return
	}
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
