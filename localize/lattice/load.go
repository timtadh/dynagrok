package lattice

import (
	"fmt"
	"io"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
)

func Load(failPath, okPath []string) (l *Lattice, err error) {
	failFile, failClose, err := cmd.Inputs(failPath)
	if err != nil {
		return nil, fmt.Errorf("Could not read profiles from failed executions: %v\n%v", failPath, err)
	}
	defer failClose()
	okFile, okClose, err := cmd.Inputs(okPath)
	if err != nil {
		return nil, fmt.Errorf("Could not read profiles from successful executions: %v\n%v", okPath, err)
	}
	defer okClose()
	return LoadFrom(failFile, okFile)
}

func LoadDot(failPath, okPath []string) (l *Lattice, err error) {
	failFile, failClose, err := cmd.Inputs(failPath)
	if err != nil {
		return nil, fmt.Errorf("Could not read profiles from failed executions: %v\n%v", failPath, err)
	}
	defer failClose()
	okFile, okClose, err := cmd.Inputs(okPath)
	if err != nil {
		return nil, fmt.Errorf("Could not read profiles from successful executions: %v\n%v", okPath, err)
	}
	defer okClose()
	return LoadFromDot(failFile, okFile)
}

func LoadFrom(failFile, okFile io.Reader) (l *Lattice, err error) {
	return NewLattice(func(l *Lattice) error {
		fail, err := digraph.LoadSimple(l.Info, l.Labels, failFile)
		if err != nil {
			return fmt.Errorf("Could not load profiles from failed executions\n%v", err)
		}
		ok, err := digraph.LoadSimple(l.Info, l.Labels, okFile)
		if err != nil {
			return fmt.Errorf("Could not load profiles from successful executions\n%v", err)
		}
		l.Fail = fail
		l.Ok = ok
		return nil
	})
}

func LoadFromDot(failFile, okFile io.Reader) (l *Lattice, err error) {
	return NewLattice(func(l *Lattice) error {
		fail, err := digraph.LoadDot(l.Info, l.Labels, l.NodeAttrs, failFile)
		if err != nil {
			return fmt.Errorf("Could not load profiles from failed executions\n%v", err)
		}
		ok, err := digraph.LoadDot(l.Info, l.Labels, l.NodeAttrs, okFile)
		if err != nil {
			return fmt.Errorf("Could not load profiles from successful executions\n%v", err)
		}
		l.Fail = fail
		l.Ok = ok
		return nil
	})
}
