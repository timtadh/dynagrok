package dgtypes

import (
	"encoding/json"
	"fmt"
)

const (
	JSONField = "JSONType"
)

func SerializeWtihType(v Value, typename string) ([]byte, error) {
	bs, err := json.Marshal(v)
	var typeMessage json.RawMessage
	typeMessage, _ = json.Marshal(typename)
	if err != nil {
		panic("problem with json")
	}
	var object map[string]*json.RawMessage
	json.Unmarshal(bs, &object)
	object[JSONField] = &typeMessage
	bs, err = json.Marshal(v)
	return bs, err
}

//func (i *IntValue) MarshalJSON() ([]byte, error) {
//	return SerializeWtihType(i, "IntValue")
//}

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
		//fmt.Printf("Unmarshaled %v\n", i)
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
			//fmt.Printf("Unmarshaled field %v\n", fields[i])
		}
		s.TypName = name
		s.Fields = fields
		//fmt.Printf("Unmarshaled struct %v\n", s)
		return &s, err
	case "StringValue":
		var s StringValue
		err = json.Unmarshal(*raw, &s)
		//fmt.Printf("Unmarshaled %v\n", s)
		return &s, err
	default:
		return nil, nil
	}
}
