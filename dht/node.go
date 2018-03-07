package dht

const nodeIDLength = 48 // bytes
const compactNodeInfoLength = nodeIDLength + 6

type Node struct {
	id   bitmap
	addr string
}

type SortedNode struct {
	node    *Node
	sortKey bitmap
}

type byXorDistance []*SortedNode

func (a byXorDistance) Len() int           { return len(a) }
func (a byXorDistance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byXorDistance) Less(i, j int) bool { return a[i].sortKey.Less(a[j].sortKey) }
