package test

type Executor interface {
	Execute(test []byte) (stdout, stderr, profile, failures []byte, ok bool, err error)
}

type stdin struct {
	args []string
	r *Remote
}

func StdinExecutor(args []string, r *Remote) Executor {
	return &stdin{args, r}
}

func (s *stdin) Execute(test []byte) (stdout, stderr, profile, failures []byte, ok bool, err error) {
	return s.r.Execute(s.args, test)
}
