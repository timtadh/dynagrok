package test

import ()

type Mutant struct {
	Test *Testcase
	I, J int
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *Mutant) Testcase() *Testcase {
	left := m.Test.Case[:m.I]
	right := m.Test.Case[m.J+1:]
	buf := make([]byte, len(left) + len(right))
	copy(buf[:len(left)], left)
	copy(buf[len(left):], left)
	return Test(m.Test.Remote, buf)
}

func (t *Testcase) MinimizingMuts() []*Mutant {
	muts := make([]*Mutant, 0, 10)
	// suffixes
	muts = append(muts, t.LineEndTrimmingMuts()...)
	// prefixes
	// for i := 0; i < len(t.Case)-1; i++ {
	// 	slices = append(slices, slice{
	// 		i: 0,
	// 		j: i,
	// 	})
	// }
	// blocks
	// for i := 1; i < len(t.Case); i++ {
	// 	end := min(
	// 		i+min(max(15, int(.1*float64(len(t.Case)))), 100),
	// 		len(t.Case))
	// 	for j := i+1; j < end; j++ {
	// 		slices = append(slices, slice{
	// 			i: i,
	// 			j: j,
	// 		})
	// 	}
	// }
	return muts
}

func (t *Testcase) EndTrimmingMuts() []*Mutant {
	muts := make([]*Mutant, 0, len(t.Case))
	for i := 1; i < len(t.Case); i++ {
		muts = append(muts, &Mutant{
			Test: t,
			I: i,
			J: len(t.Case)-1,
		})
	}
	return muts
}

func (t *Testcase) LineEndTrimmingMuts() []*Mutant {
	lines := t.Lines()
	muts := make([]*Mutant, 0, len(lines))
	for _, i := range lines {
		muts = append(muts, &Mutant{
			Test: t,
			I: i,
			J: len(t.Case)-1,
		})
	}
	return muts
}

func (t *Testcase) LineStartTrimmingMuts() []*Mutant {
	lines := t.Lines()
	muts := make([]*Mutant, 0, len(lines))
	for _, i := range lines {
		muts = append(muts, &Mutant{
			Test: t,
			I: 0,
			J: i,
		})
	}
	return muts
}

func (t *Testcase) LineBlockTrimmingMuts() []*Mutant {
	safe := func(i int) int {
		for i > len(t.Case) {
			i--
		}
		if i < 0 {
			i = 0
		}
		return i
	}
	lines := t.Lines()
	muts := make([]*Mutant, 0, len(lines))
	for sIdx := 1; sIdx < len(lines); sIdx++ {
		end := min(
			sIdx+min(max(15, int(.1*float64(len(lines)))), 100),
			len(lines))
		for eIdx := sIdx+1; eIdx < end; eIdx++ {
			i := safe(lines[sIdx] + 1)
			j := safe(lines[eIdx] - 1)
			muts = append(muts, &Mutant{
				Test: t,
				I: i,
				J: j,
			})
		}
	}
	return muts
}

func (t *Testcase) BlockTrimmingMuts() []*Mutant {
	muts := make([]*Mutant, 0, len(t.Case))
	for i := 1; i < len(t.Case); i++ {
		end := min(
			i+min(max(15, int(.1*float64(len(t.Case)))), 100),
			len(t.Case))
		for j := i+1; j < end; j++ {
			muts = append(muts, &Mutant{
				Test: t,
				I: i,
				J: j,
			})
		}
	}
	return muts
}

func (t *Testcase) Lines() []int {
	lines := make([]int, 0, 10)
	for i, c := range t.Case {
		if c == '\n' {
			lines = append(lines, i)
		}
	}
	return lines
}
