package dgtypes

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Clusterable interface {
	Dissimilar(Clusterable) float64
}

type ObjectProfile []Param

func (op ObjectProfile) Dissimilar(other Clusterable) float64 {
	if o, ok := other.(ObjectProfile); ok {
		distance := 0.0
		for param := range op {
			distance += op[param].Dissimilar(&o[param]) / float64(len(op))
		}
		return distance
	} else {
		panic("expected type ObjectProfile")
	}
}

type Param struct {
	Name string
	Val  Value
}

func (p Param) String() string {
	return fmt.Sprintf("{Name: %v, Val: %v}", p.Name, p.Val)
}

func (p *Param) Dissimilar(other Clusterable) float64 {
	if o, ok := other.(*Param); ok {
		//		fmt.Printf("Param: %v\n dissimilar from \nParam: %v\n", *p, *o)
		return p.Val.Dissimilar(o.Val)
	} else {
		panic("expected type *Param")
	}
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
	err := json.Unmarshal([]byte(str), &prof)
	if err != nil {
		fmt.Errorf("Error unmarshaling FuncProfile: %v", err)
	}
	return prof
}
