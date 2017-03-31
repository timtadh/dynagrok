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

func (p *Param) UnmarshalJSON(bs []byte) error {
	// Construct a param to return
	p = &Param{}

	// Construct a structure which mimics param
	// but ignores the 'Val' field to be interpreted later
	var paramPart struct {
		Name string
		Val  json.RawMessage
	}
	json.Unmarshal(bs, &paramPart)
	p.Name = paramPart.Name

	// Put the raw Val into an object map
	var valMap map[string]*json.RawMessage
	json.Unmarshal(paramPart.Val, &valMap)
	typeMessage, ok := valMap[JSONField]
	fmt.Printf("RawMessage: %v\n", string(paramPart.Val))
	if !ok {
		fmt.Printf("This type doesn't serialize JSON does not have enough information to unserialize\n")
		return nil
	}
	var typeName string
	json.Unmarshal(*typeMessage, &typeName)

	delete(valMap, JSONField)

	bs, _ = json.Marshal(valMap)
	switch typeName {
	case "IntValue":
		return json.Unmarshal(bs, &p.Val)
	default:
		return nil
	}
}
