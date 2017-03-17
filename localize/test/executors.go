package test

import (
	"io/ioutil"
	"os"
	"strings"
)

import (
	"github.com/google/shlex"
	"github.com/timtadh/data-structures/errors"
)

type Executor interface {
	Execute(test []byte) (stdout, stderr, profile, failures []byte, ok bool, err error)
}

type stdin struct {
	args []string
	r    *Remote
}

func StdinExecutor(args []string, r *Remote) Executor {
	return &stdin{args, r}
}

func (s *stdin) Execute(test []byte) (stdout, stderr, profile, failures []byte, ok bool, err error) {
	return s.r.Execute(s.args, test)
}

type singleInput struct {
	args   Arguments
	inputs []string
	stdin  bool
	r      *Remote
}

func SingleInputExecutor(args Arguments, r *Remote) (Executor, error) {
	if len(args.Inputs()) > 1 {
		return nil, errors.Errorf("multple input args supplied to the SingleInputExecutor %v", args)
	}
	_, stdin := args.Stdin()
	si := &singleInput{
		args:   args,
		inputs: args.Inputs(),
		stdin:  stdin,
		r:      r,
	}
	return si, nil
}

func (s *singleInput) Execute(test []byte) (stdout, stderr, profile, failures []byte, ok bool, err error) {
	if s.stdin {
		return s.r.Execute(s.args.Render(nil), test)
	} else {
		tf, err := ioutil.TempFile("", "dynagrok-test-input-")
		if err != nil {
			return nil, nil, nil, nil, false, err
		}
		defer os.Remove(tf.Name())
		_, err = tf.Write(test)
		tf.Close()
		if err != nil {
			return nil, nil, nil, nil, false, err
		}
		inputs := map[string]string{
			s.inputs[0]: tf.Name(),
		}
		return s.r.Execute(s.args.Render(inputs), nil)
	}
}

func ParseArgs(s string) (Arguments, error) {
	split, err := shlex.Split(s)
	if err != nil {
		return nil, err
	}
	args := make(Arguments, 0, len(split))
	for _, a := range split {
		args = append(args, Arg(a))
	}
	errors.Logf("DEBUG", "args %v", args)
	return args, nil
}

type Arguments []Argument

func (args Arguments) Render(inputs map[string]string) []string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if !arg.Stdin() {
			parts = append(parts, arg.Render(inputs))
		}
	}
	return parts
}

func (args Arguments) Inputs() []string {
	inputs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg.Input() {
			inputs = append(inputs, arg.InputName())
		}
	}
	return inputs
}

func (args Arguments) Stdin() (string, bool) {
	for _, arg := range args {
		if arg.Stdin() {
			return arg.InputName(), true
		}
	}
	return "", false
}

type Argument interface {
	Render(inputs map[string]string) string
	Stdin() bool
	Input() bool
	InputName() string
}

func Arg(arg string) Argument {
	if strings.HasPrefix(arg, "<$") {
		name := arg[2:]
		return &stdinArg{name}
	} else if strings.HasPrefix(arg, "$") {
		name := arg[1:]
		return &inputArg{name}
	}
	return &literalArg{arg}
}

type stdinArg struct {
	name string
}

func (a *stdinArg) Render(inputs map[string]string) string {
	panic("this is a stdin arg")
}

func (a *stdinArg) Stdin() bool {
	return true
}

func (a *stdinArg) Input() bool {
	return true
}

func (a *stdinArg) InputName() string {
	return a.name
}

func (a *stdinArg) String() string {
	return "<$" + a.name
}

type inputArg struct {
	name string
}

func (a *inputArg) Render(inputs map[string]string) string {
	return inputs[a.name]
}

func (a *inputArg) Stdin() bool {
	return false
}

func (a *inputArg) Input() bool {
	return true
}

func (a *inputArg) InputName() string {
	return a.name
}

func (a *inputArg) String() string {
	return "$" + a.name
}

type literalArg struct {
	arg string
}

func (a *literalArg) Render(inputs map[string]string) string {
	return a.arg
}

func (a *literalArg) Stdin() bool {
	return false
}

func (a *literalArg) Input() bool {
	return false
}

func (a *literalArg) InputName() string {
	panic("this is a literal arg")
}

func (a *literalArg) String() string {
	return a.arg
}
