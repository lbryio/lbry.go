package dht

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// transaction represents a single query to the dht. it stores the queried node, the request, and the response channel
type transaction struct {
	node Node
	req  Request
	res  chan Response
}

// transactionManager keeps track of the outstanding transactions
type transactionManager struct {
	dht          *DHT
	lock         *sync.RWMutex
	transactions map[messageID]*transaction
}

// newTransactionManager returns a new transactionManager
func newTransactionManager(dht *DHT) *transactionManager {
	return &transactionManager{
		lock:         &sync.RWMutex{},
		transactions: make(map[messageID]*transaction),
		dht:          dht,
	}
}

// insert adds a transaction to the manager.
func (tm *transactionManager) insert(tx *transaction) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	tm.transactions[tx.req.ID] = tx
}

// delete removes a transaction from the manager.
func (tm *transactionManager) delete(id messageID) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	delete(tm.transactions, id)
}

// Find finds a transaction for the given id. it optionally ensures that addr matches node from transaction
func (tm *transactionManager) Find(id messageID, addr *net.UDPAddr) *transaction {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	t, ok := tm.transactions[id]
	if !ok || (addr != nil && t.node.Addr().String() != addr.String()) {
		return nil
	}

	return t
}

// SendAsync sends a transaction and returns a channel that will eventually contain the transaction response
// The response channel is closed when the transaction is completed or times out.
func (tm *transactionManager) SendAsync(ctx context.Context, node Node, req Request) <-chan *Response {
	if node.id.Equals(tm.dht.node.id) {
		log.Error("sending query to self")
		return nil
	}

	ch := make(chan *Response, 1)

	go func() {
		defer close(ch)

		req.ID = newMessageID()
		req.NodeID = tm.dht.node.id
		tx := &transaction{
			node: node,
			req:  req,
			res:  make(chan Response),
		}

		tm.insert(tx)
		defer tm.delete(tx.req.ID)

		for i := 0; i < udpRetry; i++ {
			if err := send(tm.dht, node.Addr(), tx.req); err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") { // this only happens on localhost. real UDP has no connections
					log.Error("send error: ", err)
				}
				continue // try again? return?
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
		tm.dht.rt.RemoveByID(tx.node.id)
	}()

	return ch
}

// Send sends a transaction and blocks until the response is available. It returns a response, or nil
// if the transaction timed out.
func (tm *transactionManager) Send(node Node, req Request) *Response {
	return <-tm.SendAsync(context.Background(), node, req)
}

// Count returns the number of transactions in the manager
func (tm *transactionManager) Count() int {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	return len(tm.transactions)
}
