package dht

import (
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/lbryio/reflector.go/dht/bits"
	log "github.com/sirupsen/logrus"
)

const (
	bootstrapDefaultRefreshDuration = 15 * time.Minute
)

// BootstrapNode is a configured node setup for testing.
type BootstrapNode struct {
	Node

	initialPingInterval time.Duration
	checkInterval       time.Duration

	nlock   *sync.RWMutex
	nodes   map[bits.Bitmap]*peer
	nodeIDs []bits.Bitmap // necessary for efficient random ID selection
}

// NewBootstrapNode returns a BootstrapNode pointer.
func NewBootstrapNode(id bits.Bitmap, initialPingInterval, rePingInterval time.Duration) *BootstrapNode {
	b := &BootstrapNode{
		Node: *NewNode(id),

		initialPingInterval: initialPingInterval,
		checkInterval:       rePingInterval,

		nlock:   &sync.RWMutex{},
		nodes:   make(map[bits.Bitmap]*peer),
		nodeIDs: make([]bits.Bitmap, 0),
	}

	b.requestHandler = b.handleRequest

	return b
}

// Add manually adds a contact
func (b *BootstrapNode) Add(c Contact) {
	b.upsert(c)
}

// Connect connects to the given connection and starts any background threads necessary
func (b *BootstrapNode) Connect(conn UDPConn) error {
	err := b.Node.Connect(conn)
	if err != nil {
		return err
	}

	log.Debugf("[%s] bootstrap: node connected", b.id.HexShort())

	go func() {
		t := time.NewTicker(b.checkInterval / 5)
		for {
			select {
			case <-t.C:
				b.check()
			case <-b.stop.Ch():
				return
			}
		}
	}()

	return nil
}

// upsert adds the contact to the list, or updates the lastPinged time
func (b *BootstrapNode) upsert(c Contact) {
	b.nlock.Lock()
	defer b.nlock.Unlock()

	if node, exists := b.nodes[c.ID]; exists {
		log.Debugf("[%s] bootstrap: touching contact %s", b.id.HexShort(), node.Contact.ID.HexShort())
		node.Touch()
		return
	}

	log.Debugf("[%s] bootstrap: adding new contact %s", b.id.HexShort(), c.ID.HexShort())
	b.nodes[c.ID] = &peer{c, time.Now(), 0}
	b.nodeIDs = append(b.nodeIDs, c.ID)
}

// remove removes the contact from the list
func (b *BootstrapNode) remove(c Contact) {
	b.nlock.Lock()
	defer b.nlock.Unlock()

	_, exists := b.nodes[c.ID]
	if !exists {
		return
	}

	log.Debugf("[%s] bootstrap: removing contact %s", b.id.HexShort(), c.ID.HexShort())
	delete(b.nodes, c.ID)
	for i := range b.nodeIDs {
		if b.nodeIDs[i].Equals(c.ID) {
			b.nodeIDs = append(b.nodeIDs[:i], b.nodeIDs[i+1:]...)
			break
		}
	}
}

// get returns up to `limit` random contacts from the list
func (b *BootstrapNode) get(limit int) []Contact {
	b.nlock.RLock()
	defer b.nlock.RUnlock()

	if len(b.nodes) < limit {
		limit = len(b.nodes)
	}

	ret := make([]Contact, limit)
	for i, k := range randKeys(len(b.nodeIDs))[:limit] {
		ret[i] = b.nodes[b.nodeIDs[k]].Contact
	}

	return ret
}

// ping pings a node. if the node responds, it is added to the list. otherwise, it is removed
func (b *BootstrapNode) ping(c Contact) {
	log.Debugf("[%s] bootstrap: pinging %s", b.id.HexShort(), c.ID.HexShort())
	b.stop.Add(1)
	defer b.stop.Done()

	resCh, cancel := b.SendCancelable(c, Request{Method: pingMethod})

	var res *Response

	select {
	case res = <-resCh:
	case <-b.stop.Ch():
		cancel()
		return
	}

	if res != nil && res.Data == pingSuccessResponse {
		b.upsert(c)
	} else {
		b.remove(c)
	}
}

func (b *BootstrapNode) check() {
	b.nlock.RLock()
	defer b.nlock.RUnlock()

	for i := range b.nodes {
		if !b.nodes[i].ActiveInLast(b.checkInterval) {
			go b.ping(b.nodes[i].Contact)
		}
	}
}

// handleRequest handles the requests received from udp.
func (b *BootstrapNode) handleRequest(addr *net.UDPAddr, request Request) {
	switch request.Method {
	case pingMethod:
		err := b.sendMessage(addr, Response{ID: request.ID, NodeID: b.id, Data: pingSuccessResponse})
		if err != nil {
			log.Error("error sending response message - ", err)
		}
	case findNodeMethod:
		if request.Arg == nil {
			log.Errorln("request is missing arg")
			return
		}

		err := b.sendMessage(addr, Response{
			ID:       request.ID,
			NodeID:   b.id,
			Contacts: b.get(bucketSize),
		})
		if err != nil {
			log.Error("error sending 'findnodemethod' response message - ", err)
		}
	}

	go func() {
		b.nlock.RLock()
		_, exists := b.nodes[request.NodeID]
		b.nlock.RUnlock()
		if !exists {
			log.Debugf("[%s] bootstrap: queuing %s to ping", b.id.HexShort(), request.NodeID.HexShort())
			<-time.After(b.initialPingInterval)
			b.nlock.RLock()
			_, exists = b.nodes[request.NodeID]
			b.nlock.RUnlock()
			if !exists {
				b.ping(Contact{ID: request.NodeID, IP: addr.IP, Port: addr.Port})
			}
		}
	}()
}

func randKeys(max int) []int {
	keys := make([]int, max)
	for k := range keys {
		keys[k] = k
	}
	rand.Shuffle(max, func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	return keys
}
