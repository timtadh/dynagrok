package dgtypes

import (
	"bytes"
	"encoding/json"
)

type ObjectProfile []Param

type Param struct {
	Name string
	Val  Value
}

type FuncProfile struct {
	FuncName string
	In       []ObjectProfile
	Out      []ObjectProfile
}

func (fp FuncProfile) Vector() []float64 {
	return []float64{0}
}

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
