package dht

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/zeebo/bencode"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	pingMethod      = "ping"
	storeMethod     = "store"
	findNodeMethod  = "findNode"
	findValueMethod = "findValue"
)

const (
	generalError = 201 + iota
	serverError
	protocolError
	unknownError
)

const (
	requestType  = 0
	responseType = 1
	errorType    = 2
)

const (
	// these are strings because bencode requires bytestring keys
	headerTypeField      = "0"
	headerMessageIDField = "1"
	headerNodeIDField    = "2"
	headerPayloadField   = "3"
	headerArgsField      = "4"
)

type Message interface {
	GetID() string
	Encode() ([]byte, error)
}

type Request struct {
	ID     string
	NodeID string
	Method string
	Args   []string
}

func (r Request) GetID() string { return r.ID }
func (r Request) Encode() ([]byte, error) {
	return bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      requestType,
		headerMessageIDField: r.ID,
		headerNodeIDField:    r.NodeID,
		headerPayloadField:   r.Method,
		headerArgsField:      r.Args,
	})
}

type findNodeDatum struct {
	ID   string
	IP   string
	Port int
}
type Response struct {
	ID           string
	NodeID       string
	Data         string
	FindNodeData []findNodeDatum
}

func (r Response) GetID() string { return r.ID }
func (r Response) Encode() ([]byte, error) {
	data := map[string]interface{}{
		headerTypeField:      responseType,
		headerMessageIDField: r.ID,
		headerNodeIDField:    r.NodeID,
	}
	if r.Data != "" {
		data[headerPayloadField] = r.Data
	} else {
		var nodes []interface{}
		for _, n := range r.FindNodeData {
			nodes = append(nodes, []interface{}{n.ID, n.IP, n.Port})
		}
		data[headerPayloadField] = nodes
	}

	log.Info("Response data is ")
	spew.Dump(data)
	return bencode.EncodeBytes(data)
}

type Error struct {
	ID            string
	NodeID        string
	Response      []string
	ExceptionType string
}

func (e Error) GetID() string { return e.ID }
func (e Error) Encode() ([]byte, error) {
	return bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      errorType,
		headerMessageIDField: e.ID,
		headerNodeIDField:    e.NodeID,
		headerPayloadField:   e.ExceptionType,
		headerArgsField:      e.Response,
	})
}

// packet represents the information receive from udp.
type packet struct {
	data  []byte
	raddr *net.UDPAddr
}

// token represents the token when response getPeers request.
type token struct {
	data       string
	createTime time.Time
}

// tokenManager managers the tokens.
type tokenManager struct {
	*syncedMap
	expiredAfter time.Duration
	dht          *DHT
}

// newTokenManager returns a new tokenManager.
func newTokenManager(expiredAfter time.Duration, dht *DHT) *tokenManager {
	return &tokenManager{
		syncedMap:    newSyncedMap(),
		expiredAfter: expiredAfter,
		dht:          dht,
	}
}

// token returns a token. If it doesn't exist or is expired, it will add a
// new token.
func (tm *tokenManager) token(addr *net.UDPAddr) string {
	v, ok := tm.Get(addr.IP.String())
	tk, _ := v.(token)

	if !ok || time.Now().Sub(tk.createTime) > tm.expiredAfter {
		tk = token{
			data:       randomString(5),
			createTime: time.Now(),
		}

		tm.Set(addr.IP.String(), tk)
	}

	return tk.data
}

// clear removes expired tokens.
func (tm *tokenManager) clear() {
	for range time.Tick(time.Minute * 3) {
		keys := make([]interface{}, 0, 100)

		for item := range tm.Iter() {
			if time.Now().Sub(item.val.(token).createTime) > tm.expiredAfter {
				keys = append(keys, item.key)
			}
		}

		tm.DeleteMulti(keys)
	}
}

// check returns whether the token is valid.
func (tm *tokenManager) check(addr *net.UDPAddr, tokenString string) bool {
	key := addr.IP.String()
	v, ok := tm.Get(key)
	tk, _ := v.(token)

	if ok {
		tm.Delete(key)
	}

	return ok && tokenString == tk.data
}

// send sends data to the udp.
func send(dht *DHT, addr *net.UDPAddr, data Message) error {
	log.Infof("Sending %s", spew.Sdump(data))
	encoded, err := data.Encode()
	if err != nil {
		return err
	}
	log.Infof("Encoded: %s", string(encoded))

	dht.conn.SetWriteDeadline(time.Now().Add(time.Second * 15))

	_, err = dht.conn.WriteToUDP(encoded, addr)
	return err
}

// query represents the query data included queried node and query-formed data.
type query struct {
	node    *node
	request Request
}

// transaction implements transaction.
type transaction struct {
	*query
	id       string
	response chan struct{}
}

// transactionManager represents the manager of transactions.
type transactionManager struct {
	*sync.RWMutex
	transactions *syncedMap
	index        *syncedMap
	cursor       uint64
	maxCursor    uint64
	queryChan    chan *query
	dht          *DHT
}

// newTransactionManager returns new transactionManager pointer.
func newTransactionManager(maxCursor uint64, dht *DHT) *transactionManager {
	return &transactionManager{
		RWMutex:      &sync.RWMutex{},
		transactions: newSyncedMap(),
		index:        newSyncedMap(),
		maxCursor:    maxCursor,
		queryChan:    make(chan *query, 1024),
		dht:          dht,
	}
}

// genTransID generates a transaction id and returns it.
func (tm *transactionManager) genTransID() string {
	tm.Lock()
	defer tm.Unlock()

	tm.cursor = (tm.cursor + 1) % tm.maxCursor
	return string(int2bytes(tm.cursor))
}

// newTransaction creates a new transaction.
func (tm *transactionManager) newTransaction(id string, q *query) *transaction {
	return &transaction{
		id:       id,
		query:    q,
		response: make(chan struct{}, tm.dht.Try+1),
	}
}

// genIndexKey generates an indexed key which consists of queryType and
// address.
func (tm *transactionManager) genIndexKey(queryType, address string) string {
	return strings.Join([]string{queryType, address}, ":")
}

// genIndexKeyByTrans generates an indexed key by a transaction.
func (tm *transactionManager) genIndexKeyByTrans(trans *transaction) string {
	return tm.genIndexKey(trans.request.Method, trans.node.addr.String())
}

// insert adds a transaction to transactionManager.
func (tm *transactionManager) insert(trans *transaction) {
	tm.Lock()
	defer tm.Unlock()

	tm.transactions.Set(trans.id, trans)
	tm.index.Set(tm.genIndexKeyByTrans(trans), trans)
}

// delete removes a transaction from transactionManager.
func (tm *transactionManager) delete(transID string) {
	v, ok := tm.transactions.Get(transID)
	if !ok {
		return
	}

	tm.Lock()
	defer tm.Unlock()

	trans := v.(*transaction)
	tm.transactions.Delete(trans.id)
	tm.index.Delete(tm.genIndexKeyByTrans(trans))
}

// len returns how many transactions are requesting now.
func (tm *transactionManager) len() int {
	return tm.transactions.Len()
}

// transaction returns a transaction. keyType should be one of 0, 1 which
// represents transId and index each.
func (tm *transactionManager) transaction(key string, keyType int) *transaction {

	sm := tm.transactions
	if keyType == 1 {
		sm = tm.index
	}

	v, ok := sm.Get(key)
	if !ok {
		return nil
	}

	return v.(*transaction)
}

// getByTransID returns a transaction by transID.
func (tm *transactionManager) getByTransID(transID string) *transaction {
	return tm.transaction(transID, 0)
}

// getByIndex returns a transaction by indexed key.
func (tm *transactionManager) getByIndex(index string) *transaction {
	return tm.transaction(index, 1)
}

// transaction gets the proper transaction with whose id is transId and
// address is addr.
func (tm *transactionManager) filterOne(transID string, addr *net.UDPAddr) *transaction {
	trans := tm.getByTransID(transID)
	if trans == nil || trans.node.addr.String() != addr.String() {
		return nil
	}
	return trans
}

// query sends the query-formed data to udp and wait for the response.
// When timeout, it will retry `try - 1` times, which means it will query
// `try` times totally.
func (tm *transactionManager) query(q *query, try int) {
	trans := tm.newTransaction(q.request.ID, q)

	tm.insert(trans)
	defer tm.delete(trans.id)

	success := false
	for i := 0; i < try; i++ {
		if err := send(tm.dht, q.node.addr, q.request); err != nil {
			break
		}

		select {
		case <-trans.response:
			success = true
			break
		case <-time.After(time.Second * 15):
		}
	}

	if !success && q.node.id != nil {
		tm.dht.routingTable.RemoveByAddr(q.node.addr.String())
	}
}

// run starts to listen and consume the query chan.
func (tm *transactionManager) run() {
	var q *query

	for {
		select {
		case q = <-tm.queryChan:
			go tm.query(q, tm.dht.Try)
		}
	}
}

// sendQuery send query-formed data to the chan.
func (tm *transactionManager) sendQuery(no *node, request Request) {
	// If the target is self, then stop.
	if no.id != nil && no.id.RawString() == tm.dht.node.id.RawString() ||
		tm.getByIndex(tm.genIndexKey(request.Method, no.addr.String())) != nil {

		return
	}

	request.ID = tm.genTransID()
	request.NodeID = tm.dht.node.id.RawString()
	tm.queryChan <- &query{node: no, request: request}
}

// ping sends ping query to the chan.
func (tm *transactionManager) ping(no *node) {
	tm.sendQuery(no, Request{Method: pingMethod})
}

// findNode sends find_node query to the chan.
func (tm *transactionManager) findNode(no *node, target string) {
	tm.sendQuery(no, Request{Method: findNodeMethod, Args: []string{target}})
}

// handle handles packets received from udp.
func handle(dht *DHT, pkt packet) {
	log.Infof("Received message from %s: %s", pkt.raddr.IP.String(), string(pkt.data))
	if len(dht.workerTokens) == dht.PacketWorkerLimit {
		return
	}

	dht.workerTokens <- struct{}{}

	go func() {
		defer func() {
			<-dht.workerTokens
		}()

		var data map[string]interface{}
		err := bencode.DecodeBytes(pkt.data, &data)
		if err != nil {
			log.Errorf("Error decoding data: %s\n%s", err, pkt.data)
			return
		}

		msgType, ok := data[headerTypeField]
		if !ok {
			log.Errorf("Decoded data has no message type: %s", data)
			return
		}

		switch msgType.(int64) {
		case requestType:
			request := Request{
				ID:     data[headerMessageIDField].(string),
				NodeID: data[headerNodeIDField].(string),
				Method: data[headerPayloadField].(string),
				Args:   getArgs(data[headerArgsField]),
			}
			spew.Dump(request)
			handleRequest(dht, pkt.raddr, request)

		case responseType:
			response := Response{
				ID:     data[headerMessageIDField].(string),
				NodeID: data[headerNodeIDField].(string),
			}

			if reflect.TypeOf(data[headerPayloadField]).Kind() == reflect.String {
				response.Data = data[headerPayloadField].(string)
			} else {
				response.FindNodeData = getFindNodeResponse(data[headerPayloadField])
			}

			spew.Dump(response)

			handleResponse(dht, pkt.raddr, response)

		case errorType:
			e := Error{
				ID:            data[headerMessageIDField].(string),
				NodeID:        data[headerNodeIDField].(string),
				ExceptionType: data[headerPayloadField].(string),
				Response:      getArgs(data[headerArgsField]),
			}
			handleError(dht, pkt.raddr, e)

		default:
			log.Errorf("Invalid message type: %s", msgType)
			return
		}
	}()
}

func getFindNodeResponse(i interface{}) (data []findNodeDatum) {
	if reflect.TypeOf(i).Kind() != reflect.Slice {
		return
	}

	v := reflect.ValueOf(i)
	for i := 0; i < v.Len(); i++ {
		if v.Index(i).Kind() != reflect.Interface {
			continue
		}

		contact := v.Index(i).Elem()
		if contact.Type().Kind() != reflect.Slice || contact.Len() != 3 {
			continue
		}

		if contact.Index(0).Elem().Kind() != reflect.String ||
			contact.Index(1).Elem().Kind() != reflect.String ||
			!(contact.Index(2).Elem().Kind() == reflect.Int64 ||
				contact.Index(2).Elem().Kind() == reflect.Int) {
			continue
		}

		data = append(data, findNodeDatum{
			ID:   contact.Index(0).Elem().String(),
			IP:   contact.Index(1).Elem().String(),
			Port: int(contact.Index(2).Elem().Int()),
		})
	}
	return
}

func getArgs(argsInt interface{}) (args []string) {
	if reflect.TypeOf(argsInt).Kind() == reflect.Slice {
		v := reflect.ValueOf(argsInt)
		for i := 0; i < v.Len(); i++ {
			args = append(args, cast.ToString(v.Index(i).Interface()))
		}
	}
	return
}

// handleRequest handles the requests received from udp.
func handleRequest(dht *DHT, addr *net.UDPAddr, request Request) (success bool) {
	if request.NodeID == dht.node.id.RawString() {
		return
	}

	if len(request.NodeID) != nodeIDLength {
		send(dht, addr, Error{ID: request.ID, NodeID: dht.node.id.RawString(), Response: []string{"Invalid ID"}})
		return
	}

	if no, ok := dht.routingTable.GetNodeByAddress(addr.String()); ok && no.id.RawString() != request.NodeID {
		dht.routingTable.RemoveByAddr(addr.String())
		send(dht, addr, Error{ID: request.ID, NodeID: dht.node.id.RawString(), Response: []string{"Invalid ID"}})
		return
	}

	switch request.Method {
	case pingMethod:
		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id.RawString(), Data: "pong"})
	case findNodeMethod:
		if len(request.Args) < 1 {
			send(dht, addr, Error{ID: request.ID, NodeID: dht.node.id.RawString(), Response: []string{"No target"}})
			return
		}

		target := request.Args[0]
		if len(target) != nodeIDLength {
			send(dht, addr, Error{ID: request.ID, NodeID: dht.node.id.RawString(), Response: []string{"Invalid target"}})
			return
		}

		nodes := []findNodeDatum{}
		targetID := newBitmapFromString(target)

		no, _ := dht.routingTable.GetNodeKBucktByID(targetID)
		if no != nil {
			nodes = []findNodeDatum{{ID: no.id.RawString(), IP: no.addr.IP.String(), Port: no.addr.Port}}
		} else {
			neighbors := dht.routingTable.GetNeighbors(targetID, dht.K)
			for _, n := range neighbors {
				nodes = append(nodes, findNodeDatum{ID: n.id.RawString(), IP: n.addr.IP.String(), Port: n.addr.Port})
			}
		}

		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id.RawString(), FindNodeData: nodes})

	default:
		//		send(dht, addr, makeError(t, protocolError, "invalid q"))
		return
	}

	no, _ := newNode(request.NodeID, addr.Network(), addr.String())
	dht.routingTable.Insert(no)
	return true
}

// findOn puts nodes in the response to the routingTable, then if target is in
// the nodes or all nodes are in the routingTable, it stops. Otherwise it
// continues to findNode or getPeers.
func findOn(dht *DHT, nodes []findNodeDatum, target *bitmap, queryType string) error {
	hasNew, found := false, false
	for _, n := range nodes {
		no, err := newNode(n.ID, dht.Network, fmt.Sprintf("%s:%d", n.IP, n.Port))
		if err != nil {
			return err
		}

		if no.id.RawString() == target.RawString() {
			found = true
		}

		if dht.routingTable.Insert(no) {
			hasNew = true
		}
	}

	if found || !hasNew {
		return nil
	}

	targetID := target.RawString()
	for _, no := range dht.routingTable.GetNeighbors(target, dht.K) {
		switch queryType {
		case findNodeMethod:
			dht.transactionManager.findNode(no, targetID)
		default:
			panic("invalid find type")
		}
	}
	return nil
}

// handleResponse handles responses received from udp.
func handleResponse(dht *DHT, addr *net.UDPAddr, response Response) (success bool) {
	trans := dht.transactionManager.filterOne(response.ID, addr)
	if trans == nil {
		return
	}

	// If response's node id is not the same with the node id in the
	// transaction, raise error.
	// TODO: is this necessary??? why??
	if trans.node.id != nil && trans.node.id.RawString() != response.NodeID {
		dht.routingTable.RemoveByAddr(addr.String())
		return
	}

	node, err := newNode(response.NodeID, addr.Network(), addr.String())
	if err != nil {
		return
	}

	switch trans.request.Method {
	case pingMethod:
	case findNodeMethod:
		target := trans.request.Args[0]
		if findOn(dht, response.FindNodeData, newBitmapFromString(target), findNodeMethod) != nil {
			return
		}
	default:
		return
	}

	// inform transManager to delete transaction.
	trans.response <- struct{}{}

	dht.routingTable.Insert(node)

	return true
}

// handleError handles errors received from udp.
func handleError(dht *DHT, addr *net.UDPAddr, e Error) (success bool) {
	if trans := dht.transactionManager.filterOne(e.ID, addr); trans != nil {
		trans.response <- struct{}{}
	}

	return true
}
