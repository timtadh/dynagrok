package digraph

import (
	"fmt"
	"io"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/sfp/lattice"
	"github.com/timtadh/dynagrok/localize/digraph/subgraph"
)

type Formatter struct {
	g     *Digraph
	prfmt lattice.PrFormatter
}

func NewFormatter(g *Digraph, prfmt lattice.PrFormatter) *Formatter {
	return &Formatter{
		g:     g,
		prfmt: prfmt,
	}
}

func (f *Formatter) PrFormatter() lattice.PrFormatter {
	return f.prfmt
}

func (f *Formatter) FileExt() string {
	return ".dot"
}

func (f *Formatter) PatternName(node lattice.Node) string {
	switch n := node.(type) {
	case *Node:
		if len(n.Embeddings) > 0 {
			return n.SubGraph.Pretty(n.dt.Labels)
		} else {
			return "0:0"
		}
	default:
		panic(errors.Errorf("unknown node type %v", node))
	}
}

func (f *Formatter) Pattern(node lattice.Node) (string, error) {
	switch n := node.(type) {
	case *Node:
		if len(n.Embeddings) > 0 {
			Pat := n.SubGraph.Pretty(n.dt.Labels)
			dot := n.SubGraph.Dotty(n.dt.Labels, nil, nil)
			return fmt.Sprintf("// %s\n\n%s\n", Pat, dot), nil
		} else {
			return fmt.Sprintf("// {0:0}\n\ndigraph{}\n"), nil
		}
	default:
		return "", errors.Errorf("unknown node type %v", node)
	}
}

func (f *Formatter) Embeddings(node lattice.Node) ([]string, error) {
	var embeddings []*subgraph.Embedding = nil
	switch n := node.(type) {
	case *Node:
		embeddings = n.Embeddings
	default:
		return nil, errors.Errorf("unknown node type %v", node)
	}
	embs := make([]string, 0, len(embeddings))
	for _, emb := range embeddings {
		embs = append(embs, fmt.Sprintf("%v", emb))
	}
	return embs, nil
}

func (f *Formatter) FormatPattern(w io.Writer, node lattice.Node) error {
	pat, err := f.Pattern(node)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", pat)
	return err
}

func (f *Formatter) FormatEmbeddings(w io.Writer, node lattice.Node) error {
	embs, err := f.Embeddings(node)
	if err != nil {
		return err
	}
	pat := f.PatternName(node)
	embeddings := strings.Join(embs, "\n")
	_, err = fmt.Fprintf(w, "// %s\n\n%s\n\n", pat, embeddings)
	return err
}
