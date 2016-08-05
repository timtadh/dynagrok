package dgruntime

import (
	"io"
	"strconv"
	"runtime"
	"fmt"
)

type Profile struct {
	Funcs     map[uintptr]*Function
	Calls     map[Call]int
	CallCount int
}


func (p *Profile) Serialize(fout io.Writer) {
	name := func(pc uintptr) string {
		return runtime.FuncForPC(pc).Name()
	}
	// flowlist := func(flows []Flow) string {
	// 	items := make([]string, 0, len(flows))
	// 	for _, f := range flows {
	// 		items = append(items, f.String())
	// 	}
	// 	return strlist(items)
	// }
	nextfid := 1
	fids := make(map[uintptr]int)
	fids[0] = 0
	fmt.Fprintf(fout, "digraph {\n",)
	fmt.Fprintf(fout, "0 [label=\"entry\", shape=rect];\n")
	for _, f := range p.Funcs {
		fid := nextfid
		nextfid++
		fids[f.FuncPc] = fid
		fmt.Fprintf(fout, "%d [label=%v, shape=rect, calls=%d, entry_pc=%v, runtime_name=%v];\n",
			fid,
			strconv.Quote(f.Name),
			f.Calls,
			f.FuncPc,
			strconv.Quote(name(f.FuncPc)),
		)
	}
	for call, count := range p.Calls {
		fmt.Fprintf(fout, "%v -> %v [calls=%d, weight=%f];\n",
			fids[call.Caller], fids[call.Callee],
			count, float64(count)/float64(p.CallCount))
	}
	fmt.Fprintln(fout, "}\n\n")
}
