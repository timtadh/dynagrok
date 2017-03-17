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
	Action Action
	shortOpts string
	longOpts []string
	name string
	shortMsg string
	message string
}

type Sequence struct {
	runners []Runnable
}

type Alternatives struct {
	runners map[string]Runnable
}

func Cmd(name, shortMsg, msg, shortOpts string, longOpts []string, act Action) Runnable {
	return &Command{
		Action: act,
		shortOpts: shortOpts,
		longOpts: longOpts,
		name: strings.TrimSpace(name),
		shortMsg: strings.TrimSpace(shortMsg),
		message: strings.TrimSpace(msg),
	}
}

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
		name: "<unnamed-action>",
	}
}

func (c *Command) Run(argv []string) ([]string, *Error) {
	args, optargs, err := getopt.GetOpt(argv, c.ShortOpts(), c.LongOpts())
	if err != nil {
		return nil, Usage(c, -1, "could not process args: %v", err)
	}
	for _, oa := range optargs {
		switch oa.Opt() {
		case "-h", "--help":
			return nil, Usage(c, 0)
		}
	}
	return c.Action(c, args, optargs)
}

func (c *Command) ShortOpts() string {
	short := c.shortOpts
	if !strings.Contains(short, "h") {
		short += "h"
	}
	return short
}

func (c *Command) LongOpts() []string {
	has := func(list []string, s string) bool {
		for _, item := range list {
			if s == item {
				return true
			}
		}
		return false
	}
	long := c.longOpts
	if !has(long, "help") {
		long = append(long, "help")
	}
	return long
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

func (s *Sequence) Run(argv []string) ([]string, *Error) {
	_, optargs, err := getopt.GetOpt(argv, s.ShortOpts(), s.LongOpts())
	if err != nil {
		return nil, Usage(s, -1, "could not process args: %v", err)
	}
	for _, oa := range optargs {
		switch oa.Opt() {
		case "-h", "--help":
			return nil, Usage(s, 0)
		}
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
		return nil, Usage(a, -1, "Expected one of %v got %v", a.Name(), argv[0])
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


