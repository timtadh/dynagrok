package cmd

import (
	"fmt"
)


type Error struct {
	Err error
	ExitCode int
}

func Err(code int, err error) *Error {
	return &Error{Err: err, ExitCode: code}
}

func Errorf(code int, format string, args ...interface{}) *Error {
	return &Error{Err: fmt.Errorf(format, args...), ExitCode: code}
}

func Usage(cmd Runnable, code int, format_and_args ...interface{}) *Error {
	var err error
	if len(format_and_args) > 0 {
		format := format_and_args[0].(string)
		args := format_and_args[1:]
		err = fmt.Errorf("error: %v\n\n%v\n", fmt.Sprintf(format, args...), cmd.ShortUsage())
	} else {
		err = fmt.Errorf("%v\n\n%v\n", cmd.ShortUsage(), cmd.Usage())
	}
	return &Error{Err: err, ExitCode: code}
}

func (c *Error) Error() string {
	return c.Err.Error()
}

func (c *Error) String() string {
	return c.Err.Error()
}


