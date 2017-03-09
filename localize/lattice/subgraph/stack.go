package subgraph

import (
	"sync"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import ()

type embSearchNode struct {
	emb *Embedding
	eid int
}

type Stack struct {
	mu sync.Mutex
	cond  *sync.Cond
	stacks [][]embSearchNode
	mux []sync.Mutex
	threads int
	waiting int
	closed bool
}

func NewStack(expectedThreads int) *Stack {
	s := &Stack{
		stacks: make([][]embSearchNode, 0, expectedThreads + 1),
		mux: make([]sync.Mutex, expectedThreads + 1, expectedThreads + 1),
	}
	s.stacks = append(s.stacks, make([]embSearchNode, 0, 100))
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *Stack) AddThread() int {
	s.mu.Lock()
	tid := s.threads + 1
	if tid >= len(s.mux) {
		errors.Logf("DEBUG", "tid %v, len(mux) %v, len(stacks) %v", tid, len(s.mux), len(s.stacks))
		panic("more threads than expected!!")
	}
	s.mux[tid].Lock()
	s.stacks = append(s.stacks, make([]embSearchNode, 0, 100))
	s.mux[0].Lock()
	for len(s.stacks[0])/len(s.stacks) > 0 {
		s.stacks[tid] = append(s.stacks[tid], s.stacks[0][len(s.stacks[0]) - 1])
		s.stacks[0] = s.stacks[0][:len(s.stacks[0])-1]
	}
	s.mux[0].Unlock()
	s.mux[tid].Unlock()
	s.threads++
	s.mu.Unlock()
	return tid
}

func (s *Stack) Close() {
	s.mu.Lock()
	for i := 0; i < len(s.mux); i++ {
		s.mux[i].Lock()
	}
	s.closed = true
	s.stacks = nil
	for i := len(s.mux) - 1; i >= 0; i-- {
		s.mux[i].Unlock()
	}
	s.mu.Unlock()
	s.cond.Broadcast()
}

func (s *Stack) Closed() bool {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	return closed
}

func (s *Stack) WaitClosed() {
	s.mu.Lock()
	for !s.closed {
		s.cond.Wait()
	}
	s.mu.Unlock()
}

func (s *Stack) Push(tid int, emb *Embedding, eid int) {
	if len(s.stacks) < tid {
		tid = 0
	}
	if false {
		errors.Logf("DEBUG", "tid %v, len(mux) %v, len(stacks) %v", tid, len(s.mux), len(s.stacks))
	}
	s.mux[tid].Lock()
	if s.closed {
		s.mux[tid].Unlock()
		return
	}
	s.stacks[tid] = append(s.stacks[tid], embSearchNode{emb, eid})
	s.mux[tid].Unlock()
	s.cond.Broadcast()
}

func (s *Stack) Pop(tid int) (emb *Embedding, eid int) {
	if len(s.stacks) < tid {
		tid = 0
	}
	s.mux[tid].Lock()
	if s.closed {
		s.mux[tid].Unlock()
		return nil, 0
	}

	// try a local pop first
	if len(s.stacks[tid]) > 0 {
		item := s.stacks[tid][len(s.stacks[tid])-1]
		s.stacks[tid] = s.stacks[tid][:len(s.stacks[tid])-1]
		s.mux[tid].Unlock()
		return item.emb, item.eid
	}

	// local is empty we need to steal
	s.mux[tid].Unlock()

	for {
		// try a steal
		for i := 0; i < len(s.stacks); i++ {
			s.mux[i].Lock()
			if i < len(s.stacks) && len(s.stacks[i]) > 0 {
				item := s.stacks[i][len(s.stacks[i])-1]
				s.stacks[i] = s.stacks[i][:len(s.stacks[i])-1]
				s.mux[i].Unlock()
				return item.emb, item.eid
			}
			s.mux[i].Unlock()
		}

		// steal failed; wait for a broadcast of a Push
		s.mu.Lock()
		s.waiting++
		if (s.threads > 0 && s.threads == s.waiting) || s.closed {
			s.mu.Unlock()
			s.Close()
			s.cond.Broadcast()
			return nil, 0
		}
		s.cond.Wait()
		s.waiting--
		s.mu.Unlock()
	}
}
