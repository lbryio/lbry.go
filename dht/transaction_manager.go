package dht

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// query represents the query data included queried node and query-formed data.
type transaction struct {
	node *Node
	req  *Request
	res  chan *Response
}

// transactionManager represents the manager of transactions.
type transactionManager struct {
	lock         *sync.RWMutex
	transactions map[string]*transaction
	dht          *DHT
}

// newTransactionManager returns new transactionManager pointer.
func newTransactionManager(dht *DHT) *transactionManager {
	return &transactionManager{
		lock:         &sync.RWMutex{},
		transactions: make(map[string]*transaction),
		dht:          dht,
	}
}

// insert adds a transaction to transactionManager.
func (tm *transactionManager) insert(trans *transaction) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	tm.transactions[trans.req.ID] = trans
}

// delete removes a transaction from transactionManager.
func (tm *transactionManager) delete(transID string) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	delete(tm.transactions, transID)
}

// find transaction for id. optionally ensure that addr matches node from transaction
func (tm *transactionManager) Find(id string, addr *net.UDPAddr) *transaction {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	t, ok := tm.transactions[id]
	if !ok {
		return nil
	} else if addr != nil && t.node.Addr().String() != addr.String() {
		return nil
	}

	return t
}

func (tm *transactionManager) Send(node *Node, req *Request) *Response {
	if node.id.Equals(tm.dht.node.id) {
		log.Error("sending query to self")
		return nil
	}

	req.ID = newMessageID()
	trans := &transaction{
		node: node,
		req:  req,
		res:  make(chan *Response),
	}

	tm.insert(trans)
	defer tm.delete(trans.req.ID)

	for i := 0; i < udpRetry; i++ {
		if err := send(tm.dht, trans.node.Addr(), *trans.req); err != nil {
			log.Error(err)
			break
		}

		select {
		case res := <-trans.res:
			return res
		case <-time.After(udpTimeout):
		}
	}

	tm.dht.rt.RemoveByID(trans.node.id)
	return nil
}

// Count returns the number of transactions in the manager
func (tm *transactionManager) Count() int {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	return len(tm.transactions)
}
