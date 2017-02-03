package dgruntime

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"strings"
)

type Object struct {
	TypeName     string
	Ptr          uintptr
	Pos          string
	Data         []Field
	CallSequence []string
	NumFields    int
	NumCalls     int
}

func (o Object) String() string {
	b := fmt.Sprintf("{Type: %v, Pos: %v, Fields: [", o.TypeName, trimPos(o.Pos))
	for i, f := range o.Data {
		if i != 0 {
			b += fmt.Sprintf(", ")
		}
		b += fmt.Sprintf("%v", f.CompactString())
	}
	b += fmt.Sprintf("], Call Sequence: [")
	for i, c := range o.CallSequence {
		if i != 0 {
			b += fmt.Sprintf(", ")
		}
		b += fmt.Sprintf("%v", c)
	}
	b += "]}"
	return b
}

func (obj Object) Hash() uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(obj.TypeName))
	binary.Write(hash, binary.BigEndian, obj.Ptr)
	for _, call := range obj.CallSequence {
		hash.Write([]byte(call))
	}
	for _, field := range obj.Data {
		field.Hash(hash)
	}
	return hash.Sum64()
}

func NewObject(tname string, ptr uintptr, pos string, data []Field, callSequence []string) Object {
	_, nf := iSliceToArray(data)
	_, nc := sliceToArray(callSequence)
	return Object{TypeName: tname, Ptr: ptr, Pos: pos, Data: data, CallSequence: callSequence, NumFields: nf, NumCalls: nc}
}

func SerializeObject(obj Object) string {
	b := new(bytes.Buffer)
	e := json.NewEncoder(b)
	e.Encode(obj)
	return b.String()
}

func UnserializeObject(str string) Object {
	var obj Object
	json.Unmarshal([]byte(str), &obj)
	return obj
}

func sliceToArray(sl []string) ([50]string, int) {
	var a [50]string
	var bound int
	if len(sl) > 50 {
		bound = len(sl) - 50
	} else {
		bound = 0
	}
	for i, e := range sl[bound:] {
		a[i] = e
	}
	return a, len(sl) - bound
}

func iSliceToArray(sl []Field) ([50]Field, int) {
	var a [50]Field
	var bound int
	if len(sl) > 1000 {
		bound = len(sl) - 1000
	} else {
		bound = 0
	}
	for i, e := range sl[bound:] {
		a[i] = e
	}
	return a, len(sl) - bound
}

func trimPos(pos string) string {
	elements := strings.Split(pos, ":")
	if len(elements) >= 3 {
		return filepath.Join(filepath.Base(filepath.Dir(elements[0])),
			filepath.Base(elements[0])) + ":" + elements[1] + ":" + elements[2]
	} else {
		return pos
	}
}
