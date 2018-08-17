package fault

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/mutate"
)

type Fault struct {
	Position     string
	FnName       string
	BasicBlockId int
}

func (f *Fault) String() string {
	return fmt.Sprintf(`Fault {
    Position: %v,
    FnName: %v,
    BasicBlockId: %d,
}`, f.Position, f.FnName, f.BasicBlockId)
}

func (f *Fault) Equals(o *Fault) bool {
	return f.Position == o.Position && f.FnName == o.FnName && f.BasicBlockId == o.BasicBlockId
}

func LoadFault(bits []byte) (*Fault, error) {
	var e mutate.ExportedMut
	err := json.Unmarshal(bits, &e)
	if err != nil {
		return nil, err
	}
	f := &Fault{FnName: e.FnName, BasicBlockId: e.BasicBlockId, Position: e.SrcPosition.String()}
	return f, nil
}

func LoadD4JFault(bits []byte) (*Fault, error) {
	parts := strings.Split(string(bits), "#")
	path := parts[0]
	line, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	f := &Fault{
		Position: fmt.Sprintf("%v:%d", path, line),
	}
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

func LoadD4JFaults(path string) ([]*Fault, error) {
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
		f, err := LoadD4JFault(line)
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
