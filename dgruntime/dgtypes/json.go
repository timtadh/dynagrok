package dgtypes

import (
	"encoding/json"
	"fmt"
)

const (
	JSONField = "JSONType"
)

type paramPart struct {
	Name string
	Val  json.RawMessage
}

func (p *Param) UnmarshalJSON(bs []byte) error {
	// Construct a structure which mimics param
	// but ignores the 'Val' field to be interpreted later
	var pp paramPart
	json.Unmarshal(bs, &pp)
	p.Name = pp.Name
	var err error
	p.Val, err = valueFromRaw(&pp.Val)

	if err != nil {
		fmt.Errorf("Error unmarshaling: %v\n", err)
	}
	return err
}

func valueFromRaw(raw *json.RawMessage) (Value, error) {
	//fmt.Printf("RawMessage: %v\n", string(*raw))
	// Put the raw Val into an object map
	// and check if it has a field indicating its concrete type
	if raw == nil {
		return nil, nil
	}
	var valMap map[string]*json.RawMessage
	json.Unmarshal(*raw, &valMap)
	typeMessage, ok := valMap[JSONField]
	if !ok {
		return nil, fmt.Errorf("This type doesn't have enough information to unserialize\n")
	}
	var typeName string
	json.Unmarshal(*typeMessage, &typeName)

	var err error
	switch typeName {
	case "IntValue":
		var i IntValue
		err = json.Unmarshal(*raw, &i)
		return &i, err
	case "StructValue":
		var s StructValue
		var name string
		json.Unmarshal(*valMap["TypName"], &name)
		var rawfields []paramPart
		json.Unmarshal(*valMap["Fields"], &rawfields)
		json.Unmarshal(*valMap["JSONType"], &s.JSONType)
		var fields []Field = make([]Field, len(rawfields))
		for i := range rawfields {
			fields[i].Name = rawfields[i].Name
			fields[i].Val, err = valueFromRaw(&rawfields[i].Val)
		}
		s.TypName = name
		s.Fields = fields
		return &s, err
	case "StringValue":
		var s StringValue
		err = json.Unmarshal(*raw, &s)
		return &s, err
	case "BoolValue":
		var s BoolValue
		err = json.Unmarshal(*raw, &s)
		return &s, err
	case "ReferenceValue":
		var r ReferenceValue
		json.Unmarshal(*valMap["Typename"], &r.Typename)
		json.Unmarshal(*valMap["JSONType"], &r.JSONType)
		r.Elem, err = valueFromRaw(valMap["Elem"])
		return &r, err
	case "ArrayValue":
		var a ArrayValue
		json.Unmarshal(*valMap["ElemType"], &a.ElemType)
		json.Unmarshal(*valMap["JSONType"], &a.JSONType)
		var rawVals []*json.RawMessage
		json.Unmarshal(*valMap["Val"], &rawVals)
		a.Val = make([]Value, len(rawVals))
		for i := range rawVals {
			a.Val[i], err = valueFromRaw(rawVals[i])
		}
		return &a, err

	default:
		fmt.Printf("Unrecognized JSON type %s", typeName)
		return nil, error(fmt.Errorf("Unrecognized JSON type %s", typeName))
	}
}
