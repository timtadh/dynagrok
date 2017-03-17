package test

import (
	"bytes"
	"hash/fnv"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
)

type Testcase struct {
	From     string
	Case     []byte
	Exec     Executor
	executed bool
	ok       bool
	profile  []byte
	lines    []int
}

func Test(from string, e Executor, stdin []byte) *Testcase {
	return &Testcase{
		From: from,
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
	return digraph.LoadSimple(l.Info, l.Labels, &buf)
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
	stdout, stderr, profile, fails, ok, err := t.Exec.Execute(t.Case)
	if err != nil {
		return err
	}
	if false {
		errors.Logf("DEBUG", "stdout\n%v\n-------------\n", string(stdout))
		errors.Logf("DEBUG", "stderr\n%v\n-------------\n", string(stderr))
	}
	t.executed = true
	t.ok = ok && len(fails) <= 0
	t.profile = profile
	if false {
		errors.Logf("INFO", "executed %v %v %v %v %v", len(t.Case), len(profile) > 0, len(fails), ok, t.ok)
	}
	return nil
}

func (t *Testcase) ExecuteWith(e Executor) (stdout, stderr, profile, failures []byte, ok bool, err error) {
	return e.Execute(t.Case)
}
