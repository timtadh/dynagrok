package mine

import (
	"context"
	"runtime"
	"sync"

	"github.com/timtadh/data-structures/errors"
)

type Walker interface {
	Walk(*Miner) *SearchNode
	WalkFrom(*Miner, *SearchNode) *SearchNode
	WalkFromColor(*Miner, int) *SearchNode
}

func Walking(walker Walker, walks int) MinerFunc {
	return func(ctx context.Context, m *Miner) SearchNodes {
		return WalksToNodes(ctx, m, walker.Walk, walks)
	}
}

func WalksToNodes(ctx context.Context, m *Miner, walk Walk, walks int) (sni SearchNodes) {
	i := 0
	sni = func() (*SearchNode, SearchNodes) {
		if i >= walks {
			return nil, nil
		}
		var n *SearchNode
		for i < walks {
			if ctx.Err() != nil {
				return nil, nil
			}
			n = walk(m)
			i++
			if n.Node != nil && n.Node.SubGraph != nil && len(n.Node.SubGraph.E) >= m.MinEdges {
				break
			}
		}
		if n.Node == nil || n.Node.SubGraph == nil {
			return nil, nil
		}
		return n, sni
	}
	return sni
}

type topColorOpts struct {
	percentOfColors float64
	walksPerColor   int
	minGroups       int
	skipSeenColors  bool
	debug           int
}

type TopColorOpt func(*topColorOpts)

func PercentOfColors(p float64) TopColorOpt {
	return func(o *topColorOpts) {
		o.percentOfColors = p
	}
}

func WalksPerColor(w int) TopColorOpt {
	return func(o *topColorOpts) {
		o.walksPerColor = w
	}
}

func MinGroupsWalked(m int) TopColorOpt {
	return func(o *topColorOpts) {
		o.minGroups = m
	}
}

func SkipSeenColors() TopColorOpt {
	return func(o *topColorOpts) {
		o.skipSeenColors = true
	}
}

func WTCDebugLevel(i int) TopColorOpt {
	return func(o *topColorOpts) {
		o.debug = i
	}
}

func WalkingTopColors(walker Walker, opts ...TopColorOpt) MinerFunc {
	o := &topColorOpts{
		percentOfColors: .0625,
		walksPerColor:   2,
		minGroups:       2,
		skipSeenColors:  false,
		debug:           0,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(ctx context.Context, m *Miner) (sni SearchNodes) {
		locations := LocalizeNodes(m.Score)
		total := int(o.percentOfColors * float64(len(locations)))
		if total < 10 {
			total = 10
		}
		if total > len(locations) {
			total = len(locations)
		}

		added := make(map[string]bool)
		colors := make(map[int]bool)
		prevScore := 0.0
		groups := 0
		count := 0
		i := 0
		w := 0
		sni = func() (*SearchNode, SearchNodes) {
		start:
			if ctx.Err() != nil {
				if o.debug >= 2 {
					errors.Logf("DEBUG", "canceled %d/%v %d/%d %d/%d %d out of locations", groups, o.minGroups, i, total, w, o.walksPerColor, count)
				}
				return nil, nil
			}
			if w >= o.walksPerColor {
				l := locations[i]
				if prevScore-l.Score > .0001 {
					groups++
				}
				prevScore = l.Score
				w = 0
				i++
			}
			if i >= len(locations) {
				if o.debug >= 2 {
					errors.Logf("DEBUG", "done %d/%v %d/%d %d/%d %d out of locations", groups, o.minGroups, i, total, w, o.walksPerColor, count)
				}
				return nil, nil
			}
			if i >= total && groups >= o.minGroups {
				if o.debug >= 2 {
					errors.Logf("DEBUG", "done %d/%v %d/%d %d/%d %d ending condition reached", groups, o.minGroups, i, total, w, o.walksPerColor, count)
				}
				return nil, nil
			}
			color := locations[i].Color
			if o.skipSeenColors && w == 0 && colors[color] {
				i++
				if o.debug >= 3 {
					errors.Logf("DEBUG", "skipped %d/%v %d/%d %d/%d %d seen this color before", groups, o.minGroups, i, total, w, o.walksPerColor, count)
				}
				goto start
			}
			var n *SearchNode
			n = walker.WalkFromColor(m, color)
			w++
			if n.Node.SubGraph == nil || len(n.Node.SubGraph.E) < m.MinEdges {
				if o.debug >= 3 {
					errors.Logf("DEBUG", "skipped %d/%v %d/%d %d/%d %d no edges", groups, o.minGroups, i, total, w, o.walksPerColor, count)
				}
				goto start
			}
			label := string(n.Node.SubGraph.Label())
			if added[label] {
				if o.debug >= 3 {
					errors.Logf("DEBUG", "skipped %d/%v %d/%d %d/%d %d previously seen", groups, o.minGroups, i, total, w, o.walksPerColor, count)
				}
				goto start
			}
			added[label] = true
			for _, v := range n.Node.SubGraph.V {
				colors[v.Color] = true
			}
			count++
			if o.debug >= 1 {
				errors.Logf("DEBUG", "found %d/%v %d/%d %d/%d %d %v", groups, o.minGroups, i, total, w, o.walksPerColor, count, n)
			}
			return n, sni
		}
		return sni
	}
}

func ParTopColors(walker Walker, opts ...TopColorOpt) MinerFunc {
	o := &topColorOpts{
		percentOfColors: .0625,
		walksPerColor:   2,
		minGroups:       2,
		skipSeenColors:  false,
		debug:           0,
	}
	for _, opt := range opts {
		opt(o)
	}
	gen := func(ctx context.Context, m *Miner, wg *sync.WaitGroup, out chan<- int, nodes chan *SearchNode) {
		locations := LocalizeNodes(m.Score)
		total := int(o.percentOfColors * float64(len(locations)))
		if total < 10 {
			total = 10
		}
		if total > len(locations) {
			total = len(locations)
		}
		prevScore := -1e27
		groups := 0
	outer:
		for i := 0; i < len(locations) && (i < total || groups < o.minGroups); i++ {
			l := locations[i]
			for w := 0; w < o.walksPerColor; w++ {
				if ctx.Err() != nil {
					if o.debug >= 2 {
						errors.Logf("DEBUG", "canceled (%d/%d) (%d/%d) (%d/%d) %d", groups, o.minGroups, i, total, w, o.walksPerColor, l.Color)
					}
					break outer
				}
				if o.debug >= 1 {
					errors.Logf("DEBUG", "sending (%d/%d) (%d/%d) (%d/%d) %d", groups, o.minGroups, i, total, w, o.walksPerColor, l.Color)
				}
				select {
				case out <- l.Color:
				case <-ctx.Done():
					break outer
				}
			}
			if prevScore-l.Score > .0001 {
				groups++
			}
			prevScore = l.Score
		}
		close(out)
		wg.Wait()
		close(nodes)
	}
	work := func(ctx context.Context, m *Miner, wg *sync.WaitGroup, in <-chan int, out chan<- *SearchNode) {
		for color := range in {
			if ctx.Err() != nil {
				break
			}
			out <- walker.WalkFromColor(m, color)
		}
		wg.Done()
	}
	return func(ctx context.Context, m *Miner) (sni SearchNodes) {
		var wg sync.WaitGroup
		colors := make(chan int)
		nodes := make(chan *SearchNode)
		wg.Add(runtime.NumCPU())
		for i := 0; i < runtime.NumCPU(); i++ {
			go work(ctx, m, &wg, colors, nodes)
		}
		go gen(ctx, m, &wg, colors, nodes)
		added := make(map[string]bool)
		count := 0
		sni = func() (*SearchNode, SearchNodes) {
		start:
			n, ok := <-nodes
			if !ok {
				return nil, nil
			}
			if n.Node.SubGraph == nil || len(n.Node.SubGraph.E) < m.MinEdges {
				if o.debug >= 3 {
					errors.Logf("DEBUG", "skipped %d no edges", count)
				}
				goto start
			}
			label := string(n.Node.SubGraph.Label())
			if added[label] {
				if o.debug >= 3 {
					errors.Logf("DEBUG", "skipped %d previously seen", count)
				}
				goto start
			}
			added[label] = true
			count++
			if o.debug >= 1 {
				errors.Logf("DEBUG", "found %d %v", count, n)
			}
			return n, sni
		}
		return sni
	}
}
