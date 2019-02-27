package dgtypes

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"time"
	"unsafe"
)

type Function struct {
	Name   string
	FuncPc uintptr
	CFG    [][]int
	IPDom  []int
	PDG    string
	Calls  int
	DynCDP []map[int]bool // Dynamic Control Dependence Predecessors
	Values map[VarReference]*ValueReservoir
}

type ExportFunction struct {
	CFG    [][]int
	IPDom  []int
	PDG    string
	Calls  int
	DynCDP [][]int // Dynamic Control Dependence Predecessors
	Values []struct {
		VarReference VarReference
		Values       [][]byte
	}
}

type ValueReservoir struct {
	maxSize int
	values  []Value
	rand    *rand.Rand
}

func NewValueReservoir(size int) (r *ValueReservoir) {
	return &ValueReservoir{
		maxSize: size,
		values:  make([]Value, 0, size),
		rand:    rand.New(rand.NewSource(1)),
	}
}

func (r *ValueReservoir) Export() [][]byte {
	exported := make([][]byte, 0, len(r.values))
	for _, v := range r.values {
		exported = append(exported, []byte(v))
	}
	return exported
}

func (r *ValueReservoir) Add(v Value) {
	if r.has(v) {
		return
	}
	if len(r.values) < r.maxSize {
		r.values = append(r.values, v)
		return
	}
	smallest, largest := r.extremes()
	if v.Compare(r.values[smallest]) < 0 {
		r.values[smallest] = v
	} else if v.Compare(r.values[largest]) > 0 {
		r.values[largest] = v
	} else if r.rand.Float64() > .1 {
		return
	}
	r.values[r.rand.Intn(len(r.values))] = v
}

func (r *ValueReservoir) Merge(o *ValueReservoir) {
	for _, v := range o.values {
		r.Add(v)
	}
}

func (r *ValueReservoir) extremes() (smallest, largest int) {
	for idx, v := range r.values {
		if v.Compare(r.values[smallest]) < 0 {
			smallest = idx
		} else if v.Compare(r.values[largest]) > 0 {
			largest = idx
		}
	}
	return smallest, largest
}

func (r *ValueReservoir) has(v Value) bool {
	for _, cur := range r.values {
		if v.Compare(cur) == 0 {
			return true
		}
	}
	return false
}

type Value []byte

func InterfaceToValue(i interface{}) (Value, error) {
	val := reflect.ValueOf(i)
	switch val.Kind() {
	case reflect.Bool:
		if val.Bool() {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		i := val.Int()
		sh := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(&i)), Len: 8, Cap: 8}
		src := (*[]byte)(unsafe.Pointer(&sh))
		dest := make([]byte, 8)
		copy(dest, *src)
		return dest, nil
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.Uintptr:
		i := val.Uint()
		sh := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(&i)), Len: 8, Cap: 8}
		src := (*[]byte)(unsafe.Pointer(&sh))
		dest := make([]byte, 8)
		copy(dest, *src)
		return dest, nil
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		f := val.Float()
		sh := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(&f)), Len: 8, Cap: 8}
		src := (*[]byte)(unsafe.Pointer(&sh))
		dest := make([]byte, 8)
		copy(dest, *src)
		return dest, nil
	case reflect.Complex64:
		fallthrough
	case reflect.Complex128:
		c := val.Complex()
		sh := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(&c)), Len: 16, Cap: 16}
		src := (*[]byte)(unsafe.Pointer(&sh))
		dest := make([]byte, 16)
		copy(dest, *src)
		return dest, nil
	case reflect.String:
		return []byte(val.String()), nil
	default:
		return nil, fmt.Errorf("cannot convert a %v", val.Type())
	}
}

func (v Value) Compare(o Value) int {
	return bytes.Compare(v, o)
}

type FuncCall struct {
	Name     string
	FuncPc   uintptr
	CFG      [][]int
	IPDom    []int
	PDG      string
	CDStack  []int
	DynCDP   []map[int]bool // Dynamic Control Dependence Predecessors
	Last     BlkEntrance
	LastTime time.Time
	Values   map[VarReference]*ValueReservoir
}

type VarReference struct {
	VarName      string
	BlockId      int
	BlockStmtId  int
	GlobalStmtId int
}

func ExportFunctions(funcs map[uintptr]*Function) map[string]*ExportFunction {
	export := make(map[string]*ExportFunction, len(funcs))
	for _, fn := range funcs {
		dcdp := make([][]int, len(fn.DynCDP))
		for x, preds := range fn.DynCDP {
			dcdp[x] = make([]int, 0, len(preds))
			for y := range preds {
				dcdp[x] = append(dcdp[x], y)
			}
		}
		values := make([]struct {
			VarReference VarReference
			Values       [][]byte
		}, 0, 10)
		for k, v := range fn.Values {
			values = append(values, struct {
				VarReference VarReference
				Values       [][]byte
			}{k, v.Export()})
		}
		export[fn.Name] = &ExportFunction{
			CFG:    fn.CFG,
			IPDom:  fn.IPDom,
			PDG:    fn.PDG,
			Calls:  fn.Calls,
			DynCDP: dcdp,
			Values: values,
		}
	}
	return export
}

func NewFunction(fc *FuncCall) *Function {
	f := &Function{
		Name:   fc.Name,
		FuncPc: fc.FuncPc,
		CFG:    fc.CFG,
		IPDom:  fc.IPDom,
		PDG:    fc.PDG,
		DynCDP: fc.DynCDP,
		Values: fc.Values,
	}
	f.Update(fc)
	return f
}

func (f *Function) Merge(b *Function) {
	if f.FuncPc != b.FuncPc || f.Name != b.Name {
		panic("can't merge")
	}
	f.Calls += b.Calls
	for x, preds := range b.DynCDP {
		for pred := range preds {
			f.DynCDP[x][pred] = true
		}
	}
	for k, v := range b.Values {
		if f.Values[k] == nil {
			f.Values[k] = v
		} else {
			f.Values[k].Merge(v)
		}
	}
}

func (f *Function) Update(fc *FuncCall) {
	if f.FuncPc != fc.FuncPc || f.Name != fc.Name {
		panic("f not valid for fc")
	}
	f.Calls += 1
	for x, preds := range fc.DynCDP {
		for pred := range preds {
			f.DynCDP[x][pred] = true
		}
	}
	for k, v := range fc.Values {
		if f.Values[k] == nil {
			f.Values[k] = v
		} else {
			f.Values[k].Merge(v)
		}
	}
}
