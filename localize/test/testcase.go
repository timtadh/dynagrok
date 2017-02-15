package test

import (
	"hash/fnv"
)

type Testcase struct {
	Remote   *Remote
	Case     []byte
	executed bool
	ok       bool
	profile  []byte
}

func Test(r *Remote, stdin []byte) *Testcase {
	return &Testcase{
		Remote: r,
		Case: stdin,
	}
}

func (t *Testcase) String() string {
	if t == nil {
		return ""
	}
	return string(t.Case)
}

func (t *Testcase) Failed() bool {
	if !t.executed {
		panic("failed called before execute")
	}
	return !t.ok
}

func (t *Testcase) Usable() bool {
	if !t.executed {
		panic("usable called before execute")
	}
	return len(t.profile) > 0
}

func (t *Testcase) Profile() []byte {
	if !t.Usable() {
		panic("can't get the profile for this test case")
	}
	return t.profile
}

func (t *Testcase) Hash() int {
	h := fnv.New64a()
	h.Write(t.Case)
	return int(h.Sum64())
}

func (t *Testcase) Execute() error {
	if t.executed {
		return nil
	}
	_, _, profile, fails, ok, err := t.Remote.Execute(nil, t.Case)
	if err != nil {
		return err
	}
	t.executed = true
	t.ok = ok && len(fails) <= 0
	t.profile = profile
	return nil
}

func (t *Testcase) MinimizingMuts() []func()*Testcase {
	type slice struct {
		i, j int
	}
	fromSlice := func(s slice) *Testcase {
		left := t.Case[:s.i]
		right := t.Case[s.j+1:]
		buf := make([]byte, len(left) + len(right))
		copy(buf[:len(left)], left)
		copy(buf[len(left):], left)
		return Test(t.Remote, buf)
	}
	// min := func(i, j int) int {
	// 	if i < j {
	// 		return i
	// 	}
	// 	return j
	// }
	// max := func(i, j int) int {
	// 	if i > j {
	// 		return i
	// 	}
	// 	return j
	// }
	slices := make([]slice, 0, 10)
	// prefixes
	// for i := 0; i < len(t.Case)-1; i++ {
	// 	slices = append(slices, slice{
	// 		i: 0,
	// 		j: i,
	// 	})
	// }
	// suffixes
	for i := 1; i < len(t.Case); i++ {
		slices = append(slices, slice{
			i: i,
			j: len(t.Case)-1,
		})
	}
	// blocks
	// for i := 1; i < len(t.Case); i++ {
	// 	end := min(
	// 		i+min(max(15, int(.1*float64(len(t.Case)))), 100),
	// 		len(t.Case))
	// 	for j := i+1; j < end; j++ {
	// 		slices = append(slices, slice{
	// 			i: i,
	// 			j: j,
	// 		})
	// 	}
	// }
	tests := make([]func()*Testcase, 0, len(slices))
	for _, s := range slices {
		tests = append(tests, func(s slice) func() *Testcase {
			return func() *Testcase {
				return fromSlice(s)
			}
		}(s))
	}
	return tests
}
