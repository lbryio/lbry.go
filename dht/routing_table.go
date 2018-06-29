package dht

import (
	"encoding/json"
	"fmt"
	"math/big"
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

// peer is a contact with extra freshness information
type peer struct {
	Contact      Contact
	LastActivity time.Time
	// LastReplied time.Time
	// LastRequested time.Time
	// LastFailure time.Time
	// SecondLastFailure time.Time
	NumFailures  int

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
	lock        *sync.RWMutex
	peers       []peer
	lastUpdate  time.Time
	bucketRange bits.Range
}

// Len returns the number of peers in the bucket
func (b bucket) Len() int {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return len(b.peers)
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

// UpdateContact marks a contact as having been successfully contacted. if insertIfNew and the contact is does not exist yet, it is inserted
func (b *bucket) UpdateContact(c Contact, insertIfNew bool) {
	b.lock.Lock()
	defer b.lock.Unlock()

	// TODO: verify the peer is in the bucket key range

	peerIndex := find(c.ID, b.peers)
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
			peer := peer{Contact: c}
			peer.Touch()
			b.peers = append(b.peers, peer)
		}
	}
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

type routingTable struct {
	id      bits.Bitmap
	buckets []bucket
	lock *sync.RWMutex
}

func newRoutingTable(id bits.Bitmap) *routingTable {
	var rt routingTable
	rt.id = id
	rt.lock = &sync.RWMutex{}
	rt.reset()
	return &rt
}

func (rt *routingTable) reset() {
	rt.Lock()
	defer rt.Unlock()
	newBucketLock := &sync.RWMutex{}
	newBucketLock.Lock()
	rt.buckets = []bucket{}
	rt.buckets = append(rt.buckets, bucket{
		peers: make([]peer, 0, bucketSize),
		lock:  newBucketLock,
		bucketRange: bits.Range{
			Start: bits.MinP(),
			End: bits.MaxP(),
		},
	})
}

func (rt *routingTable) BucketInfo() string {
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
	rt.insertContact(c)
}

// Fresh refreshes a contact if its already in the routing table
func (rt *routingTable) Fresh(c Contact) {
	rt.bucketFor(c.ID).UpdateContact(c, false)
}

// FailContact marks a contact as having failed, and removes it if it failed too many times
func (rt *routingTable) Fail(c Contact) {
	rt.bucketFor(c.ID).FailContact(c.ID)
}

func (rt *routingTable) getClosestToUs(limit int) []Contact {
	contacts := []Contact{}
	toSort := []sortedContact{}
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	for _, bucket := range rt.buckets {
		toSort = []sortedContact{}
		toSort = appendContacts(toSort, bucket, rt.id)
		sort.Sort(byXorDistance(toSort))
		for _, sorted := range toSort {
			contacts = append(contacts, sorted.contact)
			if len(contacts) >= limit {
				break
			}
		}
	}
	return contacts
}

// GetClosest returns the closest `limit` contacts from the routing table
// It marks each bucket it accesses as having been accessed
func (rt *routingTable) GetClosest(target bits.Bitmap, limit int) []Contact {
	if target == rt.id {
		return rt.getClosestToUs(limit)
	}
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	toSort := []sortedContact{}
	for _, b := range rt.buckets {
		toSort = appendContacts(toSort, b, target)
	}
	sort.Sort(byXorDistance(toSort))
	contacts := []Contact{}
	for _, sorted := range toSort {
		contacts = append(contacts, sorted.contact)
		if len(contacts) >= limit {
			break
		}
	}
	return contacts
}

func appendContacts(contacts []sortedContact, b bucket, target bits.Bitmap) []sortedContact {
	for _, contact := range b.Contacts() {
		contacts = append(contacts, sortedContact{contact, contact.ID.Xor(target)})
	}
	return contacts
}

// Count returns the number of contacts in the routing table
func (rt *routingTable) Count() int {
	count := 0
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	for _, bucket := range rt.buckets {
		count += bucket.Len()
	}
	return count
}

// Len returns the number of buckets in the routing table
func (rt *routingTable) Len() int {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	return len(rt.buckets)
}

// BucketRanges returns a slice of ranges, where the `start` of each range is the smallest id that can
// go in that bucket, and the `end` is the largest id
func (rt *routingTable) BucketRanges() []bits.Range {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	ranges := make([]bits.Range, len(rt.buckets))
	for i, b := range rt.buckets {
		ranges[i] = b.bucketRange
	}
	return ranges
}

func (rt *routingTable) bucketNumFor(target bits.Bitmap) int {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	if rt.id.Equals(target) {
		panic("routing table does not have a bucket for its own id")
	}
	distance := target.Xor(rt.id)
	for i, b := range rt.buckets {
		if b.bucketRange.Start.Cmp(distance) <= 0 && b.bucketRange.End.Cmp(distance) >= 0 {
			return i
		}
	}
	panic("target value overflows the key space")
}

func (rt *routingTable) bucketFor(target bits.Bitmap) *bucket {
	bucketIndex := rt.bucketNumFor(target)
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	return &rt.buckets[bucketIndex]
}

func (rt *routingTable) shouldSplit(target bits.Bitmap) bool {
	b := rt.bucketFor(target)
	if b.Len() >= bucketSize {
		if b.bucketRange.Start.Equals(bits.MinP()) { // this is the bucket covering our node id
			return true
		}
		kClosest := rt.GetClosest(rt.id, bucketSize)
		kthClosest := kClosest[len(kClosest) - 1]
		if target.Xor(rt.id).Cmp(kthClosest.ID.Xor(rt.id)) < 0 {
			return true // the kth closest contact is further than this one
		}
	}
	return false
}

func (rt *routingTable) insertContact(c Contact) {
	bucketIndex := rt.bucketNumFor(c.ID)
	peersInBucket :=rt.buckets[bucketIndex].Len()
	if peersInBucket < bucketSize {
		rt.buckets[rt.bucketNumFor(c.ID)].UpdateContact(c, true)
	} else if peersInBucket >= bucketSize && rt.shouldSplit(c.ID) {
		rt.splitBucket(bucketIndex)
		rt.insertContact(c)
		rt.popEmptyBuckets()
	}
}

func (rt * routingTable) Lock() {
	rt.lock.Lock()
	for _, buk := range rt.buckets {
		buk.lock.Lock()
	}
}

func (rt * routingTable) Unlock() {
	rt.lock.Unlock()
	for _, buk := range rt.buckets {
		buk.lock.Unlock()
	}
}

func (rt *routingTable) splitBucket(bucketIndex int) {
	rt.Lock()
	defer rt.Unlock()

	b := rt.buckets[bucketIndex]
	min := b.bucketRange.Start.Big()
	max := b.bucketRange.End.Big()
	midpoint := &big.Int{}
	midpoint.Sub(max, min)
	midpoint.Div(midpoint, big.NewInt(2))
	midpoint.Add(midpoint, min)
	midpointPlusOne := &big.Int{}
	midpointPlusOne.Add(midpointPlusOne, min)
	midpointPlusOne.Add(midpoint, big.NewInt(1))

	first_half := rt.buckets[:bucketIndex+1]
	second_half := []bucket{}
	for i := bucketIndex + 1; i < len(rt.buckets); i++ {
		second_half = append(second_half, rt.buckets[i])
	}

	copiedPeers := []peer{}
	copy(copiedPeers, b.peers)
	b.peers = []peer{}

	rt.buckets = []bucket{}
	for _, buk := range first_half {
		rt.buckets = append(rt.buckets, buk)
	}
	newBucketLock := &sync.RWMutex{}
	newBucketLock.Lock() // will be unlocked by the deferred rt.Unlock()
	newBucket := bucket{
		peers: make([]peer, 0, bucketSize),
		lock: newBucketLock,
		bucketRange: bits.Range{
			Start: bits.FromBigP(midpointPlusOne),
			End:   bits.FromBigP(max),
		},
	}
	rt.buckets = append(rt.buckets, newBucket)
	for _, buk := range second_half {
		rt.buckets = append(rt.buckets, buk)
	}
	// re-size the bucket to be split
	rt.buckets[bucketIndex].bucketRange.Start = bits.FromBigP(min)
	rt.buckets[bucketIndex].bucketRange.End = bits.FromBigP(midpoint)

	// re-insert the contacts that were in the re-sized bucket
	for _, p := range copiedPeers {
		rt.insertContact(p.Contact)
	}
}

func (rt *routingTable) printBucketInfo() {
	for i, b := range rt.buckets {
		fmt.Printf("bucket %d, %d contacts\n", i + 1, len(b.peers))
		fmt.Printf("    start : %s\n", b.bucketRange.Start.String())
		fmt.Printf("    stop  : %s\n", b.bucketRange.End.String())
		fmt.Println("")
	}
}

func (rt *routingTable) popBucket(bucketIndex int) {
	canGoLower := bucketIndex >= 1
	canGoHigher := len(rt.buckets) - 1 > bucketIndex

	if canGoLower && !canGoHigher {
		// raise the end of bucket[bucketIndex-1]
		rt.buckets[bucketIndex-1].bucketRange.End = bits.FromBigP(rt.buckets[bucketIndex].bucketRange.End.Big())
	} else if !canGoLower && canGoHigher {
		// lower the start of bucket[bucketIndex+1]
		rt.buckets[bucketIndex+1].bucketRange.Start = bits.FromBigP(rt.buckets[bucketIndex].bucketRange.Start.Big())
	} else if canGoLower && canGoHigher {
		// raise the end of bucket[bucketIndex-1] and lower the start of bucket[bucketIndex+1] to the
		// midpoint of the range covered by bucket[bucketIndex]
		midpoint := &big.Int{}
		midpoint.Sub(rt.buckets[bucketIndex].bucketRange.End.Big(), rt.buckets[bucketIndex].bucketRange.Start.Big())
		midpoint.Div(midpoint, big.NewInt(2))
		midpointPlusOne := &big.Int{}
		midpointPlusOne.Add(midpoint, big.NewInt(1))
		rt.buckets[bucketIndex-1].bucketRange.End = bits.FromBigP(midpoint)
		rt.buckets[bucketIndex+1].bucketRange.Start = bits.FromBigP(midpointPlusOne)
	} else {
		return
	}
	// pop the bucket
	rt.buckets = rt.buckets[:bucketIndex+copy(rt.buckets[bucketIndex:], rt.buckets[bucketIndex+1:])]
}

func (rt *routingTable) popNextEmptyBucket() bool {
	for bucketIndex := 0; bucketIndex < len(rt.buckets); bucketIndex += 1 {
		if len(rt.buckets[bucketIndex].peers) == 0 {
			rt.popBucket(bucketIndex)
			return true
		}
	}
	return false
}

func (rt *routingTable) popEmptyBuckets() {
	rt.Lock()
	defer rt.Unlock()

	if len(rt.buckets) > 1 {
		popBuckets := rt.popNextEmptyBucket()
		for popBuckets == true {
			popBuckets = rt.popNextEmptyBucket()
		}
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
