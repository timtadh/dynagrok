package lattice

import (
	"fmt"
)

import ()

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
)



func Load(failPath, okPath string) (l *Lattice, err error) {
	return NewLattice(func(l *Lattice) error {
		failFile, failClose, err := cmd.Input(failPath)
		if err != nil {
			return fmt.Errorf("Could not read profiles from failed executions: %v\n%v", failPath, err)
		}
		defer failClose()
		fail, err := digraph.LoadDot(l.Positions, l.FnNames, l.BBIds, l.Labels, failFile)
		if err != nil {
			return fmt.Errorf("Could not load profiles from failed executions: %v\n%v", failPath, err)
		}
		okFile, okClose, err := cmd.Input(okPath)
		if err != nil {
			return fmt.Errorf("Could not read profiles from successful executions: %v\n%v", okPath, err)
		}
		defer okClose()
		ok, err := digraph.LoadDot(l.Positions, l.FnNames, l.BBIds, l.Labels, okFile)
		if err != nil {
			return fmt.Errorf("Could not load profiles from successful executions: %v\n%v", okPath, err)
		}
		l.Fail = fail
		l.Ok = ok
		return nil
	})
}
