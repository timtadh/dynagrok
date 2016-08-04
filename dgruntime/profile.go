package dgruntime

import (
	"io"
	"strconv"
	"fmt"
)

type Profile struct {
	Funcs     map[string]*Function
	Calls     map[Call]int
	CallCount int
}


func (p *Profile) Serialize(fout io.Writer) {
	strlist := func(list []string) string {
		str := "["
		for i, item := range list {
			str += item
			if i+1 < len(list) {
				str += ", "
			}
		}
		str += "]"
		return strconv.Quote(str)
	}
	intlist := func(list []uintptr) string {
		items := make([]string, 0, len(list))
		for _, i := range list {
			items = append(items, fmt.Sprintf("%v", i))
		}
		return strlist(items)
	}
	// flowlist := func(flows []Flow) string {
	// 	items := make([]string, 0, len(flows))
	// 	for _, f := range flows {
	// 		items = append(items, f.String())
	// 	}
	// 	return strlist(items)
	// }
	nextfid := 1
	fids := make(map[string]int)
	fids["<entry>"] = 0
	fmt.Fprintf(fout, "digraph {\n",)
	fmt.Fprintf(fout, "0 [label=\"entry\", shape=rect];\n")
	for _, f := range p.Funcs {
		fid := nextfid
		nextfid++
		fids[f.Name] = fid
		fmt.Fprintf(fout, "%d [label=%v, shape=rect, calls=%d, runtime_names=%v, entry_pcs=%v, call_pcs=%v];\n",
			fid, strconv.Quote(f.Name), f.Calls, strlist(f.RuntimeNames),
			intlist(f.FuncPcs),
			intlist(f.CallPcs),
		)
	}
	for call, count := range p.Calls {
		fmt.Fprintf(fout, "%v -> %v [calls=%d, weight=%f];\n",
			fids[call.Caller], fids[call.Callee],
			count, float64(count)/float64(p.CallCount))
	}
	fmt.Fprintln(fout, "}\n\n")
}
