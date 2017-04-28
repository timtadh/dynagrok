package dgtypes

import (
	"fmt"
	"reflect"
)

type Type interface {
	Name() string
}

type PrimitiveType struct {
	Tname string
}

func NewPrimitiveType(typ reflect.Type) *PrimitiveType {
	switch typ.Kind() {
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
		return &PrimitiveType{Tname: "Int"}
	case reflect.Bool:
		return &PrimitiveType{Tname: "Bool"}
	case reflect.String:
		return &PrimitiveType{Tname: "string"}
	case reflect.Float64:
		return &PrimitiveType{Tname: "float"}
	default:
		panic("Unrecognizable type")
	}

}

func (p *PrimitiveType) Name() string {
	return p.Tname
}

type PointerType struct {
	Tname string
	Elem  Type
}

func NewReferenceType(typ reflect.Type) *PointerType {
	if typ.Kind() == reflect.Ptr {
		return &PointerType{Tname: "*" + typ.Elem().Name(), Elem: newType(typ.Elem())}
	} else {
		panic("Expected pointer type")
	}
}

func (p *PointerType) Name() string {
	return p.Tname
}

type InterfaceType struct {
	Tname   string
	Methods []FuncType
}

func NewInterfaceType(typ reflect.Type) *InterfaceType {
	return &InterfaceType{Tname: typ.Name(), Methods: nil}
}

func (i *InterfaceType) Name() string {
	return i.Tname
}

type FuncType struct {
	Receivers map[string]Type
	Inputs    map[string]Type
	Outputs   map[Type]string
}

type CollectionType struct {
	Tname string
	Elem  Type
	Len   int
}

func NewCollectionType(typ reflect.Type) *CollectionType {
	switch typ.Kind() {
	case reflect.Array:
		len := typ.Len()
		name := fmt.Sprintf("[%d]%s", len, typ.Elem().Name())
		return &CollectionType{Tname: name, Elem: newType(typ.Elem()), Len: len}
	case reflect.Slice:
		name := fmt.Sprintf("[]%s", typ.Elem().Name())
		return &CollectionType{Tname: name, Elem: newType(typ.Elem())}
	default:
		panic("Expected slice or array")
	}
}

func (p *CollectionType) Name() string {
	return p.Tname
}

type StructType struct {
	Tname  string
	Fields map[string]Type
}

func NewStructType(typ reflect.Type) *StructType {
	fields := make(map[string]Type)
	if typ.Kind() == reflect.Struct {
		for k := 0; k < typ.NumField(); k++ {
			//if typ.Field(k).Type == typ {
			//
			//}
			//fields[typ.Field(k).Name] = newType(typ.Field(k).Type)
		}
		return &StructType{Tname: typ.Name(), Fields: fields}
	}
	panic("Expected struct type")

}

func (s *StructType) Name() string {
	return s.Tname
}

func newType(typ reflect.Type) Type {
	switch typ.Kind() {
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
		reflect.Uintptr,
		reflect.String,
		reflect.Bool:
		return NewPrimitiveType(typ)
	case reflect.Ptr:
		return NewReferenceType(typ)
	case reflect.Interface:
		return NewInterfaceType(typ)
	case reflect.Struct:
		return NewStructType(typ)
	case reflect.Array, reflect.Slice:
		return NewCollectionType(typ)
	default:
		//panic(fmt.Errorf("%v has unidentified Kind %v", i, vType.Kind()))
		return nil

	}
}

func NewType(i interface{}) Type {
	typ := reflect.TypeOf(i)
	return newType(typ)
}
