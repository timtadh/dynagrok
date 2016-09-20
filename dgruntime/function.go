package dgruntime

type Function struct {
	Name   string
	FuncPc uintptr
	Calls  int
}

type FuncCall struct {
	Name   string
	FuncPc uintptr
	Last   BlkEntrance
}

func newFunction(fc *FuncCall) *Function {
	f := &Function {
		Name: fc.Name,
		FuncPc: fc.FuncPc,
	}
	f.Update(fc)
	return f
}

func (f *Function) Merge(b *Function) {
	if f.FuncPc != b.FuncPc || f.Name != b.Name {
		panic("can't merge")
	}
	f.Calls += b.Calls
}

func (f *Function) Update(fc *FuncCall) {
	if f.FuncPc != fc.FuncPc || f.Name != fc.Name {
		panic("f not valid for fc")
	}
	f.Calls += 1
}

