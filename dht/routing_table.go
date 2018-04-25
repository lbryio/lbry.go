package dht

import (
	"bytes"
	"container/list"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/lbryio/errors.go"

	"github.com/lyoshenka/bencode"
)

type Node struct {
	id    Bitmap
	ip    net.IP
	port  int
	token string // this is set when the node is returned from a FindNode call
}

func (n Node) String() string {
	return n.id.HexShort() + "@" + n.Addr().String()
}

func (n Node) Addr() *net.UDPAddr {
	return &net.UDPAddr{IP: n.ip, Port: n.port}
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
	n.ip = net.IPv4(b[0], b[1], b[2], b[3]).To4()
	n.port = int(uint16(b[5]) | uint16(b[4])<<8)
	n.id = BitmapFromBytesP(b[6:])
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

type sortedNode struct {
	node                Node
	xorDistanceToTarget Bitmap
}

type byXorDistance []sortedNode

func (a byXorDistance) Len() int      { return len(a) }
func (a byXorDistance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byXorDistance) Less(i, j int) bool {
	return a[i].xorDistanceToTarget.Less(a[j].xorDistanceToTarget)
}

type routingTable struct {
	node    Node
	buckets [numBuckets]*list.List
	lock    *sync.RWMutex
}

func newRoutingTable(node *Node) *routingTable {
	var rt routingTable
	for i := range rt.buckets {
		rt.buckets[i] = list.New()
	}
	rt.node = *node
	rt.lock = &sync.RWMutex{}
	return &rt
}

func (rt *routingTable) BucketInfo() string {
	rt.lock.RLock()
	defer rt.lock.RUnlock()

	bucketInfo := []string{}
	for i, b := range rt.buckets {
		contents := bucketContents(b)
		if contents != "" {
			bucketInfo = append(bucketInfo, fmt.Sprintf("Bucket %d: %s", i, contents))
		}
	}
	if len(bucketInfo) == 0 {
		return "buckets are empty"
	}
	return strings.Join(bucketInfo, "\n")
}

func bucketContents(b *list.List) string {
	count := 0
	ids := ""
	for curr := b.Front(); curr != nil; curr = curr.Next() {
		count++
		if ids != "" {
			ids += ", "
		}
		ids += curr.Value.(Node).id.HexShort()
	}

	if count > 0 {
		return fmt.Sprintf("(%d) %s", count, ids)
	} else {
		return ""
	}
}

// Update inserts or refreshes a node
func (rt *routingTable) Update(node Node) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	bucketNum := bucketFor(rt.node.id, node.id)
	bucket := rt.buckets[bucketNum]
	element := findInList(bucket, node.id)
	if element == nil {
		if bucket.Len() >= bucketSize {
			// TODO: Ping front node first. Only remove if it does not respond
			bucket.Remove(bucket.Front())
		}
		bucket.PushBack(node)
	} else {
		bucket.MoveToBack(element)
	}
}

// UpdateIfExists refreshes a node if its already in the routing table
func (rt *routingTable) UpdateIfExists(node Node) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	bucketNum := bucketFor(rt.node.id, node.id)
	bucket := rt.buckets[bucketNum]
	element := findInList(bucket, node.id)
	if element != nil {
		bucket.MoveToBack(element)
	}
}

func (rt *routingTable) RemoveByID(id Bitmap) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	bucketNum := bucketFor(rt.node.id, id)
	bucket := rt.buckets[bucketNum]
	element := findInList(bucket, rt.node.id)
	if element != nil {
		bucket.Remove(element)
	}
}

func (rt *routingTable) GetClosest(target Bitmap, limit int) []Node {
	rt.lock.RLock()
	defer rt.lock.RUnlock()

	var toSort []sortedNode
	var bucketNum int

	if rt.node.id.Equals(target) {
		bucketNum = 0
	} else {
		bucketNum = bucketFor(rt.node.id, target)
	}

	bucket := rt.buckets[bucketNum]
	toSort = appendNodes(toSort, bucket.Front(), target)

	for i := 1; (bucketNum-i >= 0 || bucketNum+i < numBuckets) && len(toSort) < limit; i++ {
		if bucketNum-i >= 0 {
			bucket = rt.buckets[bucketNum-i]
			toSort = appendNodes(toSort, bucket.Front(), target)
		}
		if bucketNum+i < numBuckets {
			bucket = rt.buckets[bucketNum+i]
			toSort = appendNodes(toSort, bucket.Front(), target)
		}
	}

	sort.Sort(byXorDistance(toSort))

	var nodes []Node
	for _, c := range toSort {
		nodes = append(nodes, c.node)
		if len(nodes) >= limit {
			break
		}
	}

	return nodes
}

// Count returns the number of nodes in the routing table
func (rt *routingTable) Count() int {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	count := 0
	for _, bucket := range rt.buckets {
		for curr := bucket.Front(); curr != nil; curr = curr.Next() {
			count++
		}
	}
	return count
}

func findInList(bucket *list.List, value Bitmap) *list.Element {
	for curr := bucket.Front(); curr != nil; curr = curr.Next() {
		if curr.Value.(Node).id.Equals(value) {
			return curr
		}
	}
	return nil
}

func appendNodes(nodes []sortedNode, start *list.Element, target Bitmap) []sortedNode {
	for curr := start; curr != nil; curr = curr.Next() {
		node := curr.Value.(Node)
		nodes = append(nodes, sortedNode{node, node.id.Xor(target)})
	}
	return nodes
}

func bucketFor(id Bitmap, target Bitmap) int {
	if id.Equals(target) {
		panic("nodes do not have a bucket for themselves")
	}
	return numBuckets - 1 - target.Xor(id).PrefixLen()
}

func sortNodesInPlace(nodes []Node, target Bitmap) {
	toSort := make([]sortedNode, len(nodes))

	for i, n := range nodes {
		toSort[i] = sortedNode{n, n.id.Xor(target)}
	}

	sort.Sort(byXorDistance(toSort))

	for i, c := range toSort {
		nodes[i] = c.node
	}
}
