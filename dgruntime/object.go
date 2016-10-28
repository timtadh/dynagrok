package dgruntime

// Todo: instrumentation
// Consider how to distinguish by type

type ObjectType struct {
	Name    string
	Fields  []ObjectType
	Methods []string
}

type Instance struct {
	Interface ObjectType
	Fields    map[string]interface{}
	History   []string
	Reference uintptr
}

func newObjectType(n string, f []ObjectType) *ObjectType {
	return &ObjectType{Name: n, Fields: f}
}

func newInstance(tipe ObjectType, initValues map[string]interface{}, ptr uintptr) *Instance {
	return &Instance{Interface: tipe, Fields: initValues, Reference: ptr, History: make([]string, 3)}
}

func (o *Instance) addCall(method string) {
	o.History = append(o.History, method)
}

func (o *Instance) snap(pos string) {
	println("snapping")
	exec.Profile.Instances[pos] = o
}
