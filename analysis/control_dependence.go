package analysis

import (
	"fmt"
	"strconv"
	"strings"
)

type ControlDependenceGraph struct {
	prev [][]*Block
	next [][]*Block
}

// Algorithm in Fig. 14 from Cytron's classic paper:
//
// Cytron R., Ferrante J., Rosen B. K., and Wegman M. N. "Efficiently Computing
// Static Single Assignment Form and the Control Dependence Graph." ACM TOPLAS.
// https://doi.org/10.1145/115372.115320
func ControlDependencies(cfg *CFG) *ControlDependenceGraph {
	g := &ControlDependenceGraph{
		prev: make([][]*Block, len(cfg.Blocks)),
		next: make([][]*Block, len(cfg.Blocks)),
	}
	next := make(map[int]map[int]bool, len(cfg.Blocks))
	frontier := cfg.PostDominators().Frontier()
	for _, y := range cfg.Blocks {
		for _, x := range frontier.Frontier(y) {
			if next[x.Id] == nil {
				next[x.Id] = make(map[int]bool, len(cfg.Blocks))
			}
			next[x.Id][y.Id] = true
		}
	}
	for x, ys := range next {
		for y := range ys {
			g.next[x] = append(g.next[x], cfg.Blocks[y])
			g.prev[y] = append(g.prev[y], cfg.Blocks[x])
		}
	}
	for x, prevs := range g.prev {
		// x is not entry or is unconnected to graph
		if x != 0 && (len(prevs) == 0 || (len(prevs) == 1 && prevs[0].Id == x)) {
			g.next[0] = append(g.next[0], cfg.Blocks[x])
			g.prev[x] = append(g.prev[x], cfg.Blocks[0])
		}
	}
	return g
}

func (cdg *ControlDependenceGraph) Next(blk *Block) []*Block {
	blks := make([]*Block, len(cdg.next[blk.Id]))
	copy(blks, cdg.next[blk.Id])
	return blks
}

func (cdg *ControlDependenceGraph) Prev(blk *Block) []*Block {
	blks := make([]*Block, len(cdg.prev[blk.Id]))
	copy(blks, cdg.prev[blk.Id])
	return blks
}

func (cdg *ControlDependenceGraph) String() string {
	return ""
}

func (cdg *ControlDependenceGraph) Dotty(cfg *CFG) string {
	nodes := make([]string, 0, len(cfg.Blocks))
	edges := make([]string, 0, len(cfg.Blocks))
	for _, b := range cfg.Blocks {
		label := strconv.Quote(b.DotLabel())
		label = strings.Replace(label, "\\n", "\\l", -1)
		nodes = append(nodes, fmt.Sprintf("n%d [label=%v]", b.Id, label))
		for _, n := range cdg.Next(b) {
			edges = append(edges, fmt.Sprintf("n%d -> n%d", b.Id, n.Id))
		}
	}
	return fmt.Sprintf(`digraph %v {
label=%v
labelloc=top
node [shape="rect", labeljust=l]
%v
%v
}`, strconv.Quote("cdg-"+cfg.Name), strconv.Quote("cdg-"+cfg.Name), strings.Join(nodes, "\n"), strings.Join(edges, "\n"))
}
