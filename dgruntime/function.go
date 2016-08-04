package dgruntime

type Function struct {
	Name string
	RuntimeNames []string
	FuncPcs []uintptr
	CallPcs []uintptr
	Flows   []Flow
	Calls int
}

type FuncCall struct {
	Name, RuntimeName string
	FuncPc, CallPc uintptr
	Flow    []BlkEntrance
}

func newFunction(fc *FuncCall) *Function {
	f := &Function {
		Name: fc.Name,
		RuntimeNames: make([]string, 0, 10),
		FuncPcs: make([]uintptr, 0, 10),
		CallPcs: make([]uintptr, 0, 10),
	}
	f.Update(fc)
	return f
}

func (f *Function) Merge(b *Function) {
	f.Calls += b.Calls
	for _, bName := range b.RuntimeNames {
		hasName := false
		for _, name := range f.RuntimeNames {
			if name == bName {
				hasName = true
				break
			}
		}
		if !hasName {
			f.RuntimeNames = append(f.RuntimeNames, bName)
		}
	}
	for _, bCallPc := range b.CallPcs {
		hasCallPc := false
		for _, pc := range f.CallPcs {
			if pc == bCallPc {
				hasCallPc = true
				break
			}
		}
		if !hasCallPc {
			f.CallPcs = append(f.CallPcs, bCallPc)
		}
	}
	for _, bFuncPc := range b.FuncPcs {
		hasFuncPc := false
		for _, pc := range f.FuncPcs {
			if pc == bFuncPc {
				hasFuncPc = true
				break
			}
		}
		if !hasFuncPc {
			f.FuncPcs = append(f.FuncPcs, bFuncPc)
		}
	}
	for _, bFlow := range b.Flows {
		hasFlow := false
		for _, flow := range f.Flows {
			if flow.equals(bFlow) {
				hasFlow = true
				break
			}
		}
		if !hasFlow {
			f.Flows = append(f.Flows, bFlow)
		}
	}
}

func (f *Function) Update(fc *FuncCall) {
	f.Calls++
	hasName := false
	for _, name := range f.RuntimeNames {
		if name == fc.RuntimeName {
			hasName = true
			break
		}
	}
	if !hasName {
		f.RuntimeNames = append(f.RuntimeNames, fc.RuntimeName)
	}
	hasCallPc := false
	for _, pc := range f.CallPcs {
		if pc == fc.CallPc {
			hasCallPc = true
			break
		}
	}
	if !hasCallPc {
		f.CallPcs = append(f.CallPcs, fc.CallPc)
	}
	hasFuncPc := false
	for _, pc := range f.FuncPcs {
		if pc == fc.FuncPc {
			hasFuncPc = true
			break
		}
	}
	if !hasFuncPc {
		f.FuncPcs = append(f.FuncPcs, fc.FuncPc)
	}
	hasFlow := false
	for _, flow := range f.Flows {
		if flow.equals(fc.Flow) {
			hasFlow = true
			break
		}
	}
	if !hasFlow {
		f.Flows = append(f.Flows, fc.Flow)
	}
}

