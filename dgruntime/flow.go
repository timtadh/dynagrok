package dgruntime

import (
	"fmt"
)

type Flow []BlkEntrance

type FlowEdge struct {
	Src BlkEntrance
	Targ BlkEntrance
}

type BlkEntrance struct {
	In uintptr
	BlkId, At int
}

func (a Flow) equals(b Flow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].equals(&b[i]) {
			return false
		}
	}
	return true
}

func (a *BlkEntrance) equals(b *BlkEntrance) bool {
	return a.BlkId == b.BlkId && a.At == b.At
}

func (f Flow) String() string {
	str := "["
	for i := range f {
		str += f[i].String()
		if i+1 < len(f) {
			str += ", "
		}
	}
	str += "]"
	return str
}

func (b *BlkEntrance) String() string {
	if b.At == 0 {
		return fmt.Sprintf("{blk: %d}", b.BlkId)
	}
	return fmt.Sprintf("{blk: %d, at: %d}", b.BlkId, b.At)
}
