package digraph

import (
	"io"
	"bufio"
	"strconv"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import ()


type SimpleLoader struct {
	Builder   *Builder
	Labels    *Labels
	Positions map[int]string
	FnNames   map[int]string
	BBIds     map[int]int
	vidxs     map[int]int
}

func LoadSimple(positions, fnNames map[int]string, bbids map[int]int, labels *Labels, input io.Reader) (*Indices, error) {
	l := &SimpleLoader{
		Builder: Build(100, 1000),
		Labels: labels,
		Positions: positions,
		FnNames: fnNames,
		BBIds: bbids,
		vidxs: make(map[int]int),
	}
	return l.load(input)
}

func (l *SimpleLoader) load(input io.Reader) (*Indices, error) {
	graph := 0
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		split := strings.SplitN(line, "\t", 2)
		kind, rest := split[0], split[1:]
		switch kind {
		case "start-graph":
		case "end-graph":
			graph++
		case "vertex":
			err := l.vertex(rest)
			if err != nil {
				return nil, err
			}
		case "edge":
			err := l.edge(rest)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.Errorf("Unexpected kind `%v` for line `%v`", kind, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	l.Builder.Graphs = graph
	return NewIndices(l.Builder, 0), nil
}

func (l *SimpleLoader) vertex(rest []string) (error) {
	if len(rest) != 1 {
		return errors.Errorf("line in unexpected format: `%v`", rest)
	}
	tokens, err := l.tokens(rest[0])
	if err != nil {
		return err
	}
	if len(tokens) != 5 {
		return errors.Errorf("line in unexpected format (expected 5 tokens): `%v`", tokens)
	}
	id, err := strconv.Atoi(tokens[0])
	if err != nil {
		return err
	}
	label, err := strconv.Unquote(tokens[1])
	if err != nil {
		return err
	}
	bbid, err := strconv.Atoi(tokens[2])
	if err != nil {
		return err
	}
	fnName, err := strconv.Unquote(tokens[3])
	if err != nil {
		return err
	}
	pos, err := strconv.Unquote(tokens[4])
	if err != nil {
		return err
	}
	l.addVertex(id, l.Labels.Color(label), bbid, fnName, pos)
	return nil
}

func (l *SimpleLoader) edge(rest []string) (error) {
	if len(rest) != 1 {
		return errors.Errorf("line in unexpected format: `%v`", rest)
	}
	tokens, err := l.tokens(rest[0])
	if err != nil {
		return err
	}
	if len(tokens) != 3 {
		return errors.Errorf("line in unexpected format (expected 3 tokens): `%v`", tokens)
	}
	src, err := strconv.Atoi(tokens[0])
	if err != nil {
		return err
	}
	targ, err := strconv.Atoi(tokens[1])
	if err != nil {
		return err
	}
	return l.addEdge(src, targ)
}

func (l *SimpleLoader) tokens(s string) ([]string, error) {
	buf := make([]rune, 0, len(s))
	parts := make([]string, 0, 6)
	quotes := false
	backslash := false
	for _, c := range s {
		switch c {
		case '"':
			if !backslash {
				quotes = !quotes
			}
		case ',':
			if !backslash && !quotes {
				parts = append(parts, strings.TrimSpace(string(buf)))
				buf = buf[:0]
				continue
			}
		}
		if c == '\\' {
			backslash = !backslash
		} else if backslash {
			backslash = false
		}
		buf = append(buf, c)
	}
	if backslash {
		return nil, errors.Errorf("unfinished backslash: `%v`", s)
	}
	if quotes {
		return nil, errors.Errorf("unclosed quote: `%v`", s)
	}
	if len(buf) > 0 {
		parts = append(parts, strings.TrimSpace(string(buf)))
	}
	return parts, nil
}

func (l *SimpleLoader) addVertex(id, color, bbid int, fnName, pos string) {
	vertex := l.Builder.AddVertex(color)
	l.vidxs[id] = vertex.Idx
	l.Positions[color] = pos
	l.FnNames[color] = fnName
	l.BBIds[color] = bbid
}

func (l *SimpleLoader) addEdge(sid, tid int) error {
	if sidx, has := l.vidxs[sid]; !has {
		return errors.Errorf("unknown src id %v", tid)
	} else if tidx, has := l.vidxs[tid]; !has{
		return errors.Errorf("unknown targ id %v", tid)
	} else {
		l.Builder.AddEdge(&l.Builder.V[sidx], &l.Builder.V[tidx], l.Labels.Color(""))
	}
	return nil
}

