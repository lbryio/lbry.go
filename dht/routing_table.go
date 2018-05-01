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

type Contact struct {
	id   Bitmap
	ip   net.IP
	port int
}

func (c Contact) Addr() *net.UDPAddr {
	return &net.UDPAddr{IP: c.ip, Port: c.port}
}

func (c Contact) String() string {
	return c.id.HexShort() + "@" + c.Addr().String()
}

func (c Contact) MarshalCompact() ([]byte, error) {
	if c.ip.To4() == nil {
		return nil, errors.Err("ip not set")
	}
	if c.port < 0 || c.port > 65535 {
		return nil, errors.Err("invalid port")
	}

	var buf bytes.Buffer
	buf.Write(c.ip.To4())
	buf.WriteByte(byte(c.port >> 8))
	buf.WriteByte(byte(c.port))
	buf.Write(c.id[:])

	if buf.Len() != compactNodeInfoLength {
		return nil, errors.Err("i dont know how this happened")
	}

	return buf.Bytes(), nil
}

func (c *Contact) UnmarshalCompact(b []byte) error {
	if len(b) != compactNodeInfoLength {
		return errors.Err("invalid compact length")
	}
	c.ip = net.IPv4(b[0], b[1], b[2], b[3]).To4()
	c.port = int(uint16(b[5]) | uint16(b[4])<<8)
	c.id = BitmapFromBytesP(b[6:])
	return nil
}

func (c Contact) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes([]interface{}{c.id, c.ip.String(), c.port})
}

func (c *Contact) UnmarshalBencode(b []byte) error {
	var raw []bencode.RawMessage
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return err
	}

	if len(raw) != 3 {
		return errors.Err("contact must have 3 elements; got %d", len(raw))
	}

	err = bencode.DecodeBytes(raw[0], &c.id)
	if err != nil {
		return err
	}

	var ipStr string
	err = bencode.DecodeBytes(raw[1], &ipStr)
	if err != nil {
		return err
	}
	c.ip = net.ParseIP(ipStr).To4()
	if c.ip == nil {
		return errors.Err("invalid IP")
	}

	err = bencode.DecodeBytes(raw[2], &c.port)
	if err != nil {
		return err
	}

	return nil
}

type sortedContact struct {
	contact             Contact
	xorDistanceToTarget Bitmap
}

type byXorDistance []sortedContact

func (a byXorDistance) Len() int      { return len(a) }
func (a byXorDistance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byXorDistance) Less(i, j int) bool {
	return a[i].xorDistanceToTarget.Less(a[j].xorDistanceToTarget)
}

type routingTable struct {
	id      Bitmap
	buckets [numBuckets]*list.List
	lock    *sync.RWMutex
}

func newRoutingTable(id Bitmap) *routingTable {
	var rt routingTable
	for i := range rt.buckets {
		rt.buckets[i] = list.New()
	}
	rt.id = id
	rt.lock = &sync.RWMutex{}
	return &rt
}

func (rt *routingTable) BucketInfo() string {
	rt.lock.RLock()
	defer rt.lock.RUnlock()

	var bucketInfo []string
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
		ids += curr.Value.(Contact).id.HexShort()
	}

	if count > 0 {
		return fmt.Sprintf("(%d) %s", count, ids)
	} else {
		return ""
	}
}

// Update inserts or refreshes a contact
func (rt *routingTable) Update(c Contact) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	bucketNum := rt.bucketFor(c.id)
	bucket := rt.buckets[bucketNum]
	element := findInList(bucket, c.id)
	if element == nil {
		if bucket.Len() >= bucketSize {
			// TODO: Ping front contact first. Only remove if it does not respond
			bucket.Remove(bucket.Front())
		}
		bucket.PushBack(c)
	} else {
		bucket.MoveToBack(element)
	}
}

// UpdateIfExists refreshes a contact if its already in the routing table
func (rt *routingTable) UpdateIfExists(c Contact) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	bucketNum := rt.bucketFor(c.id)
	bucket := rt.buckets[bucketNum]
	element := findInList(bucket, c.id)
	if element != nil {
		bucket.MoveToBack(element)
	}
}

func (rt *routingTable) Remove(id Bitmap) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	bucketNum := rt.bucketFor(id)
	bucket := rt.buckets[bucketNum]
	element := findInList(bucket, rt.id)
	if element != nil {
		bucket.Remove(element)
	}
}

func (rt *routingTable) GetClosest(target Bitmap, limit int) []Contact {
	rt.lock.RLock()
	defer rt.lock.RUnlock()

	var toSort []sortedContact
	var bucketNum int

	if rt.id.Equals(target) {
		bucketNum = 0
	} else {
		bucketNum = rt.bucketFor(target)
	}

	bucket := rt.buckets[bucketNum]
	toSort = appendContacts(toSort, bucket.Front(), target)

	for i := 1; (bucketNum-i >= 0 || bucketNum+i < numBuckets) && len(toSort) < limit; i++ {
		if bucketNum-i >= 0 {
			bucket = rt.buckets[bucketNum-i]
			toSort = appendContacts(toSort, bucket.Front(), target)
		}
		if bucketNum+i < numBuckets {
			bucket = rt.buckets[bucketNum+i]
			toSort = appendContacts(toSort, bucket.Front(), target)
		}
	}

	sort.Sort(byXorDistance(toSort))

	var contacts []Contact
	for _, sorted := range toSort {
		contacts = append(contacts, sorted.contact)
		if len(contacts) >= limit {
			break
		}
	}

	return contacts
}

func appendContacts(contacts []sortedContact, start *list.Element, target Bitmap) []sortedContact {
	for curr := start; curr != nil; curr = curr.Next() {
		c := toContact(curr)
		contacts = append(contacts, sortedContact{c, c.id.Xor(target)})
	}
	return contacts
}

// Count returns the number of contacts in the routing table
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

func (rt *routingTable) bucketFor(target Bitmap) int {
	if rt.id.Equals(target) {
		panic("routing table does not have a bucket for its own id")
	}
	return numBuckets - 1 - target.Xor(rt.id).PrefixLen()
}

func findInList(bucket *list.List, value Bitmap) *list.Element {
	for curr := bucket.Front(); curr != nil; curr = curr.Next() {
		if toContact(curr).id.Equals(value) {
			return curr
		}
	}
	return nil
}

func toContact(el *list.Element) Contact {
	return el.Value.(Contact)
}
