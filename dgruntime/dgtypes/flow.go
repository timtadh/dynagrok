package dgtypes

import (
	"fmt"
)

type Flow []BlkEntrance

type FlowEdge struct {
	Src  BlkEntrance
	Targ BlkEntrance
}

type BlkEntrance struct {
	In           uintptr
	BasicBlockId int
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
	return a.BasicBlockId == b.BasicBlockId
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
	return fmt.Sprintf("{blk: %d}", b.BasicBlockId)
}
