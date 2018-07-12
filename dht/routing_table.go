package dht

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/stop"
	"github.com/lbryio/reflector.go/dht/bits"
)

// TODO: if routing table is ever empty (aka the node is isolated), it should re-bootstrap

// TODO: use a tree with bucket splitting instead of a fixed bucket list. include jack's optimization (see link in commit mesg)
// https://github.com/lbryio/lbry/pull/1211/commits/341b27b6d21ac027671d42458826d02735aaae41

// peer is a contact with extra information
type peer struct {
	Contact      Contact
	Distance     bits.Bitmap
	LastActivity time.Time
	// LastReplied time.Time
	// LastRequested time.Time
	// LastFailure time.Time
	// SecondLastFailure time.Time
	NumFailures int

	//<lastPublished>,
	//<originallyPublished>
	//	<originalPublisherID>
}

func (p *peer) Touch() {
	p.LastActivity = time.Now()
	p.NumFailures = 0
}

// ActiveSince returns whether a peer has responded in the last `d` duration
// this is used to check if the peer is "good", meaning that we believe the peer will respond to our requests
func (p *peer) ActiveInLast(d time.Duration) bool {
	return time.Since(p.LastActivity) < d
}

// IsBad returns whether a peer is "bad", meaning that it has failed to respond to multiple pings in a row
func (p *peer) IsBad(maxFalures int) bool {
	return p.NumFailures >= maxFalures
}

// Fail marks a peer as having failed to respond. It returns whether or not the peer should be removed from the routing table
func (p *peer) Fail() {
	p.NumFailures++
}

type bucket struct {
	lock       *sync.RWMutex
	peers      []peer
	lastUpdate time.Time
	Range      bits.Range // capitalized because `range` is a keyword
}

func newBucket(r bits.Range) *bucket {
	return &bucket{
		peers: make([]peer, 0, bucketSize),
		lock:  &sync.RWMutex{},
		Range: r,
	}
}

// Len returns the number of peers in the bucket
func (b bucket) Len() int {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return len(b.peers)
}

func (b bucket) Has(c Contact) bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for _, p := range b.peers {
		if p.Contact.Equals(c, true) {
			return true
		}
	}
	return false
}

// Contacts returns a slice of the bucket's contacts
func (b bucket) Contacts() []Contact {
	b.lock.RLock()
	defer b.lock.RUnlock()
	contacts := make([]Contact, len(b.peers))
	for i := range b.peers {
		contacts[i] = b.peers[i].Contact
	}
	return contacts
}

// UpdatePeer marks a contact as having been successfully contacted. if insertIfNew and the contact is does not exist yet, it is inserted
func (b *bucket) UpdatePeer(p peer, insertIfNew bool) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if !b.Range.Contains(p.Distance) {
		return errors.Err("this bucket range does not cover this peer")
	}

	peerIndex := find(p.Contact.ID, b.peers)
	if peerIndex >= 0 {
		b.lastUpdate = time.Now()
		b.peers[peerIndex].Touch()
		moveToBack(b.peers, peerIndex)
	} else if insertIfNew {
		hasRoom := true

		if len(b.peers) >= bucketSize {
			hasRoom = false
			for i := range b.peers {
				if b.peers[i].IsBad(maxPeerFails) {
					// TODO: Ping contact first. Only remove if it does not respond
					b.peers = append(b.peers[:i], b.peers[i+1:]...)
					hasRoom = true
					break
				}
			}
		}

		if hasRoom {
			b.lastUpdate = time.Now()
			p.Touch()
			b.peers = append(b.peers, p)
		}
	}

	return nil
}

// FailContact marks a contact as having failed, and removes it if it failed too many times
func (b *bucket) FailContact(id bits.Bitmap) {
	b.lock.Lock()
	defer b.lock.Unlock()
	i := find(id, b.peers)
	if i >= 0 {
		// BEP5 says not to remove the contact until the bucket is full and you try to insert
		b.peers[i].Fail()
	}
}

// find returns the contact in the bucket, or nil if the bucket does not contain the contact
func find(id bits.Bitmap, peers []peer) int {
	for i := range peers {
		if peers[i].Contact.ID.Equals(id) {
			return i
		}
	}
	return -1
}

// NeedsRefresh returns true if bucket has not been updated in the last `refreshInterval`, false otherwise
func (b *bucket) NeedsRefresh(refreshInterval time.Duration) bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return time.Since(b.lastUpdate) > refreshInterval
}

func (b *bucket) Split() (*bucket, *bucket) {
	b.lock.Lock()
	defer b.lock.Unlock()

	left := newBucket(b.Range.IntervalP(1, 2))
	right := newBucket(b.Range.IntervalP(2, 2))
	left.lastUpdate = b.lastUpdate
	right.lastUpdate = b.lastUpdate

	for _, p := range b.peers {
		if left.Range.Contains(p.Distance) {
			left.peers = append(left.peers, p)
		} else {
			right.peers = append(right.peers, p)
		}
	}

	if len(b.peers) > 1 {
		if len(left.peers) == 0 {
			left, right = right.Split()
			left.Range.Start = b.Range.Start
		} else if len(right.peers) == 0 {
			left, right = left.Split()
			right.Range.End = b.Range.End
		}
	}

	return left, right
}

type routingTable struct {
	id      bits.Bitmap
	buckets []*bucket
	mu      *sync.RWMutex // this mutex is write-locked only when CHANGING THE NUMBER OF BUCKETS in the table
}

func newRoutingTable(id bits.Bitmap) *routingTable {
	rt := routingTable{
		id: id,
		mu: &sync.RWMutex{},
	}
	rt.reset()
	return &rt
}

func (rt *routingTable) reset() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.buckets = []*bucket{newBucket(bits.MaxRange())}
}

func (rt *routingTable) BucketInfo() string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var bucketInfo []string
	for i, b := range rt.buckets {
		if b.Len() > 0 {
			contacts := b.Contacts()
			s := make([]string, len(contacts))
			for j, c := range contacts {
				s[j] = c.ID.HexShort()
			}
			bucketInfo = append(bucketInfo, fmt.Sprintf("bucket %d: (%d) %s", i, len(contacts), strings.Join(s, ", ")))
		}
	}
	if len(bucketInfo) == 0 {
		return "buckets are empty"
	}
	return strings.Join(bucketInfo, "\n")
}

// Update inserts or refreshes a contact
func (rt *routingTable) Update(c Contact) {
	rt.mu.Lock() // write lock, because updates may cause bucket splits
	defer rt.mu.Unlock()

	b := rt.bucketFor(c.ID)

	if rt.shouldSplit(b, c) {
		left, right := b.Split()

		for i := range rt.buckets {
			if rt.buckets[i].Range.Start.Equals(left.Range.Start) {
				rt.buckets = append(rt.buckets[:i], append([]*bucket{left, right}, rt.buckets[i+1:]...)...)
				break
			}
		}

		if left.Range.Contains(c.ID) {
			b = left
		} else {
			b = right
		}
	}

	b.UpdatePeer(peer{Contact: c, Distance: rt.id.Xor(c.ID)}, true)
}

// Fresh refreshes a contact if its already in the routing table
func (rt *routingTable) Fresh(c Contact) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	rt.bucketFor(c.ID).UpdatePeer(peer{Contact: c, Distance: rt.id.Xor(c.ID)}, false)
}

// FailContact marks a contact as having failed, and removes it if it failed too many times
func (rt *routingTable) Fail(c Contact) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	rt.bucketFor(c.ID).FailContact(c.ID)
}

// GetClosest returns the closest `limit` contacts from the routing table.
// This is a locking wrapper around getClosest()
func (rt *routingTable) GetClosest(target bits.Bitmap, limit int) []Contact {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.getClosest(target, limit)
}

// getClosest returns the closest `limit` contacts from the routing table
func (rt *routingTable) getClosest(target bits.Bitmap, limit int) []Contact {
	var toSort []sortedContact
	for _, b := range rt.buckets {
		for _, c := range b.Contacts() {
			toSort = append(toSort, sortedContact{c, c.ID.Xor(target)})
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

// Count returns the number of contacts in the routing table
func (rt *routingTable) Count() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	count := 0
	for _, bucket := range rt.buckets {
		count += bucket.Len()
	}
	return count
}

// Len returns the number of buckets in the routing table
func (rt *routingTable) Len() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return len(rt.buckets)
}

func (rt *routingTable) bucketFor(target bits.Bitmap) *bucket {
	if rt.id.Equals(target) {
		panic("routing table does not have a bucket for its own id")
	}
	distance := target.Xor(rt.id)
	for _, b := range rt.buckets {
		if b.Range.Contains(distance) {
			return b
		}
	}
	panic("target is not contained in any buckets")
}

func (rt *routingTable) shouldSplit(b *bucket, c Contact) bool {
	if b.Has(c) {
		return false
	}
	if b.Len() >= bucketSize {
		if b.Range.Start.Equals(bits.Bitmap{}) { // this is the bucket covering our node id
			return true
		}
		kClosest := rt.getClosest(rt.id, bucketSize)
		kthClosest := kClosest[len(kClosest)-1]
		if rt.id.Closer(c.ID, kthClosest.ID) {
			return true
		}
	}
	return false
}

func (rt *routingTable) printBucketInfo() {
	fmt.Printf("there are %d contacts in %d buckets\n", rt.Count(), rt.Len())
	for i, b := range rt.buckets {
		fmt.Printf("bucket %d, %d contacts\n", i+1, len(b.peers))
		fmt.Printf("    start : %s\n", b.Range.Start.String())
		fmt.Printf("    stop  : %s\n", b.Range.End.String())
		fmt.Println("")
	}
}

func (rt *routingTable) GetIDsForRefresh(refreshInterval time.Duration) []bits.Bitmap {
	var bitmaps []bits.Bitmap
	for i, bucket := range rt.buckets {
		if bucket.NeedsRefresh(refreshInterval) {
			bitmaps = append(bitmaps, bits.Rand().Prefix(i, false))
		}
	}
	return bitmaps
}

const rtContactSep = "-"

type rtSave struct {
	ID       string   `json:"id"`
	Contacts []string `json:"contacts"`
}

func (rt *routingTable) MarshalJSON() ([]byte, error) {
	var data rtSave
	data.ID = rt.id.Hex()
	for _, b := range rt.buckets {
		for _, c := range b.Contacts() {
			data.Contacts = append(data.Contacts, strings.Join([]string{c.ID.Hex(), c.IP.String(), strconv.Itoa(c.Port)}, rtContactSep))
		}
	}
	return json.Marshal(data)
}

func (rt *routingTable) UnmarshalJSON(b []byte) error {
	var data rtSave
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}

	rt.id, err = bits.FromHex(data.ID)
	if err != nil {
		return errors.Prefix("decoding ID", err)
	}
	rt.reset()

	for _, s := range data.Contacts {
		parts := strings.Split(s, rtContactSep)
		if len(parts) != 3 {
			return errors.Err("decoding contact %s: wrong number of parts", s)
		}
		var c Contact
		c.ID, err = bits.FromHex(parts[0])
		if err != nil {
			return errors.Err("decoding contact %s: invalid ID: %s", s, err)
		}
		c.IP = net.ParseIP(parts[1])
		if c.IP == nil {
			return errors.Err("decoding contact %s: invalid IP", s)
		}
		c.Port, err = strconv.Atoi(parts[2])
		if err != nil {
			return errors.Err("decoding contact %s: invalid port: %s", s, err)
		}
		rt.Update(c)
	}

	return nil
}

// RoutingTableRefresh refreshes any buckets that need to be refreshed
func RoutingTableRefresh(n *Node, refreshInterval time.Duration, parentGrp *stop.Group) {
	done := stop.New()

	for _, id := range n.rt.GetIDsForRefresh(refreshInterval) {
		done.Add(1)
		go func(id bits.Bitmap) {
			defer done.Done()
			_, _, err := FindContacts(n, id, false, parentGrp)
			if err != nil {
				log.Error("error finding contact during routing table refresh - ", err)
			}
		}(id)
	}

	done.Wait()
	done.Stop()
}

func moveToBack(peers []peer, index int) {
	if index < 0 || len(peers) <= index+1 {
		return
	}
	p := peers[index]
	for i := index; i < len(peers)-1; i++ {
		peers[i] = peers[i+1]
	}
	peers[len(peers)-1] = p
}
