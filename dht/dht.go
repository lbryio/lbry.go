package dht

import (
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
		res := dht.tm.Send(tmpNode, Request{Method: pingMethod})
		if res == nil {
			log.Errorf("[%s] join: no response from seed node %s", dht.node.id.HexShort(), addr)
		}
	}

	// now call iterativeFind on yourself
	_, err := dht.Get(dht.node.id)
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
	log.Infof("[%s] DHT ready on %s", dht.node.id.HexShort(), dht.node.Addr().String())
}

// Shutdown shuts down the dht
func (dht *DHT) Shutdown() {
	log.Debugf("[%s] DHT shutting down", dht.node.id.HexShort())
	dht.stop.Stop()
	dht.stopWG.Wait()
	dht.conn.Close()
	log.Infof("[%s] DHT stopped", dht.node.id.HexShort())
}

// Get returns the list of nodes that have the blob for the given hash
func (dht *DHT) Get(hash bitmap) ([]Node, error) {
	nf := newNodeFinder(dht, hash, true)
	res, err := nf.Find()
	if err != nil {
		return nil, err
	}

	if res.Found {
		return res.Nodes, nil
	}
	return nil, nil
}

// Announce announces to the DHT that this node has the blob for the given hash
func (dht *DHT) Announce(hash bitmap) error {
	nf := newNodeFinder(dht, hash, false)
	res, err := nf.Find()
	if err != nil {
		return err
	}

	for _, node := range res.Nodes {
		send(dht, node.Addr(), Request{
			Method: storeMethod,
			StoreArgs: &storeArgs{
				BlobHash: hash,
				Value: storeArgsValue{
					Token:  "",
					LbryID: dht.node.id,
					Port:   dht.node.port,
				},
			},
		})
	}

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
