package mine

import "context"

type TopMiner interface {
	Mine(context.Context, *Miner) SearchNodes
	MineFrom(context.Context, *Miner, *SearchNode) SearchNodes
}
