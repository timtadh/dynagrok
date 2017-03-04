package test

import (
	"bytes"
	"hash/fnv"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
)

type Testcase struct {
	Case     []byte
	Exec     Executor
	executed bool
	ok       bool
	profile  []byte
}

func Test(e Executor, stdin []byte) *Testcase {
	return &Testcase{
		Case: stdin,
		Exec: e,
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

func (t *Testcase) Digraph(l *lattice.Lattice) (*digraph.Indices, error) {
	var buf bytes.Buffer
	_, err := buf.Write(t.Profile())
	if err != nil {
		return nil, err
	}
	return digraph.LoadDot(l.Positions, l.FnNames, l.BBIds, l.Labels, &buf)
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
	_, _, profile, fails, ok, err := t.Exec.Execute(t.Case)
	if err != nil {
		return err
	}
	t.executed = true
	t.ok = ok && len(fails) <= 0
	t.profile = profile
	return nil
}

func (t *Testcase) ExecuteWith(remote *Remote) (stdout, stderr, profile, failures []byte, ok bool, err error) {
	return remote.Execute(nil, t.Case)
}

