package dht

import (
	"context"
	"encoding/hex"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/lbryio/errors.go"
	"github.com/lbryio/lbry.go/stopOnce"
	"github.com/lbryio/lbry.go/util"

	"github.com/davecgh/go-spew/spew"
	"github.com/lyoshenka/bencode"
	log "github.com/sirupsen/logrus"
)

// packet represents the information receive from udp.
type packet struct {
	data  []byte
	raddr *net.UDPAddr
}

// UDPConn allows using a mocked connection to test sending/receiving data
type UDPConn interface {
	ReadFromUDP([]byte) (int, *net.UDPAddr, error)
	WriteToUDP([]byte, *net.UDPAddr) (int, error)
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	Close() error
}

type Node struct {
	// the node's id
	id Bitmap
	// UDP connection for sending and receiving data
	conn UDPConn
	// token manager
	tokens *tokenManager

	// map of outstanding transactions + mutex
	txLock       *sync.RWMutex
	transactions map[messageID]*transaction

	// routing table
	rt *routingTable
	// data store
	store *peerStore

	stop   *stopOnce.Stopper
	stopWG *sync.WaitGroup
}

// New returns a Node pointer.
func NewNode(id Bitmap) (*Node, error) {
	n := &Node{
		id:    id,
		rt:    newRoutingTable(id),
		store: newPeerStore(),

		txLock:       &sync.RWMutex{},
		transactions: make(map[messageID]*transaction),

		stop:   stopOnce.New(),
		stopWG: &sync.WaitGroup{},
		tokens: &tokenManager{},
	}

	n.tokens.Start(tokenSecretRotationInterval)
	return n, nil
}

func (n *Node) Connect(conn UDPConn) error {
	n.conn = conn

	//if dht.conf.PrintState > 0 {
	//	go func() {
	//		t := time.NewTicker(dht.conf.PrintState)
	//		for {
	//			dht.PrintState()
	//			select {
	//			case <-t.C:
	//			case <-dht.stop.Chan():
	//				return
	//			}
	//		}
	//	}()
	//}

	packets := make(chan packet)

	go func() {
		n.stopWG.Add(1)
		defer n.stopWG.Done()

		buf := make([]byte, udpMaxMessageLength)

		for {
			select {
			case <-n.stop.Chan():
				return
			default:
			}

			n.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)) // need this to periodically check shutdown chan
			n, raddr, err := n.conn.ReadFromUDP(buf)
			if err != nil {
				if e, ok := err.(net.Error); !ok || !e.Timeout() {
					log.Errorf("udp read error: %v", err)
				}
				continue
			} else if raddr == nil {
				log.Errorf("udp read with no raddr")
				continue
			}

			data := make([]byte, n)
			copy(data, buf[:n]) // slices use the same underlying array, so we need a new one for each packet

			packets <- packet{data: data, raddr: raddr}
		}
	}()

	go func() {
		n.stopWG.Add(1)
		defer n.stopWG.Done()

		var pkt packet

		for {
			select {
			case pkt = <-packets:
				n.handlePacket(pkt)
			case <-n.stop.Chan():
				return
			}
		}
	}()

	return nil
}

// Shutdown shuts down the node
func (n *Node) Shutdown() {
	log.Debugf("[%s] node shutting down", n.id.HexShort())
	n.stop.Stop()
	n.stopWG.Wait()
	n.tokens.Stop()
	n.conn.Close()
	log.Debugf("[%s] node stopped", n.id.HexShort())
}

// handlePacket handles packets received from udp.
func (n *Node) handlePacket(pkt packet) {
	//log.Debugf("[%s] Received message from %s (%d bytes) %s", n.id.HexShort(), pkt.raddr.String(), len(pkt.data), hex.EncodeToString(pkt.data))

	if !util.InSlice(string(pkt.data[0:5]), []string{"d1:0i", "di0ei"}) {
		log.Errorf("[%s] data is not a well-formatted dict: (%d bytes) %s", n.id.HexShort(), len(pkt.data), hex.EncodeToString(pkt.data))
		return
	}

	// TODO: test this stuff more thoroughly

	// the following is a bit of a hack, but it lets us avoid decoding every message twice
	// it depends on the data being a dict with 0 as the first key (so it starts with "d1:0i") and the message type as the first value

	switch pkt.data[5] {
	case '0' + requestType:
		request := Request{}
		err := bencode.DecodeBytes(pkt.data, &request)
		if err != nil {
			log.Errorf("[%s] error decoding request from %s: %s: (%d bytes) %s", n.id.HexShort(), pkt.raddr.String(), err.Error(), len(pkt.data), hex.EncodeToString(pkt.data))
			return
		}
		log.Debugf("[%s] query %s: received request from %s: %s(%s)", n.id.HexShort(), request.ID.HexShort(), request.NodeID.HexShort(), request.Method, request.ArgsDebug())
		n.handleRequest(pkt.raddr, request)

	case '0' + responseType:
		response := Response{}
		err := bencode.DecodeBytes(pkt.data, &response)
		if err != nil {
			log.Errorf("[%s] error decoding response from %s: %s: (%d bytes) %s", n.id.HexShort(), pkt.raddr.String(), err.Error(), len(pkt.data), hex.EncodeToString(pkt.data))
			return
		}
		log.Debugf("[%s] query %s: received response from %s: %s", n.id.HexShort(), response.ID.HexShort(), response.NodeID.HexShort(), response.ArgsDebug())
		n.handleResponse(pkt.raddr, response)

	case '0' + errorType:
		e := Error{}
		err := bencode.DecodeBytes(pkt.data, &e)
		if err != nil {
			log.Errorf("[%s] error decoding error from %s: %s: (%d bytes) %s", n.id.HexShort(), pkt.raddr.String(), err.Error(), len(pkt.data), hex.EncodeToString(pkt.data))
			return
		}
		log.Debugf("[%s] query %s: received error from %s: %s", n.id.HexShort(), e.ID.HexShort(), e.NodeID.HexShort(), e.ExceptionType)
		n.handleError(pkt.raddr, e)

	default:
		log.Errorf("[%s] invalid message type: %s", n.id.HexShort(), pkt.data[5])
		return
	}
}

// handleRequest handles the requests received from udp.
func (n *Node) handleRequest(addr *net.UDPAddr, request Request) {
	if request.NodeID.Equals(n.id) {
		log.Warn("ignoring self-request")
		return
	}

	switch request.Method {
	default:
		//		n.send(addr, makeError(t, protocolError, "invalid q"))
		log.Errorln("invalid request method")
		return
	case pingMethod:
		n.sendMessage(addr, Response{ID: request.ID, NodeID: n.id, Data: pingSuccessResponse})
	case storeMethod:
		// TODO: we should be sending the IP in the request, not just using the sender's IP
		// TODO: should we be using StoreArgs.NodeID or StoreArgs.Value.LbryID ???
		if n.tokens.Verify(request.StoreArgs.Value.Token, request.NodeID, addr) {
			n.store.Upsert(request.StoreArgs.BlobHash, Contact{id: request.StoreArgs.NodeID, ip: addr.IP, port: request.StoreArgs.Value.Port})
			n.sendMessage(addr, Response{ID: request.ID, NodeID: n.id, Data: storeSuccessResponse})
		} else {
			n.sendMessage(addr, Error{ID: request.ID, NodeID: n.id, ExceptionType: "invalid-token"})
		}
	case findNodeMethod:
		if request.Arg == nil {
			log.Errorln("request is missing arg")
			return
		}
		n.sendMessage(addr, Response{
			ID:       request.ID,
			NodeID:   n.id,
			Contacts: n.rt.GetClosest(*request.Arg, bucketSize),
		})

	case findValueMethod:
		if request.Arg == nil {
			log.Errorln("request is missing arg")
			return
		}

		res := Response{
			ID:     request.ID,
			NodeID: n.id,
			Token:  n.tokens.Get(request.NodeID, addr),
		}

		if contacts := n.store.Get(*request.Arg); len(contacts) > 0 {
			res.FindValueKey = request.Arg.RawString()
			res.Contacts = contacts
		} else {
			res.Contacts = n.rt.GetClosest(*request.Arg, bucketSize)
		}

		n.sendMessage(addr, res)
	}

	// nodes that send us requests should not be inserted, only refreshed.
	// the routing table must only contain "good" nodes, which are nodes that reply to our requests
	// if a node is already good (aka in the table), its fine to refresh it
	// http://www.bittorrent.org/beps/bep_0005.html#routing-table
	n.rt.UpdateIfExists(Contact{id: request.NodeID, ip: addr.IP, port: addr.Port})
}

// handleResponse handles responses received from udp.
func (n *Node) handleResponse(addr *net.UDPAddr, response Response) {
	tx := n.txFind(response.ID, addr)
	if tx != nil {
		tx.res <- response
	}

	n.rt.Update(Contact{id: response.NodeID, ip: addr.IP, port: addr.Port})
}

// handleError handles errors received from udp.
func (n *Node) handleError(addr *net.UDPAddr, e Error) {
	spew.Dump(e)
	n.rt.UpdateIfExists(Contact{id: e.NodeID, ip: addr.IP, port: addr.Port})
}

// send sends data to a udp address
func (n *Node) sendMessage(addr *net.UDPAddr, data Message) error {
	encoded, err := bencode.EncodeBytes(data)
	if err != nil {
		return errors.Err(err)
	}

	if req, ok := data.(Request); ok {
		log.Debugf("[%s] query %s: sending request to %s (%d bytes) %s(%s)",
			n.id.HexShort(), req.ID.HexShort(), addr.String(), len(encoded), req.Method, req.ArgsDebug())
	} else if res, ok := data.(Response); ok {
		log.Debugf("[%s] query %s: sending response to %s (%d bytes) %s",
			n.id.HexShort(), res.ID.HexShort(), addr.String(), len(encoded), res.ArgsDebug())
	} else {
		log.Debugf("[%s] (%d bytes) %s", n.id.HexShort(), len(encoded), spew.Sdump(data))
	}

	n.conn.SetWriteDeadline(time.Now().Add(time.Second * 15))

	_, err = n.conn.WriteToUDP(encoded, addr)
	return errors.Err(err)
}

// transaction represents a single query to the dht. it stores the queried contact, the request, and the response channel
type transaction struct {
	contact Contact
	req     Request
	res     chan Response
}

// insert adds a transaction to the manager.
func (n *Node) txInsert(tx *transaction) {
	n.txLock.Lock()
	defer n.txLock.Unlock()
	n.transactions[tx.req.ID] = tx
}

// delete removes a transaction from the manager.
func (n *Node) txDelete(id messageID) {
	n.txLock.Lock()
	defer n.txLock.Unlock()
	delete(n.transactions, id)
}

// Find finds a transaction for the given id. it optionally ensures that addr matches contact from transaction
func (n *Node) txFind(id messageID, addr *net.UDPAddr) *transaction {
	n.txLock.RLock()
	defer n.txLock.RUnlock()

	// TODO: also check that the response's nodeid matches the id you thought you sent to?

	t, ok := n.transactions[id]
	if !ok || (addr != nil && t.contact.Addr().String() != addr.String()) {
		return nil
	}

	return t
}

// SendAsync sends a transaction and returns a channel that will eventually contain the transaction response
// The response channel is closed when the transaction is completed or times out.
func (n *Node) SendAsync(ctx context.Context, contact Contact, req Request) <-chan *Response {
	if contact.id.Equals(n.id) {
		log.Error("sending query to self")
		return nil
	}

	ch := make(chan *Response, 1)

	go func() {
		defer close(ch)

		req.ID = newMessageID()
		req.NodeID = n.id
		tx := &transaction{
			contact: contact,
			req:     req,
			res:     make(chan Response),
		}

		n.txInsert(tx)
		defer n.txDelete(tx.req.ID)

		for i := 0; i < udpRetry; i++ {
			if err := n.sendMessage(contact.Addr(), tx.req); err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") { // this only happens on localhost. real UDP has no connections
					log.Error("send error: ", err)
				}
				continue
			}

			select {
			case res := <-tx.res:
				ch <- &res
				return
			case <-ctx.Done():
				return
			case <-time.After(udpTimeout):
			}
		}

		// if request timed out each time
		n.rt.Remove(tx.contact.id)
	}()

	return ch
}

// Send sends a transaction and blocks until the response is available. It returns a response, or nil
// if the transaction timed out.
func (n *Node) Send(contact Contact, req Request) *Response {
	return <-n.SendAsync(context.Background(), contact, req)
}

// Count returns the number of transactions in the manager
func (n *Node) CountActiveTransactions() int {
	n.txLock.Lock()
	defer n.txLock.Unlock()
	return len(n.transactions)
}
