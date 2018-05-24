package dht

import (
	"math/rand"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	bootstrapDefaultRefreshDuration = 15 * time.Minute
)

type BootstrapNode struct {
	Node

	initialPingInterval time.Duration
	checkInterval       time.Duration

	nlock    *sync.RWMutex
	nodes    []peer
	nodeKeys map[Bitmap]int
}

// New returns a BootstrapNode pointer.
func NewBootstrapNode(id Bitmap, initialPingInterval, rePingInterval time.Duration) *BootstrapNode {
	b := &BootstrapNode{
		Node: *NewNode(id),

		initialPingInterval: initialPingInterval,
		checkInterval:       rePingInterval,

		nlock:    &sync.RWMutex{},
		nodes:    make([]peer, 0),
		nodeKeys: make(map[Bitmap]int),
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

// ypsert adds the contact to the list, or updates the lastPinged time
func (b *BootstrapNode) upsert(c Contact) {
	b.nlock.Lock()
	defer b.nlock.Unlock()

	if i, exists := b.nodeKeys[c.ID]; exists {
		log.Debugf("[%s] bootstrap: touching contact %s", b.id.HexShort(), b.nodes[i].Contact.ID.HexShort())
		b.nodes[i].Touch()
		return
	}

	log.Debugf("[%s] bootstrap: adding new contact %s", b.id.HexShort(), c.ID.HexShort())
	b.nodeKeys[c.ID] = len(b.nodes)
	b.nodes = append(b.nodes, peer{c, time.Now(), 0})
}

// remove removes the contact from the list
func (b *BootstrapNode) remove(c Contact) {
	b.nlock.Lock()
	defer b.nlock.Unlock()

	i, exists := b.nodeKeys[c.ID]
	if !exists {
		return
	}

	log.Debugf("[%s] bootstrap: removing contact %s", b.id.HexShort(), c.ID.HexShort())
	b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
	delete(b.nodeKeys, c.ID)
}

// get returns up to `limit` random contacts from the list
func (b *BootstrapNode) get(limit int) []Contact {
	b.nlock.RLock()
	defer b.nlock.RUnlock()

	if len(b.nodes) < limit {
		limit = len(b.nodes)
	}

	ret := make([]Contact, limit)
	for i, k := range randKeys(len(b.nodes))[:limit] {
		ret[i] = b.nodes[k].Contact
	}

	return ret
}

// ping pings a node. if the node responds, it is added to the list. otherwise, it is removed
func (b *BootstrapNode) ping(c Contact) {
	b.stopWG.Add(1)
	defer b.stopWG.Done()

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
		b.sendMessage(addr, Response{ID: request.ID, NodeID: b.id, Data: pingSuccessResponse})
	case findNodeMethod:
		if request.Arg == nil {
			log.Errorln("request is missing arg")
			return
		}
		b.sendMessage(addr, Response{
			ID:       request.ID,
			NodeID:   b.id,
			Contacts: b.get(bucketSize),
		})
	}

	go func() {
		log.Debugf("[%s] bootstrap: queuing %s to ping", b.id.HexShort(), request.NodeID.HexShort())
		<-time.After(b.initialPingInterval)
		b.ping(Contact{ID: request.NodeID, IP: addr.IP, Port: addr.Port})
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
