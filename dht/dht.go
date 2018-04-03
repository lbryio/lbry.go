package dht

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/lbryio/errors.go"
	"github.com/lbryio/lbry.go/stopOnce"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

func init() {
	//log.SetFormatter(&log.TextFormatter{ForceColors: true})
	//log.SetLevel(log.DebugLevel)
}

const network = "udp4"

const alpha = 3            // this is the constant alpha in the spec
const nodeIDLength = 48    // bytes. this is the constant B in the spec
const messageIDLength = 20 // bytes.
const bucketSize = 8       // this is the constant k in the spec

const udpRetry = 3
const udpTimeout = 10 * time.Second

const tExpire = 86400 * time.Second    // the time after which a key/value pair expires; this is a time-to-live (TTL) from the original publication date
const tRefresh = 3600 * time.Second    // the time after which an otherwise unaccessed bucket must be refreshed
const tReplicate = 3600 * time.Second  // the interval between Kademlia replication events, when a node is required to publish its entire database
const tRepublish = 86400 * time.Second // the time after which the original publisher must republish a key/value pair

const numBuckets = nodeIDLength * 8
const compactNodeInfoLength = nodeIDLength + 6

// packet represents the information receive from udp.
type packet struct {
	data  []byte
	raddr *net.UDPAddr
}

// Config represents the configure of dht.
type Config struct {
	// this node's address. format is `ip:port`
	Address string
	// the seed nodes through which we can join in dht network
	SeedNodes []string
	// the hex-encoded node id for this node. if string is empty, a random id will be generated
	NodeID string
	// print the state of the dht every minute
	PrintState bool
}

// NewStandardConfig returns a Config pointer with default values.
func NewStandardConfig() *Config {
	return &Config{
		Address: "127.0.0.1:4444",
		SeedNodes: []string{
			"lbrynet1.lbry.io:4444",
			"lbrynet2.lbry.io:4444",
			"lbrynet3.lbry.io:4444",
		},
	}
}

// UDPConn allows using a mocked connection for testing sending/receiving data
type UDPConn interface {
	ReadFromUDP([]byte) (int, *net.UDPAddr, error)
	WriteToUDP([]byte, *net.UDPAddr) (int, error)
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	Close() error
}

// DHT represents a DHT node.
type DHT struct {
	conf    *Config
	conn    UDPConn
	node    *Node
	rt      *RoutingTable
	packets chan packet
	store   *peerStore
	tm      *transactionManager
	stop    *stopOnce.Stopper
	stopWG  *sync.WaitGroup
}

// New returns a DHT pointer. If config is nil, then config will be set to the default config.
func New(config *Config) (*DHT, error) {
	if config == nil {
		config = NewStandardConfig()
	}

	var id bitmap
	if config.NodeID == "" {
		id = newRandomBitmap()
	} else {
		id = newBitmapFromHex(config.NodeID)
	}

	ip, port, err := net.SplitHostPort(config.Address)
	if err != nil {
		return nil, errors.Err(err)
	} else if ip == "" {
		return nil, errors.Err("address does not contain an IP")
	} else if port == "" {
		return nil, errors.Err("address does not contain a port")
	}

	portInt, err := cast.ToIntE(port)
	if err != nil {
		return nil, errors.Err(err)
	}

	node := &Node{id: id, ip: net.ParseIP(ip), port: portInt}
	if node.ip == nil {
		return nil, errors.Err("invalid ip")
	}

	d := &DHT{
		conf:    config,
		node:    node,
		rt:      newRoutingTable(node),
		packets: make(chan packet),
		store:   newPeerStore(),
		stop:    stopOnce.New(),
		stopWG:  &sync.WaitGroup{},
	}
	d.tm = newTransactionManager(d)
	return d, nil
}

// init initializes global variables.
func (dht *DHT) init() error {
	log.Debugf("Initializing DHT on %s (node id %s)", dht.conf.Address, dht.node.id.HexShort())

	listener, err := net.ListenPacket(network, dht.conf.Address)
	if err != nil {
		return errors.Err(err)
	}

	dht.conn = listener.(*net.UDPConn)

	if dht.conf.PrintState {
		go printState(dht)
	}

	return nil
}

// listen receives message from udp.
func (dht *DHT) listen() {
	dht.stopWG.Add(1)
	defer dht.stopWG.Done()

	buf := make([]byte, 8192)

	for {
		select {
		case <-dht.stop.Chan():
			return
		default:
		}

		dht.conn.SetReadDeadline(time.Now().Add(1 * time.Second)) // need this to periodically check shutdown chan
		n, raddr, err := dht.conn.ReadFromUDP(buf)
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

		dht.packets <- packet{data: data, raddr: raddr}
	}
}

// join makes current node join the dht network.
func (dht *DHT) join() {
	log.Debugf("[%s] joining network", dht.node.id.HexShort())
	// get real node IDs and add them to the routing table
	for _, addr := range dht.conf.SeedNodes {
		raddr, err := net.ResolveUDPAddr(network, addr)
		if err != nil {
			log.Errorln(err)
			continue
		}

		tmpNode := Node{id: newRandomBitmap(), ip: raddr.IP, port: raddr.Port}
		res := dht.tm.Send(tmpNode, &Request{Method: pingMethod})
		if res == nil {
			log.Errorf("[%s] join: no response from seed node %s", dht.node.id.HexShort(), addr)
		}
	}

	// now call iterativeFind on yourself
	_, err := dht.FindNodes(dht.node.id)
	if err != nil {
		log.Errorf("[%s] join: %s", dht.node.id.HexShort(), err.Error())
	}
}

func (dht *DHT) runHandler() {
	dht.stopWG.Add(1)
	defer dht.stopWG.Done()

	var pkt packet

	for {
		select {
		case pkt = <-dht.packets:
			handlePacket(dht, pkt)
		case <-dht.stop.Chan():
			return
		}
	}
}

// Start starts the dht
func (dht *DHT) Start() {
	err := dht.init()
	if err != nil {
		log.Error(err)
		return
	}

	go dht.listen()
	go dht.runHandler()

	dht.join()
	log.Infof("[%s] DHT ready", dht.node.id.HexShort())
}

// Shutdown shuts down the dht
func (dht *DHT) Shutdown() {
	log.Debugf("[%s] DHT shutting down", dht.node.id.HexShort())
	dht.stop.Stop()
	dht.stopWG.Wait()
	dht.conn.Close()
	log.Infof("[%s] DHT stopped", dht.node.id.HexShort())
}

func printState(dht *DHT) {
	t := time.NewTicker(60 * time.Second)
	for {
		log.Printf("DHT state at %s", time.Now().Format(time.RFC822Z))
		log.Printf("Outstanding transactions: %d", dht.tm.Count())
		log.Printf("Known nodes: %d", dht.store.CountKnownNodes())
		log.Printf("Buckets: \n%s", dht.rt.BucketInfo())
		<-t.C
	}
}

func (dht *DHT) FindNodes(hash bitmap) ([]Node, error) {
	nf := newNodeFinder(dht, hash, false)
	res, err := nf.Find()
	if err != nil {
		return nil, err
	}
	return res.Nodes, nil
}

func (dht *DHT) FindValue(hash bitmap) ([]Node, bool, error) {
	nf := newNodeFinder(dht, hash, true)
	res, err := nf.Find()
	if err != nil {
		return nil, false, err
	}
	return res.Nodes, res.Found, nil
}

type nodeFinder struct {
	findValue bool // true if we're using findValue
	target    bitmap
	dht       *DHT

	done *stopOnce.Stopper

	findValueMutex  *sync.Mutex
	findValueResult []Node

	activeNodesMutex *sync.Mutex
	activeNodes      []Node

	shortlistContactedMutex *sync.Mutex
	shortlist               []Node
	contacted               map[bitmap]bool
}

type findNodeResponse struct {
	Found bool
	Nodes []Node
}

func newNodeFinder(dht *DHT, target bitmap, findValue bool) *nodeFinder {
	return &nodeFinder{
		dht:                     dht,
		target:                  target,
		findValue:               findValue,
		findValueMutex:          &sync.Mutex{},
		activeNodesMutex:        &sync.Mutex{},
		shortlistContactedMutex: &sync.Mutex{},
		contacted:               make(map[bitmap]bool),
		done:                    stopOnce.New(),
	}
}

func (nf *nodeFinder) Find() (findNodeResponse, error) {
	log.Debugf("[%s] starting an iterative Find() for %s (findValue is %t)", nf.dht.node.id.HexShort(), nf.target.HexShort(), nf.findValue)
	nf.appendNewToShortlist(nf.dht.rt.GetClosest(nf.target, alpha))
	if len(nf.shortlist) == 0 {
		return findNodeResponse{}, errors.Err("no nodes in routing table")
	}

	wg := &sync.WaitGroup{}

	for i := 0; i < alpha; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nf.iterationWorker(i + 1)
		}(i)
	}

	wg.Wait()

	// TODO: what to do if we have less than K active nodes, shortlist is empty, but we
	// TODO: have other nodes in our routing table whom we have not contacted. prolly contact them?

	result := findNodeResponse{}
	if nf.findValue && len(nf.findValueResult) > 0 {
		result.Found = true
		result.Nodes = nf.findValueResult
	} else {
		result.Nodes = nf.activeNodes
		if len(result.Nodes) > bucketSize {
			result.Nodes = result.Nodes[:bucketSize]
		}
	}

	return result, nil
}

func (nf *nodeFinder) iterationWorker(num int) {
	log.Debugf("[%s] starting worker %d", nf.dht.node.id.HexShort(), num)
	defer func() { log.Debugf("[%s] stopping worker %d", nf.dht.node.id.HexShort(), num) }()

	for {
		maybeNode := nf.popFromShortlist()
		if maybeNode == nil {
			// TODO: block if there are pending requests out from other workers. there may be more shortlist values coming
			log.Debugf("[%s] no more nodes in shortlist", nf.dht.node.id.HexShort())
			return
		}
		node := *maybeNode

		if node.id.Equals(nf.dht.node.id) {
			continue // cannot contact self
		}

		req := &Request{Args: []string{nf.target.RawString()}}
		if nf.findValue {
			req.Method = findValueMethod
		} else {
			req.Method = findNodeMethod
		}

		log.Debugf("[%s] contacting %s", nf.dht.node.id.HexShort(), node.id.HexShort())

		var res *Response
		ctx, cancel := context.WithCancel(context.Background())
		resCh := nf.dht.tm.SendAsync(ctx, node, req)
		select {
		case res = <-resCh:
		case <-nf.done.Chan():
			log.Debugf("[%s] worker %d: canceled", nf.dht.node.id.HexShort(), num)
			cancel()
			return
		}

		if res == nil {
			// nothing to do, response timed out
		} else if nf.findValue && res.FindValueKey != "" {
			log.Debugf("[%s] worker %d: got value", nf.dht.node.id.HexShort(), num)
			nf.findValueMutex.Lock()
			nf.findValueResult = res.FindNodeData
			nf.findValueMutex.Unlock()
			nf.done.Stop()
			return
		} else {
			log.Debugf("[%s] worker %d: got more contacts", nf.dht.node.id.HexShort(), num)
			nf.insertIntoActiveList(node)
			nf.appendNewToShortlist(res.FindNodeData)
		}

		if nf.isSearchFinished() {
			log.Debugf("[%s] worker %d: search is finished", nf.dht.node.id.HexShort(), num)
			nf.done.Stop()
			return
		}
	}
}

func (nf *nodeFinder) appendNewToShortlist(nodes []Node) {
	nf.shortlistContactedMutex.Lock()
	defer nf.shortlistContactedMutex.Unlock()

	notContacted := []Node{}
	for _, n := range nodes {
		if _, ok := nf.contacted[n.id]; !ok {
			notContacted = append(notContacted, n)
		}
	}

	nf.shortlist = append(nf.shortlist, notContacted...)
	sortNodesInPlace(nf.shortlist, nf.target)
}

func (nf *nodeFinder) popFromShortlist() *Node {
	nf.shortlistContactedMutex.Lock()
	defer nf.shortlistContactedMutex.Unlock()

	if len(nf.shortlist) == 0 {
		return nil
	}

	first := nf.shortlist[0]
	nf.shortlist = nf.shortlist[1:]
	nf.contacted[first.id] = true
	return &first
}

func (nf *nodeFinder) insertIntoActiveList(node Node) {
	nf.activeNodesMutex.Lock()
	defer nf.activeNodesMutex.Unlock()

	inserted := false
	for i, n := range nf.activeNodes {
		if node.id.Xor(nf.target).Less(n.id.Xor(nf.target)) {
			nf.activeNodes = append(nf.activeNodes[:i], append([]Node{node}, nf.activeNodes[i:]...)...)
			inserted = true
		}
	}
	if !inserted {
		nf.activeNodes = append(nf.activeNodes, node)
	}
}

func (nf *nodeFinder) isSearchFinished() bool {
	if nf.findValue && len(nf.findValueResult) > 0 {
		return true
	}

	select {
	case <-nf.done.Chan():
		return true
	default:
	}

	nf.shortlistContactedMutex.Lock()
	defer nf.shortlistContactedMutex.Unlock()

	if len(nf.shortlist) == 0 {
		return true
	}

	nf.activeNodesMutex.Lock()
	defer nf.activeNodesMutex.Unlock()

	if len(nf.activeNodes) >= bucketSize && nf.activeNodes[bucketSize-1].id.Xor(nf.target).Less(nf.shortlist[0].id.Xor(nf.target)) {
		// we have at least K active nodes, and we don't have any closer nodes yet to contact
		return true
	}

	return false
}
