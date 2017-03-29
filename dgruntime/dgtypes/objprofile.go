package dgtypes

import (
	"bytes"
	"encoding/json"
)

type ObjectProfile []Param

func (op ObjectProfile) Dissimilar(other ObjectProfile) float64 {
	distance := 0.0
	for param := range op {
		distance += op[param].Dissimilar(&other[param]) / float64(len(op))
	}
	return distance
}

type Param struct {
	Name string
	Val  Value
}

func (p *Param) Dissimilar(other *Param) float64 {
	return p.Val.Dissimilar(other.Val)
}

type FuncProfile struct {
	FuncName string
	In       []ObjectProfile
	Out      []ObjectProfile
}

type TypeProfile struct {
	Types []Type
}

func (tp TypeProfile) Serialize() string {
	b := new(bytes.Buffer)
	e := json.NewEncoder(b)
	e.Encode(tp)
	return b.String()
}

func UnserializeType(str string) TypeProfile {
	var prof TypeProfile
	json.Unmarshal([]byte(str), &prof)
	return prof
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
