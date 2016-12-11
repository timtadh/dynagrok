package dgruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Object struct {
	TypeName     string
	Ptr          uintptr
	Pos          string
	Data         [50]Field
	CallSequence [50]string
	NumFields    int
	NumCalls     int
}

func (o Object) String() string {
	b := fmt.Sprintf("{Type: %v, Ref: %v, Pos: %v, Fields: [%v", o.TypeName, o.Ptr, o.Pos, o.Data[0])
	for i := 1; i < o.NumFields; i++ {
		b += fmt.Sprintf(", %v", o.Data[i])
	}
	b += fmt.Sprintf("], Call Sequence: [%v", o.CallSequence[0])
	for i := 1; i < o.NumCalls; i++ {
		b += fmt.Sprintf(", %v", o.CallSequence[i])
	}
	b += "]}"
	return b
}

func NewObject(tname string, ptr uintptr, pos string, data []Field, callSequence []string) Object {
	fields, nf := iSliceToArray(data)
	calls, nc := sliceToArray(callSequence)
	return Object{TypeName: tname, Ptr: ptr, Pos: pos, Data: fields, CallSequence: calls, NumFields: nf, NumCalls: nc}
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
