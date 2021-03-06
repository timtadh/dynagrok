package test

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
	buf := make([]byte, len(left)+len(right))
	copy(buf[:len(left)], left)
	copy(buf[len(left):], right)
	return Test(m.Test.From, m.Test.Exec, buf)
}

func (t *Testcase) safe(i int) int {
	if i >= len(t.Case) {
		i = len(t.Case) - 1
	}
	if i < 0 {
		i = 0
	}
	return i
}

func (t *Testcase) LineEndTrimmingMuts() []*Mutant {
	if len(t.Case) <= 0 {
		return nil
	}
	lines := t.Lines()
	muts := make([]*Mutant, 0, len(lines))
	for _, i := range lines {
		if t.safe(i+1) >= len(t.Case)-1 {
			continue
		}
		muts = append(muts, &Mutant{
			Test: t,
			I:    t.safe(i + 1),
			J:    len(t.Case) - 1,
		})
	}
	return muts
}

func (t *Testcase) LineStartTrimmingMuts() []*Mutant {
	if len(t.Case) <= 0 {
		return nil
	}
	lines := t.Lines()
	muts := make([]*Mutant, 0, len(lines))
	for _, i := range lines {
		muts = append(muts, &Mutant{
			Test: t,
			I:    0,
			J:    i,
		})
	}
	return muts
}

func (t *Testcase) LineTrimmingMuts() []*Mutant {
	if len(t.Case) <= 0 {
		return nil
	}
	lines := t.Lines()
	muts := make([]*Mutant, 0, len(lines))
	for idx := 0; idx < len(lines)-1; idx++ {
		i := lines[idx]
		if i > 0 && t.Case[i] == '\n' {
			i = t.safe(i + 1)
		}
		j := lines[idx+1]
		if i > j {
			continue
		}
		muts = append(muts, &Mutant{
			Test: t,
			I:    i,
			J:    j,
		})
	}
	return muts
}

func (t *Testcase) LineBlockTrimmingMuts() []*Mutant {
	if len(t.Case) <= 0 {
		return nil
	}
	lines := t.Lines()
	muts := make([]*Mutant, 0, len(lines))
	for sIdx := 0; sIdx < len(lines); sIdx++ {
		end := min(
			sIdx+min(max(15, int(.1*float64(len(lines)))), 100),
			len(lines))
		for eIdx := sIdx + 1; eIdx < end; eIdx++ {
			i := lines[sIdx]
			if t.Case[i] == '\n' {
				i = t.safe(i + 1)
			}
			j := lines[eIdx]
			if i+1 >= j {
				continue
			}
			muts = append(muts, &Mutant{
				Test: t,
				I:    i,
				J:    j,
			})
		}
	}
	return muts
}

func (t *Testcase) BlockTrimmingMuts() []*Mutant {
	if len(t.Case) <= 0 {
		return nil
	}
	muts := make([]*Mutant, 0, len(t.Case))
	for i := 0; i < len(t.Case); i++ {
		end := min(
			i+min(max(15, int(.1*float64(len(t.Case)))), 10),
			len(t.Case))
		for j := i; j < end; j++ {
			muts = append(muts, &Mutant{
				Test: t,
				I:    i,
				J:    j,
			})
		}
	}
	return muts
}

func (t *Testcase) CurlyBlocks() []*Mutant {
	pop := func(items []int) ([]int, int) {
		return items[:len(items)-1], items[len(items)-1]
	}
	lefts := make([]int, 0, 10)
	blocks := make([]*Mutant, 0, 10)
	for i, c := range t.Case {
		switch c {
		case '{':
			lefts = append(lefts, i)
		case '}':
			if len(lefts) > 0 {
				var open int
				lefts, open = pop(lefts)
				blocks = append(blocks, &Mutant{
					Test: t,
					I:    open,
					J:    i,
				})
			}
		}
	}
	return blocks
}

func (t *Testcase) LineCurlyBlocks() []*Mutant {
	pop := func(items []int) ([]int, int) {
		return items[:len(items)-1], items[len(items)-1]
	}
	lefts := make([]int, 0, 10)
	blocks := make([]int, 0, 10)
	muts := make([]*Mutant, 0, 10)
	lastNl := 0
	for i, c := range t.Case {
		if i > 0 && t.Case[i-1] == '\n' {
			lastNl = i
		}
		switch c {
		case '\n':
			for _, openNl := range blocks {
				muts = append(muts, &Mutant{
					Test: t,
					I:    openNl,
					J:    i,
				})
			}
			blocks = blocks[:0]
		case '{':
			lefts = append(lefts, lastNl)
		case '}':
			if len(lefts) > 0 {
				var openNl int
				lefts, openNl = pop(lefts)
				blocks = append(blocks, openNl)
			}
		}
	}
	return muts
}

func (t *Testcase) Lines() []int {
	if t.lines != nil {
		return t.lines
	}
	lines := make([]int, 0, 10)
	if len(t.Case) > 0 {
		lines = append(lines, 0)
	}
	for i, c := range t.Case {
		if c == '\n' {
			lines = append(lines, i)
		}
	}
	if len(t.Case) > 0 && lines[len(lines)-1] != len(t.Case)-1 {
		lines = append(lines, len(t.Case)-1)
	}
	t.lines = lines
	return lines
}
