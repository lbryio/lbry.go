package dht

import (
	"bytes"
	"container/list"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

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

// peer is a contact with extra freshness information
type peer struct {
	contact      Contact
	lastActivity time.Time
	numFailures  int
	//<lastPublished>,
	//<originallyPublished>
	//	<originalPublisherID>
}

func (p *peer) Touch() {
	p.lastActivity = time.Now()
	p.numFailures = 0
}

// ActiveSince returns whether a peer has responded in the last `d` duration
// this is used to check if the peer is "good", meaning that we believe the peer will respond to our requests
func (p *peer) ActiveInLast(d time.Duration) bool {
	return time.Now().Sub(p.lastActivity) > d
}

// IsBad returns whether a peer is "bad", meaning that it has failed to respond to multiple pings in a row
func (p *peer) IsBad(maxFalures int) bool {
	return p.numFailures >= maxFalures
}

// Fail marks a peer as having failed to respond. It returns whether or not the peer should be removed from the routing table
func (p *peer) Fail() {
	p.numFailures++
}

// toPeer converts a generic *list.Element into a *peer
// this (along with newPeer) keeps all conversions between *list.Element and peer in one place
func toPeer(el *list.Element) *peer {
	return el.Value.(*peer)
}

// newPeer creates a new peer from a contact
// this (along with toPeer) keeps all conversions between *list.Element and peer in one place
func newPeer(c Contact) peer {
	return peer{
		contact: c,
	}
}

type bucket struct {
	lock       *sync.RWMutex
	peers      *list.List
	lastUpdate time.Time
}

// Len returns the number of peers in the bucket
func (b bucket) Len() int {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.peers.Len()
}

// Contacts returns a slice of the bucket's contacts
func (b bucket) Contacts() []Contact {
	b.lock.RLock()
	defer b.lock.RUnlock()
	contacts := make([]Contact, b.peers.Len())
	for i, curr := 0, b.peers.Front(); curr != nil; i, curr = i+1, curr.Next() {
		contacts[i] = toPeer(curr).contact
	}
	return contacts
}

// UpdateContact marks a contact as having been successfully contacted. if insertIfNew and the contact is does not exist yet, it is inserted
func (b *bucket) UpdateContact(c Contact, insertIfNew bool) {
	b.lock.Lock()
	defer b.lock.Unlock()

	element := find(c.id, b.peers)
	if element != nil {
		b.lastUpdate = time.Now()
		toPeer(element).Touch()
		b.peers.MoveToBack(element)

	} else if insertIfNew {
		hasRoom := true

		if b.peers.Len() >= bucketSize {
			hasRoom = false
			for curr := b.peers.Front(); curr != nil; curr = curr.Next() {
				if toPeer(curr).IsBad(maxPeerFails) {
					// TODO: Ping contact first. Only remove if it does not respond
					b.peers.Remove(curr)
					hasRoom = true
					break
				}
			}
		}

		if hasRoom {
			b.lastUpdate = time.Now()
			peer := newPeer(c)
			peer.Touch()
			b.peers.PushBack(&peer)
		}
	}
}

// FailContact marks a contact as having failed, and removes it if it failed too many times
func (b *bucket) FailContact(id Bitmap) {
	b.lock.Lock()
	defer b.lock.Unlock()
	element := find(id, b.peers)
	if element != nil {
		// BEP5 says not to remove the contact until the bucket is full and you try to insert
		toPeer(element).Fail()
	}
}

// find returns the contact in the bucket, or nil if the bucket does not contain the contact
func find(id Bitmap, peers *list.List) *list.Element {
	for curr := peers.Front(); curr != nil; curr = curr.Next() {
		if toPeer(curr).contact.id.Equals(id) {
			return curr
		}
	}
	return nil
}

// NeedsRefresh returns true if bucket has not been updated in the last `refreshInterval`, false otherwise
func (b *bucket) NeedsRefresh(refreshInterval time.Duration) bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return time.Now().Sub(b.lastUpdate) > refreshInterval
}

type RoutingTable interface {
	Update(Contact)
	Fresh(Contact)
	Fail(Contact)
	GetClosest(Bitmap, int) []Contact
	Count() int
	GetIDsForRefresh(time.Duration) []Bitmap
	BucketInfo() string // for debugging
}

type routingTableImpl struct {
	id      Bitmap
	buckets [numBuckets]bucket
}

func newRoutingTable(id Bitmap) *routingTableImpl {
	var rt routingTableImpl
	rt.id = id
	for i := range rt.buckets {
		rt.buckets[i] = bucket{
			peers: list.New(),
			lock:  &sync.RWMutex{},
		}
	}
	return &rt
}

func (rt *routingTableImpl) BucketInfo() string {
	var bucketInfo []string
	for i, b := range rt.buckets {
		if b.Len() > 0 {
			contacts := b.Contacts()
			s := make([]string, len(contacts))
			for j, c := range contacts {
				s[j] = c.id.HexShort()
			}
			bucketInfo = append(bucketInfo, fmt.Sprintf("Bucket %d: (%d) %s", i, len(contacts), strings.Join(s, ", ")))
		}
	}
	if len(bucketInfo) == 0 {
		return "buckets are empty"
	}
	return strings.Join(bucketInfo, "\n")
}

// Update inserts or refreshes a contact
func (rt *routingTableImpl) Update(c Contact) {
	rt.bucketFor(c.id).UpdateContact(c, true)
}

// Fresh refreshes a contact if its already in the routing table
func (rt *routingTableImpl) Fresh(c Contact) {
	rt.bucketFor(c.id).UpdateContact(c, false)
}

// FailContact marks a contact as having failed, and removes it if it failed too many times
func (rt *routingTableImpl) Fail(c Contact) {
	rt.bucketFor(c.id).FailContact(c.id)
}

// GetClosest returns the closest `limit` contacts from the routing table
// It marks each bucket it accesses as having been accessed
func (rt *routingTableImpl) GetClosest(target Bitmap, limit int) []Contact {
	var toSort []sortedContact
	var bucketNum int

	if rt.id.Equals(target) {
		bucketNum = 0
	} else {
		bucketNum = rt.bucketNumFor(target)
	}

	toSort = appendContacts(toSort, rt.buckets[bucketNum], target)

	for i := 1; (bucketNum-i >= 0 || bucketNum+i < numBuckets) && len(toSort) < limit; i++ {
		if bucketNum-i >= 0 {
			toSort = appendContacts(toSort, rt.buckets[bucketNum-i], target)
		}
		if bucketNum+i < numBuckets {
			toSort = appendContacts(toSort, rt.buckets[bucketNum+i], target)
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

func appendContacts(contacts []sortedContact, b bucket, target Bitmap) []sortedContact {
	for _, contact := range b.Contacts() {
		contacts = append(contacts, sortedContact{contact, contact.id.Xor(target)})
	}
	return contacts
}

// Count returns the number of contacts in the routing table
func (rt *routingTableImpl) Count() int {
	count := 0
	for _, bucket := range rt.buckets {
		count = bucket.Len()
	}
	return count
}

func (rt *routingTableImpl) bucketNumFor(target Bitmap) int {
	if rt.id.Equals(target) {
		panic("routing table does not have a bucket for its own id")
	}
	return numBuckets - 1 - target.Xor(rt.id).PrefixLen()
}

func (rt *routingTableImpl) bucketFor(target Bitmap) *bucket {
	return &rt.buckets[rt.bucketNumFor(target)]
}

func (rt *routingTableImpl) GetIDsForRefresh(refreshInterval time.Duration) []Bitmap {
	var bitmaps []Bitmap
	for i, bucket := range rt.buckets {
		if bucket.NeedsRefresh(refreshInterval) {
			bitmaps = append(bitmaps, RandomBitmapP().ZeroPrefix(i))
		}
	}
	return bitmaps
}

// RoutingTableRefresh refreshes any buckets that need to be refreshed
// It returns a channel that will be closed when the refresh is done
func RoutingTableRefresh(n *Node, refreshInterval time.Duration, cancel <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})

	var wg sync.WaitGroup

	for _, id := range n.rt.GetIDsForRefresh(refreshInterval) {
		wg.Add(1)
		go func(id Bitmap) {
			defer wg.Done()

			nf := newContactFinder(n, id, false)

			if cancel != nil {
				go func() {
					select {
					case <-cancel:
						nf.Cancel()
					case <-done:
					}
				}()
			}

			nf.Find()
		}(id)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}
