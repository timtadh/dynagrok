package subgraph

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice/digraph"
)


func (sg *SubGraph) SupportOf(indices *digraph.Indices) (size, support int, err error) {
	// errors.Logf("DEBUG", "checking %v", sg)
	_, rsg, err := sg.EstimateMatch(indices)
	if err != nil {
		return 0, 0, err
	}
	// errors.Logf("DEBUG", "rsg %v, %p", rsg, g.Labels)
	if len(rsg.V) == 0 {
		return 0, 0, nil
	}
	ei := rsg.IterEmbeddings(MostConnected, indices, nil, nil)
	count := 0
	seen := make(map[int]bool)
	stop := false
	for emb, ei := ei(stop); ei != nil; emb, ei = ei(stop) {
		seenIt := false
		for ve := emb; ve != nil; ve = ve.Prev {
			if seen[ve.EmbIdx] {
				seenIt = true
			}
			seen[ve.EmbIdx] = true
		}
		if emb != nil && !seenIt {
			count++
		}
	}
	return len(rsg.E), count, nil
}

func (sg *SubGraph) EstimateMatch(indices *digraph.Indices) (match float64, csg *SubGraph, err error) {
	csg = sg
	for len(csg.E) >= 0 {
		found, chain, maxEid, _ := csg.Embedded(indices)
		if false {
			errors.Logf("INFO", "found: %v %v %v %v", found, chain, maxEid, nil)
		}
		if found {
			if len(sg.E) == 0 {
				match = 1
			} else {
				match = float64(maxEid)/float64(len(sg.E))
			}
			return match, csg, nil
		}
		var b *Builder
		connected := false
		eid := maxEid
		if len(csg.E) == 1 {
			break
		}
		for !connected && eid >= 0 && eid < len(chain) {
			b = csg.Builder()
			if err := b.RemoveEdge(chain[eid]); err != nil {
				return 0, nil, err
			}
			connected = b.Connected()
			eid++
		}
		if !connected {
			break
		}
		csg = b.Build()
	}
	return 0, EmptySubGraph(), nil
}

func (sg *SubGraph) Embedded(indices *digraph.Indices) (found bool, edgeChain []int, largestEid int, longest *Embedding) {
	type entry struct {
		ids *Embedding
		eid int
	}
	pop := func(stack []entry) (entry, []entry) {
		return stack[len(stack)-1], stack[0 : len(stack)-1]
	}
	largestEid = -1
	for startIdx := 0; startIdx < len(sg.V); startIdx++ {
		// startIdx := sg.searchStartingPoint(MostExtensions, indices, nil)
		chain := sg.edgeChain(indices, nil, startIdx)
		vembs := sg.startEmbeddings(indices, startIdx)
		// errors.Logf("DEBUG", "chain %v", chain)
		// errors.Logf("DEBUG", "vembs %v", vembs)
		// errors.Logf("DEBUG", "color idx %v", indices.ColorIndex)
		stack := make([]entry, 0, len(vembs)*2)
		for _, vemb := range vembs {
			stack = append(stack, entry{vemb, 0})
		}
		if false {
			errors.Logf("DEBUG", "stack %v", stack)
		}
		for len(stack) > 0 {
			var i entry
			i, stack = pop(stack)
			if i.eid > largestEid {
				edgeChain = chain
				longest = i.ids
				largestEid = i.eid
			}
			if i.eid >= len(chain) {
				return true, chain, i.eid, i.ids
			} else {
				sg.extendEmbedding(indices, i.ids, &sg.E[chain[i.eid]], nil, func(ext *Embedding) {
					stack = append(stack, entry{ext, i.eid + 1})
				})
			}
		}
	}
	return false, edgeChain, largestEid, longest
}
