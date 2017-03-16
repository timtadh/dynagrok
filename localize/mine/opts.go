package mine

import (
	"fmt"
)

type Options struct {
	MaxEdges int
	MinEdges int
	MinFails int
}

type Option func(*Options)

func MinEdges(minEdges int) Option {
	if minEdges < 1 {
		panic(fmt.Errorf("MinEdges must be >= 1 (got %v)", minEdges))
	}
	return func(o *Options) {
		o.MinEdges = minEdges
	}
}

func MaxEdges(maxEdges int) Option {
	if maxEdges < 2 {
		panic(fmt.Errorf("MaxEdges must be >= 2 (got %v)", maxEdges))
	}
	return func(o *Options) {
		o.MaxEdges = maxEdges
	}
}

func MinFails(minFails int) Option {
	if minFails < 1 {
		panic(fmt.Errorf("minFails must be >= 1 (got %v)", minFails))
	}
	return func(o *Options) {
		o.MinFails = minFails
	}
}

