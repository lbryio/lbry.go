package dht

import (
	"net"
	"time"

	"github.com/lbryio/errors.go"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

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
	SetWriteDeadline(time.Time) error
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
	}
	d.tm = newTransactionManager(d)
	return d, nil
}

// init initializes global variables.
func (dht *DHT) init() error {
	log.Info("Initializing DHT on " + dht.conf.Address)
	log.Infof("Node ID is %s", dht.node.id.Hex())

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
	go func() {
		buf := make([]byte, 8192)
		for {
			n, raddr, err := dht.conn.ReadFromUDP(buf)
			if err != nil {
				log.Errorf("udp read error: %v", err)
				continue
			} else if raddr == nil {
				log.Errorf("udp read with no raddr")
				continue
			}
			dht.packets <- packet{data: buf[:n], raddr: raddr}
		}
	}()
}

// join makes current node join the dht network.
func (dht *DHT) join() {
	for _, addr := range dht.conf.SeedNodes {
		raddr, err := net.ResolveUDPAddr(network, addr)
		if err != nil {
			continue
		}

		_ = raddr

		// NOTE: Temporary node has NO node id.
		//dht.transactionManager.findNode(
		//	&node{addr: raddr},
		//	dht.node.id.RawString(),
		//)
	}
}

func (dht *DHT) runHandler() {
	var pkt packet

	for {
		select {
		case pkt = <-dht.packets:
			handlePacket(dht, pkt)
		}
	}
}

// Run starts the dht.
func (dht *DHT) Run() error {
	err := dht.init()
	if err != nil {
		return err
	}

	dht.listen()
	dht.join()
	log.Info("DHT ready")
	dht.runHandler()
	return nil
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

//func (dht *DHT) Get(hash bitmap) ([]Node, error) {
//	return iterativeFindNode(dht, hash)
//}
//
//func iterativeFindNode(dht *DHT, hash bitmap) ([]Node, error) {
//	shortlist := dht.rt.FindClosest(hash, alpha)
//	if len(shortlist) == 0 {
//		return nil, errors.Err("no nodes in routing table")
//	}
//
//	pending := make(chan *Node)
//	contacted := make(map[bitmap]bool)
//	contactedMutex := &sync.Mutex{}
//	closestNodeMutex := &sync.Mutex{}
//	closestNode := shortlist[0]
//	wg := sync.WaitGroup{}
//
//	for i := 0; i < alpha; i++ {
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			for {
//				node, ok := <-pending
//				if !ok {
//					return
//				}
//
//				contactedMutex.Lock()
//				if _, ok := contacted[node.id]; ok {
//					contactedMutex.Unlock()
//					continue
//				}
//				contacted[node.id] = true
//				contactedMutex.Unlock()
//
//				res := dht.tm.Send(node, &Request{
//					NodeID: dht.node.id.RawString(),
//					Method: findNodeMethod,
//					Args:   []string{hash.RawString()},
//				})
//				if res == nil {
//					// remove node from shortlist
//					continue
//				}
//
//				for _, n := range res.FindNodeData {
//					pending <- &n
//					closestNodeMutex.Lock()
//					if n.id.Xor(hash).Less(closestNode.id.Xor(hash)) {
//						closestNode = &n
//					}
//					closestNodeMutex.Unlock()
//				}
//			}
//		}()
//	}
//
//	for _, n := range shortlist {
//		pending <- n
//	}
//
//	wg.Wait()
//
//	return nil, nil
//}
