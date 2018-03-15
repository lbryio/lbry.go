package dht

import (
	"bytes"
	"container/list"
	"net"
	"sort"

	"github.com/lbryio/errors.go"

	"github.com/zeebo/bencode"
)

type Node struct {
	id   bitmap
	ip   net.IP
	port int
}

func (n Node) MarshalCompact() ([]byte, error) {
	if n.ip.To4() == nil {
		return nil, errors.Err("ip not set")
	}
	if n.port < 0 || n.port > 65535 {
		return nil, errors.Err("invalid port")
	}

	var buf bytes.Buffer
	buf.Write(n.ip.To4())
	buf.WriteByte(byte(n.port >> 8))
	buf.WriteByte(byte(n.port))
	buf.Write(n.id[:])

	if buf.Len() != compactNodeInfoLength {
		return nil, errors.Err("i dont know how this happened")
	}

	return buf.Bytes(), nil
}

func (n *Node) UnmarshalCompact(b []byte) error {
	if len(b) != compactNodeInfoLength {
		return errors.Err("invalid compact length")
	}
	n.ip = net.IPv4(b[0], b[1], b[2], b[3])
	n.port = int(uint16(b[5]) | uint16(b[4])<<8)
	n.id = newBitmapFromBytes(b[6:])
	return nil
}

func (n Node) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes([]interface{}{n.id, n.ip.String(), n.port})
}

func (n *Node) UnmarshalBencode(b []byte) error {
	var raw []bencode.RawMessage
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return err
	}

	if len(raw) != 3 {
		return errors.Err("contact must have 3 elements; got %d", len(raw))
	}

	err = bencode.DecodeBytes(raw[0], &n.id)
	if err != nil {
		return err
	}

	var ipStr string
	err = bencode.DecodeBytes(raw[1], &ipStr)
	if err != nil {
		return err
	}
	n.ip = net.ParseIP(ipStr).To4()
	if n.ip == nil {
		return errors.Err("invalid IP")
	}

	err = bencode.DecodeBytes(raw[2], &n.port)
	if err != nil {
		return err
	}

	return nil
}

type SortedNode struct {
	node                *Node
	xorDistanceToTarget bitmap
}

type byXorDistance []*SortedNode

func (a byXorDistance) Len() int      { return len(a) }
func (a byXorDistance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byXorDistance) Less(i, j int) bool {
	return a[i].xorDistanceToTarget.Less(a[j].xorDistanceToTarget)
}

type RoutingTable struct {
	node    Node
	buckets [numBuckets]*list.List
}

func newRoutingTable(node *Node) *RoutingTable {
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
	var toSort []*SortedNode

	prefixLength := target.Xor(rt.node.id).PrefixLen()
	bucket := rt.buckets[prefixLength]
	toSort = appendNodes(toSort, bucket.Front(), nil, target)

	for i := 1; (prefixLength-i >= 0 || prefixLength+i < numBuckets) && len(toSort) < count; i++ {
		if prefixLength-i >= 0 {
			bucket = rt.buckets[prefixLength-i]
			toSort = appendNodes(toSort, bucket.Front(), nil, target)
		}
		if prefixLength+i < numBuckets {
			bucket = rt.buckets[prefixLength+i]
			toSort = appendNodes(toSort, bucket.Front(), nil, target)
		}
	}

	sort.Sort(byXorDistance(toSort))

	var nodes []*Node
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

func appendNodes(nodes []*SortedNode, start, end *list.Element, target bitmap) []*SortedNode {
	for curr := start; curr != end; curr = curr.Next() {
		node := curr.Value.(*Node)
		nodes = append(nodes, &SortedNode{node, node.id.Xor(target)})
	}
	return nodes
}
