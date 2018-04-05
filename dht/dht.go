package dht

import (
	"context"
	"net"
	"strings"
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
	// print the state of the dht every X time
	PrintState time.Duration
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
	// config
	conf *Config
	// UDP connection for sending and receiving data
	conn UDPConn
	// the local dht node
	node *Node
	// routing table
	rt *routingTable
	// channel of incoming packets
	packets chan packet
	// data store
	store *peerStore
	// transaction manager
	tm *transactionManager
	// stopper to shut down DHT
	stop *stopOnce.Stopper
	// wait group for all the things that need to be stopped when DHT shuts down
	stopWG *sync.WaitGroup
	// channel is closed when DHT joins network
	joined chan struct{}
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
		joined:  make(chan struct{}),
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

	if dht.conf.PrintState > 0 {
		go func() {
			t := time.NewTicker(dht.conf.PrintState)
			for {
				dht.PrintState()
				<-t.C
			}
		}()
	}

	return nil
}

// listen receives message from udp.
func (dht *DHT) listen() {
	dht.stopWG.Add(1)
	defer dht.stopWG.Done()

	buf := make([]byte, 16384)

	for {
		select {
		case <-dht.stop.Chan():
			return
		default:
		}

		dht.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)) // need this to periodically check shutdown chan
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

	close(dht.joined) // if anyone's waiting for join to finish, they'll know its done
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
	log.Debugf("[%s] DHT ready on %s", dht.node.id.HexShort(), dht.node.Addr().String())
}

func (dht *DHT) WaitUntilJoined() {
	if dht.joined == nil {
		panic("dht not initialized")
	}
	<-dht.joined
}

// Shutdown shuts down the dht
func (dht *DHT) Shutdown() {
	log.Debugf("[%s] DHT shutting down", dht.node.id.HexShort())
	dht.stop.Stop()
	dht.stopWG.Wait()
	dht.conn.Close()
	log.Debugf("[%s] DHT stopped", dht.node.id.HexShort())
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
		dht.tm.SendAsync(context.Background(), node, Request{
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

func (dht *DHT) PrintState() {
	log.Printf("DHT state at %s", time.Now().Format(time.RFC822Z))
	log.Printf("Outstanding transactions: %d", dht.tm.Count())
	log.Printf("Stored hashes: %d", dht.store.CountStoredHashes())
	log.Printf("Buckets:")
	for _, line := range strings.Split(dht.rt.BucketInfo(), "\n") {
		log.Println(line)
	}
}

func printNodeList(list []Node) {
	for i, n := range list {
		log.Printf("%d) %s %s:%d", i, n.id.HexShort(), n.ip.String(), n.port)
	}
}
