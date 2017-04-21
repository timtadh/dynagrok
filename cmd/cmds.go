package cmd

import (
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/getopt"
)

type Runnable interface {
	Run(argv []string) ([]string, *Error)
	ShortOpts() string
	LongOpts() []string
	Name() string
	ShortUsage() string
	Usage() string
}

type Action func(r Runnable, argv []string, optargs []getopt.OptArg) ([]string, *Error)

type Command struct {
	Action    Action
	shortOpts string
	longOpts  []string
	name      string
	shortMsg  string
	message   string
}

type Annotation struct {
	name     string
	shortMsg string
	message  string
	runner   Runnable
}

type Sequence struct {
	runners []Runnable
}

type Partial struct {
	name    string
	runners []Runnable
}

type Alternatives struct {
	runners map[string]Runnable
}

func Cmd(name, shortMsg, msg, shortOpts string, longOpts []string, act Action) Runnable {
	return &Command{
		Action:    act,
		shortOpts: shortOpts,
		longOpts:  longOpts,
		name:      strings.TrimSpace(name),
		shortMsg:  strings.TrimSpace(shortMsg),
		message:   strings.TrimSpace(msg),
	}
}

func Annotate(r Runnable, name, beforeShort, afterShort, before, after string) Runnable {
	return &Annotation{
		name: name,
		shortMsg: strings.Join([]string{
			strings.TrimSpace(beforeShort),
			strings.TrimSpace(r.ShortUsage()),
			strings.TrimSpace(afterShort),
		}, " "),
		message: strings.Join([]string{
			before,
			r.Usage(),
			after,
		}, "\n"),
		runner: r,
	}
}

// Join takes multiple runners. Each is assumed to be partial parser
// for a larger list of options. Each runner is given just those
// parsed options ([]getopt.OptArg) which are specified by their
// ShortOpts() and LongOpts(). The last runner given
// also gets to parse the left over arguments.
func Join(name string, runners ...Runnable) Runnable {
	return &Partial{
		name:    name,
		runners: runners,
	}
}

// Concat has concatenates the parsers for each runner
// together. Each successive runner parses the arguments
// returned by the pervious runner.
func Concat(runners ...Runnable) Runnable {
	return &Sequence{
		runners: runners,
	}
}

func Commands(runners map[string]Runnable) Runnable {
	return &Alternatives{
		runners: runners,
	}
}

func BareCmd(act Action) Runnable {
	return &Command{
		Action: act,
		name:   "<unnamed-action>",
	}
}

func run(r Runnable, argv []string) ([]string, []getopt.OptArg, *Error) {
	has := func(list []string, s string) bool {
		for _, item := range list {
			if s == item {
				return true
			}
		}
		return false
	}
	short := r.ShortOpts()
	if !strings.Contains(short, "h") {
		short += "h"
	}
	long := r.LongOpts()
	if !has(long, "help") {
		long = append(long, "help")
	}
	args, optargs, err := getopt.GetOpt(argv, short, long)
	if err != nil {
		return nil, nil, Errorf(-1, "could not process args: %v", err)
	}
	for _, oa := range optargs {
		switch oa.Opt() {
		case "-h", "--help":
			return nil, nil, Usage(r, 0)
		}
	}
	return args, optargs, nil
}

func (c *Command) Run(argv []string) ([]string, *Error) {
	args, optargs, err := run(c, argv)
	if err != nil {
		return nil, err
	}
	return c.Action(c, args, optargs)
}

func (c *Command) ShortOpts() string {
	return c.shortOpts
}

func (c *Command) LongOpts() []string {
	return c.longOpts
}

func (c *Command) Name() string {
	return c.name
}

func (c *Command) ShortUsage() string {
	return fmt.Sprintf("%v %v", c.name, c.shortMsg)
}

func (c *Command) Usage() string {
	return c.message
}

func (a *Annotation) Run(argv []string) ([]string, *Error) {
	return a.runner.Run(argv)
}

func (a *Annotation) ShortOpts() string {
	return a.runner.ShortOpts()
}

func (a *Annotation) LongOpts() []string {
	return a.runner.LongOpts()
}

func (a *Annotation) Name() string {
	return a.name
}

func (a *Annotation) ShortUsage() string {
	return fmt.Sprintf("%v %v", a.name, strings.TrimSpace(a.shortMsg))
}

func (a *Annotation) Usage() string {
	return a.message
}

func (p *Partial) Run(argv []string) ([]string, *Error) {
	if len(p.runners) <= 0 {
		panic("no runners")
	}
	args, optargs, e := run(p, argv)
	if e != nil {
		return nil, e
	}
	myopts := func(r Runnable) []string {
		short := r.ShortOpts()
		long := r.LongOpts()
		has := func(opt string) bool {
			if len(opt) > 1 && strings.Contains(short, opt[1:]) {
				return true
			} else if len(opt) > 2 {
				for _, item := range long {
					if opt[2:] == strings.TrimRight(item, "=") {
						return true
					}
				}
			}
			return false
		}
		mine := make([]string, 0, len(optargs))
		for _, oa := range optargs {
			if has(oa.Opt()) {
				mine = append(mine, oa.Opt())
				if oa.Arg() != "" {
					mine = append(mine, oa.Arg())
				}
			}
		}
		return mine
	}
	for _, r := range p.runners[:len(p.runners)-1] {
		_, err := r.Run(myopts(r))
		if err != nil {
			return nil, err
		}
	}
	r := p.runners[len(p.runners)-1]
	argv, err := r.Run(append(myopts(r), args...))
	if err != nil {
		return nil, err
	}
	return argv, nil
}

func (p *Partial) Name() string {
	return p.name
}

func (p *Partial) ShortOpts() string {
	shortOpts := ""
	for _, r := range p.runners {
		shortOpts += r.ShortOpts()
	}
	return shortOpts
}

func (p *Partial) LongOpts() []string {
	var longOpts []string
	for _, r := range p.runners {
		longOpts = append(longOpts, r.LongOpts()...)
	}
	return longOpts
}

func (p *Partial) ShortUsage() string {
	shorts := make([]string, 0, len(p.runners))
	for _, r := range p.runners {
		u := r.ShortUsage()
		if len(u) > 0 {
			shorts = append(shorts, u)
		}
	}
	return strings.Join(shorts, " ")
}

func (p *Partial) Usage() string {
	longs := make([]string, 0, len(p.runners))
	for _, r := range p.runners {
		u := strings.TrimSpace(r.Usage())
		if len(u) > 0 {
			longs = append(longs, u)
		}
	}
	return indent(fmt.Sprintf("%v", strings.Join(longs, "\n")), 4)
}

func (s *Sequence) Run(argv []string) ([]string, *Error) {
	_, _, err := run(s, argv)
	if err != nil {
		return nil, err
	}
	for _, r := range s.runners {
		var err *Error
		argv, err = r.Run(argv)
		if err != nil {
			return nil, err
		}
	}
	return argv, nil
}

func (s *Sequence) Name() string {
	return s.runners[0].Name()
}

func (s *Sequence) ShortOpts() string {
	return s.runners[0].ShortOpts()
}

func (s *Sequence) LongOpts() []string {
	return s.runners[0].LongOpts()
}

func (s *Sequence) ShortUsage() string {
	shorts := make([]string, 0, len(s.runners))
	for _, r := range s.runners {
		if r.Name() != "<unnamed-action>" {
			shorts = append(shorts, r.ShortUsage())
		}
	}
	return strings.Join(shorts, " ")
}

func (s *Sequence) Usage() string {
	longs := make([]string, 0, len(s.runners))
	for _, r := range s.runners {
		longs = append(longs, r.Usage())
	}
	return fmt.Sprintf("%v", strings.Join(longs, "\n\n"))
}

func (a *Alternatives) Run(argv []string) ([]string, *Error) {
	if len(argv) == 0 {
		if r, has := a.runners[""]; has {
			return r.Run(argv)
		}
	}
	if len(argv) <= 0 {
		return nil, Usage(a, -1, "Expected one of %v got end of arguments", a.Name())
	}
	if r, has := a.runners[argv[0]]; !has {
		if r, has := a.runners[""]; has {
			return r.Run(argv)
		} else {
			return nil, Usage(a, -1, "Expected one of %v got %v", a.Name(), argv[0])
		}
	} else {
		return r.Run(argv[1:])
	}
}

func (a *Alternatives) Name() string {
	if len(a.runners) == 1 {
		for k := range a.runners {
			return k
		}
	}
	optional := ""
	keys := make([]string, 0, len(a.runners))
	for k := range a.runners {
		if k != "" {
			keys = append(keys, k)
		} else {
			optional = "?"
		}
	}
	return fmt.Sprintf("(%v)%v", strings.Join(keys, "|"), optional)
}

func (a *Alternatives) ShortOpts() string {
	return ""
}

func (a *Alternatives) LongOpts() []string {
	return nil
}

func (a *Alternatives) ShortUsage() string {
	return a.Name()
}

func (a *Alternatives) Usage() string {
	if len(a.runners) == 1 {
	}
	names := make([]string, 0, len(a.runners))
	longs := make([]string, 0, len(a.runners))
	for name, r := range a.runners {
		if name != "" {
			names = append(names, fmt.Sprintf("    %-15v", r.ShortUsage()))
			longs = append(longs, fmt.Sprintf("%v\n%v", r.ShortUsage(), indent(r.Usage(), 2)))
		} else {
			if strings.TrimSpace(r.ShortUsage()) != "<unnamed-action>" {
				longs = append(longs, fmt.Sprintf("%v\n%v", r.ShortUsage(), indent(r.Usage(), 2)))
			}
		}
	}
	if len(names) <= 1 {
		return fmt.Sprintf("%v", strings.Join(longs, "\n\n"))
	}
	return fmt.Sprintf("Commands\n%v\n\n%v",
		strings.Join(names, "\n"), indent(strings.Join(longs, "\n\n"), 2))
}

func keys(runners map[string]Runnable) []string {
	keys := make([]string, 0, len(runners))
	for k := range runners {
		keys = append(keys, k)
	}
	return keys
}

func indent(s string, spaces int) string {
	olines := strings.Split(s, "\n")
	nlines := make([]string, 0, len(olines))
	for _, line := range olines {
		nlines = append(nlines, fmt.Sprintf(fmt.Sprintf("%%-%dv%%v", 4), "", line))
	}
	return strings.Join(nlines, "\n")
}

func noindent(s string) string {
	olines := strings.Split(s, "\n")
	nlines := make([]string, 0, len(olines))
	for _, line := range olines {
		nlines = append(nlines, strings.TrimSpace(line))
	}
	return strings.Join(nlines, "\n")
}
