package dgruntime

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
)

type Profile struct {
	Objects   map[string]ObjectType
	Instances map[string][]Instance
	Funcs     map[uintptr]*Function
	Calls     map[Call]int
	Flows     map[FlowEdge]int
	Positions map[BlkEntrance]string
	CallCount int
}

func (p *Profile) Serialize(fout io.Writer) {
	runtime_name := func(pc uintptr) string {
		return runtime.FuncForPC(pc).Name()
	}
	blk_name := func(n BlkEntrance) string {
		if n.In == 0 && n.BlkId == 0 && n.At == 0 {
			return "entry"
		}
		if f, has := p.Funcs[n.In]; has {
			return fmt.Sprintf("%v blk %d:%d", f.Name, n.BlkId, n.At)
		} else {
			return fmt.Sprintf("%v blk %d:%d", runtime_name(n.In), n.BlkId, n.At)
		}

	}
	// flowlist := func(flows []Flow) string {
	// 	items := make([]string, 0, len(flows))
	// 	for _, f := range flows {
	// 		items = append(items, f.String())
	// 	}
	// 	return strlist(items)
	// }
	nextid := 1
	// fids := make(map[uintptr]int)
	// fids[0] = 0
	blks := make(map[BlkEntrance]int)
	fmt.Fprintf(fout, "digraph {\n")
	entry := blk_name(BlkEntrance{})
	fmt.Fprintf(fout, "%d [label=%v, shape=rect];\n",
		0,
		strconv.Quote(entry),
	)
	blks[BlkEntrance{}] = 0
	// fmt.Fprintf(fout, "0 [label=\"entry\", shape=rect];\n")
	// for _, f := range p.Funcs {
	// 	// fid := nextid
	// 	// nextid++
	// 	// fids[f.FuncPc] = fid
	// 	// fmt.Fprintf(fout, "%d [label=%v, shape=rect, calls=%d, entry_pc=%v, runtime_name=%v];\n",
	// 	// 	fid,
	// 	// 	strconv.Quote(f.Name),
	// 	// 	f.Calls,
	// 	// 	f.FuncPc,
	// 	// 	strconv.Quote(name(f.FuncPc)),
	// 	// )
	// }
	for e, _ := range p.Flows {
		src := blk_name(e.Src)
		targ := blk_name(e.Targ)
		if _, has := blks[e.Src]; !has {
			s := nextid
			nextid++
			fmt.Fprintf(fout, "%d [label=%v, shape=rect, position=%v, runtime_name=%v];\n",
				s,
				strconv.Quote(src),
				strconv.Quote(p.Positions[e.Src]),
				strconv.Quote(runtime_name(e.Src.In)),
			)
			blks[e.Src] = s
		}
		if _, has := blks[e.Targ]; !has {
			t := nextid
			nextid++
			fmt.Fprintf(fout, "%d [label=%v, shape=rect, position=%v, runtime_name=%v];\n",
				t,
				strconv.Quote(targ),
				strconv.Quote(p.Positions[e.Targ]),
				strconv.Quote(runtime_name(e.Targ.In)),
			)
			blks[e.Targ] = t
		}
	}
	// for call, count := range p.Calls {
	// 	fmt.Fprintf(fout, "%v -> %v [calls=%d, weight=%f];\n",
	// 		fids[call.Caller], fids[call.Callee],
	// 		count, float64(count)/float64(p.CallCount))
	// }
	// for _, fid := range fids {
	// 	targ := fmt.Sprintf("fn_%d_blk_%d_at_%d", fid, 0, 0)
	// 	if targId, has := blks[targ]; has {
	// 		fmt.Fprintf(fout, "%v -> %v;\n",
	// 			fid, targId)
	// 	}
	// }
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

func (p *Profile) PrettyObjectState(fout io.Writer) {
	for pos, instslice := range p.Instances {
		fmt.Fprintln(fout)
		fmt.Fprintln(fout, pos)
		for _, inst := range instslice {
			fmt.Fprintln(fout, inst.PrettyString())
		}
	}
}

func (p *Profile) SerializeObjectState(fout io.Writer) {
	for pos, instslice := range p.Instances {
		for _, inst := range instslice {
			fmt.Fprint(fout, inst.Serialize(pos))
		}
	}
}
