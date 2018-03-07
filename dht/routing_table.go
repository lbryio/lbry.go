package dht

import (
	"container/list"
	"sort"
)

type RoutingTable struct {
	node    Node
	buckets [numBuckets]*list.List
}

func NewRoutingTable(node *Node) *RoutingTable {
	var rt RoutingTable
	for i := range rt.buckets {
		rt.buckets[i] = list.New()
	}
	rt.node = *node
	return &rt
}

func (rt *RoutingTable) Update(node *Node) {
	prefixLength := node.id.Xor(rt.node.id).PrefixLen()
	bucket := rt.buckets[prefixLength]
	element := findInList(bucket, rt.node.id)
	if element == nil {
		if bucket.Len() <= bucketSize {
			bucket.PushBack(node)
		}
		// TODO: Handle insertion when the list is full by evicting old elements if
		// they don't respond to a ping.
	} else {
		bucket.MoveToBack(element)
	}
}

func (rt *RoutingTable) FindClosest(target bitmap, count int) []*Node {
	toSort := []*SortedNode{}

	prefixLength := target.Xor(rt.node.id).PrefixLen()
	bucket := rt.buckets[prefixLength]
	appendNodes(bucket.Front(), nil, &toSort, target)

	for i := 1; (prefixLength-i >= 0 || prefixLength+i < nodeIDLength*8) && len(toSort) < count; i++ {
		if prefixLength-i >= 0 {
			bucket = rt.buckets[prefixLength-i]
			appendNodes(bucket.Front(), nil, &toSort, target)
		}
		if prefixLength+i < nodeIDLength*8 {
			bucket = rt.buckets[prefixLength+i]
			appendNodes(bucket.Front(), nil, &toSort, target)
		}
	}

	sort.Sort(byXorDistance(toSort))

	nodes := []*Node{}
	for _, c := range toSort {
		nodes = append(nodes, c.node)
	}

	return nodes
}

func findInList(bucket *list.List, value bitmap) *list.Element {
	for curr := bucket.Front(); curr != nil; curr = curr.Next() {
		if curr.Value.(*Node).id.Equals(value) {
			return curr
		}
	}
	return nil
}

func appendNodes(start, end *list.Element, nodes *[]*SortedNode, target bitmap) {
	for curr := start; curr != end; curr = curr.Next() {
		node := curr.Value.(*Node)
		*nodes = append(*nodes, &SortedNode{node, node.id.Xor(target)})
	}
}
