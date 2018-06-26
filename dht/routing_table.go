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
	"math/big"
)

// TODO: if routing table is ever empty (aka the node is isolated), it should re-bootstrap

// TODO: use a tree with bucket splitting instead of a fixed bucket list. include jack's optimization (see link in commit mesg)
// https://github.com/lbryio/lbry/pull/1211/commits/341b27b6d21ac027671d42458826d02735aaae41

// peer is a contact with extra freshness information
type peer struct {
	Contact      Contact
	LastActivity time.Time
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
	bucketRange *bits.Range
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
}

func newRoutingTable(id bits.Bitmap) *routingTable {
	var rt routingTable
	rt.id = id
	rt.reset()
	return &rt
}

func (rt *routingTable) reset() {
	start := big.NewInt(0)
	end := big.NewInt(1)
	end.Lsh(end, bits.NumBits)
	end.Sub(end, big.NewInt(1))
	rt.buckets = []bucket{}
	rt.buckets = append(rt.buckets, bucket{
		peers: make([]peer, 0, bucketSize),
		lock:  &sync.RWMutex{},
		bucketRange: &bits.Range{
			Start: bits.FromBigP(start),
			End: bits.FromBigP(end),
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
			bucketInfo = append(bucketInfo, fmt.Sprintf("Bucket %d: (%d) %s", i, len(contacts), strings.Join(s, ", ")))
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

// GetClosest returns the closest `limit` contacts from the routing table
// It marks each bucket it accesses as having been accessed
func (rt *routingTable) GetClosest(target bits.Bitmap, limit int) []Contact {
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
	for _, bucket := range rt.buckets {
		count += bucket.Len()
	}
	return count
}

// BucketRanges returns a slice of ranges, where the `start` of each range is the smallest id that can
// go in that bucket, and the `end` is the largest id
func (rt *routingTable) BucketRanges() []bits.Range {
	ranges := make([]bits.Range, len(rt.buckets))
	for i, b := range rt.buckets {
		ranges[i] = *b.bucketRange
	}
	return ranges
}

func (rt *routingTable) bucketNumFor(target bits.Bitmap) int {
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
	return &rt.buckets[rt.bucketNumFor(target)]
}

func (rt *routingTable) shouldSplit(target bits.Bitmap) bool {
	bucketIndex := rt.bucketNumFor(target)
	if len(rt.buckets[bucketIndex].peers) >= bucketSize {
		if bucketIndex == 0 { // this is the bucket covering our node id
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
	if len(rt.buckets[rt.bucketNumFor(c.ID)].peers) < bucketSize {
		rt.buckets[rt.bucketNumFor(c.ID)].UpdateContact(c, true)
	} else if rt.shouldSplit(c.ID) {
		rt.recursiveInsertContact(c)
	}
}

func (rt *routingTable) recursiveInsertContact(c Contact) {
	bucketIndex := rt.bucketNumFor(c.ID)
	b := rt.buckets[bucketIndex]
	min := b.bucketRange.Start.Big()
	max := b.bucketRange.End.Big()

	midpoint := max.Sub(max, min)
	midpoint.Div(midpoint, big.NewInt(2))

	// re-size the bucket to be split
	b.bucketRange.Start = bits.FromBigP(min)
	b.bucketRange.End = bits.FromBigP(midpoint.Sub(midpoint, big.NewInt(1)))

	movedPeers := []peer{}
	resizedPeers := []peer{}

	// set the re-sized bucket to only have peers still in range
	for _, p := range b.peers {
		if rt.bucketNumFor(p.Contact.ID) != bucketIndex {
			movedPeers = append(movedPeers, p)
		} else {
			resizedPeers = append(resizedPeers, p)
		}
	}
	b.peers = resizedPeers

	// add the new bucket
	insert := bucket{
		peers: make([]peer, 0, bucketSize),
		lock:  &sync.RWMutex{},
		bucketRange: &bits.Range{
			Start: bits.FromBigP(midpoint),
			End:   bits.FromBigP(max),
		},
	}
	rt.buckets = append(rt.buckets[:bucketIndex], append([]bucket{insert}, rt.buckets[bucketIndex:]...)...)

	// re-insert the contacts that where out of range of the split bucket
	for _, p := range movedPeers {
		rt.insertContact(p.Contact)
	}

	// insert the new contact
	rt.insertContact(c)
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
