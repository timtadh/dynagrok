package mine

type TopMiner interface {
	Mine(*Miner) SearchNodes
	MineFrom(*Miner, *SearchNode) SearchNodes
}
