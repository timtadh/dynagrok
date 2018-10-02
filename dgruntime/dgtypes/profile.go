package dgtypes

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"time"
)

type Profile struct {
	Inputs    map[string][]ObjectProfile
	Outputs   map[string][]ObjectProfile
	Types     map[string]Type
	Funcs     map[uintptr]*Function
	Calls     map[Call]int
	Flows     map[FlowEdge]int
	Positions map[BlkEntrance]string
	Durations map[BlkEntrance]time.Duration
	CallCount int
}

func NewProfile() *Profile {
	return &Profile{
		Calls:     make(map[Call]int),
		Funcs:     make(map[uintptr]*Function),
		Flows:     make(map[FlowEdge]int),
		Positions: make(map[BlkEntrance]string),
		Durations: make(map[BlkEntrance]time.Duration),
		Inputs:    make(map[string][]ObjectProfile),
		Outputs:   make(map[string][]ObjectProfile),
		Types:     make(map[string]Type),
	}
}

type Call struct {
	Caller uintptr
	Callee uintptr
}

func (p *Profile) Empty() bool {
	return len(p.Flows) == 0
}

func (p *Profile) WriteFunctions(fout io.Writer) error {
	e := json.NewEncoder(fout)
	// e.SetIndent("", "  ")
	fmt.Println("Functions", p.Funcs)
	exported := ExportFunctions(p.Funcs)
	fmt.Println(exported)
	return e.Encode(exported)
}

func (p *Profile) WriteDotty(fout io.Writer) {
	nextid := 1
	blks := make(map[BlkEntrance]int)
	fmt.Fprintf(fout, "digraph {\n")
	entry := p.blk_name(BlkEntrance{})
	fmt.Fprintf(fout, "%d [label=%v, shape=rect];\n",
		0,
		strconv.Quote(entry),
	)
	blks[BlkEntrance{}] = 0
	for e, _ := range p.Flows {
		src := p.blk_name(e.Src)
		targ := p.blk_name(e.Targ)
		if _, has := blks[e.Src]; !has {
			s := nextid
			nextid++
			fmt.Fprintf(fout, "%d [label=%v, shape=rect, position=%v, runtime_name=%v, fn_name=%v, bbid=%d, duration=%v];\n",
				s,
				strconv.Quote(src),
				strconv.Quote(p.Positions[e.Src]),
				strconv.Quote(p.runtime_name(e.Src.In)),
				strconv.Quote(p.fn_name(e.Src)),
				e.Src.BasicBlockId,
				strconv.Quote(p.Durations[e.Src].String()),
			)
			blks[e.Src] = s
		}
		if _, has := blks[e.Targ]; !has {
			t := nextid
			nextid++
			fmt.Fprintf(fout, "%d [label=%v, shape=rect, position=%v, runtime_name=%v, fn_name=%v, bbid=%d, duration=%v];\n",
				t,
				strconv.Quote(targ),
				strconv.Quote(p.Positions[e.Targ]),
				strconv.Quote(p.runtime_name(e.Targ.In)),
				strconv.Quote(p.fn_name(e.Targ)),
				e.Targ.BasicBlockId,
				strconv.Quote(p.Durations[e.Targ].String()),
			)
			blks[e.Targ] = t
		}
	}
	for e, count := range p.Flows {
		if _, has := blks[e.Src]; !has {
			continue
		}
		if _, has := blks[e.Targ]; !has {
			continue
		}
		fmt.Fprintf(fout, "%v -> %v [traversed=%d];\n",
			blks[e.Src], blks[e.Targ], count)
	}
	fmt.Fprintln(fout, "}\n\n")
}

func (p *Profile) runtime_name(pc uintptr) string {
	return runtime.FuncForPC(pc).Name()
}

func (p *Profile) fn_name(n BlkEntrance) string {
	if n.In == 0 && n.BasicBlockId == 0 {
		return "entry"
	}
	if f, has := p.Funcs[n.In]; has {
		return f.Name
	} else {
		return "unknown"
	}
}

func (p *Profile) blk_name(n BlkEntrance) string {
	if n.In == 0 && n.BasicBlockId == 0 {
		return "entry"
	}
	if f, has := p.Funcs[n.In]; has {
		return fmt.Sprintf("%v blk %d", f.Name, n.BasicBlockId)
	} else {
		return fmt.Sprintf("%v blk %d", p.runtime_name(n.In), n.BasicBlockId)
	}
}

func (p *Profile) WriteSimple(fout io.Writer) {
	nextid := 1
	blks := make(map[BlkEntrance]int)
	entry := p.blk_name(BlkEntrance{})
	fmt.Fprintln(fout, "start-graph")
	fmt.Fprintf(fout, "vertex\t%d, %v, %d, %v, %v, %v\n",
		0,
		strconv.Quote(entry),
		0,
		strconv.Quote("entry"),
		strconv.Quote("<none>"),
		strconv.Quote("0s"),
	)
	blks[BlkEntrance{}] = 0
	for e, _ := range p.Flows {
		src := p.blk_name(e.Src)
		targ := p.blk_name(e.Targ)
		if _, has := blks[e.Src]; !has {
			s := nextid
			nextid++
			fmt.Fprintf(fout, "vertex\t%d, %v, %d, %v, %v, %v\n",
				s,
				strconv.Quote(src),
				e.Src.BasicBlockId,
				strconv.Quote(p.fn_name(e.Src)),
				strconv.Quote(p.Positions[e.Src]),
				strconv.Quote(p.Durations[e.Src].String()),
			)
			blks[e.Src] = s
		}
		if _, has := blks[e.Targ]; !has {
			t := nextid
			nextid++
			fmt.Fprintf(fout, "vertex\t%d, %v, %d, %v, %v, %v\n",
				t,
				strconv.Quote(targ),
				e.Targ.BasicBlockId,
				strconv.Quote(p.fn_name(e.Targ)),
				strconv.Quote(p.Positions[e.Targ]),
				strconv.Quote(p.Durations[e.Targ].String()),
			)
			blks[e.Targ] = t
		}
	}
	for e, count := range p.Flows {
		if _, has := blks[e.Src]; !has {
			continue
		}
		if _, has := blks[e.Targ]; !has {
			continue
		}
		fmt.Fprintf(fout, "edge\t%d, %d, %d\n",
			blks[e.Src], blks[e.Targ], count)
	}
	fmt.Fprintln(fout, "end-graph")
}

func LoadSimple(fout io.Writer) (*Profile, error) {
	p := NewProfile()
	return p, nil
}

func (p *Profile) SerializeProfs(fout io.Writer) {
	types := make([]Type, len(p.Types))
	for _, typ := range p.Types {
		types = append(types, typ)
	}
	fmt.Fprint(fout, TypeProfile{types}.Serialize())
	for fname := range p.Inputs {
		if _, ok := p.Outputs[fname]; ok {
			fmt.Fprint(fout, FuncProfile{fname, p.Inputs[fname], p.Outputs[fname]}.Serialize())
		} else {
			fmt.Fprint(fout, FuncProfile{fname, p.Inputs[fname], []ObjectProfile{}}.Serialize())
		}
	}
	for fname, profs := range p.Outputs {
		if _, ok := p.Inputs[fname]; !ok {
			fmt.Fprint(fout, FuncProfile{fname, []ObjectProfile{}, profs}.Serialize())
		}
	}
}
