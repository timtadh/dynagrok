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

func (t *Testcase) MinimizingMuts() []*Testcase {
	return nil
}
