package fault

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/mutate"
)

type Fault struct {
	FnName       string
	BasicBlockId int
}

func (f *Fault) String() string {
	return fmt.Sprintf(`Fault {
    FnName: %v,
    BasicBlockId: %d,
}`, f.FnName, f.BasicBlockId)
}

func (f *Fault) Equals(o *Fault) bool {
	return f.FnName == o.FnName && f.BasicBlockId == o.BasicBlockId
}

func LoadFault(bits []byte) (*Fault, error) {
	var e mutate.ExportedMut
	err := json.Unmarshal(bits, &e)
	if err != nil {
		return nil, err
	}
	f := &Fault{FnName: e.FnName, BasicBlockId: e.BasicBlockId}
	return f, nil
}

func LoadFaults(path string) ([]*Fault, error) {
	fin, failClose, err := cmd.Input(path)
	if err != nil {
		return nil, fmt.Errorf("Could not read the list of failures: %v\n%v", path, err)
	}
	defer failClose()
	seen := make(map[Fault]bool)
	failures := make([]*Fault, 0, 10)
	s := bufio.NewScanner(fin)
	for s.Scan() {
		line := bytes.TrimSpace(s.Bytes())
		if len(line) == 0 {
			continue
		}
		f, err := LoadFault(line)
		if err != nil {
			return nil, fmt.Errorf("Could not load failure: `%v`\nerror: %v", string(line), err)
		}
		if !seen[*f] {
			seen[*f] = true
			failures = append(failures, f)
		}
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("Could not read the failures file: %v, error: %v", path, err)
	}
	return failures, nil
}