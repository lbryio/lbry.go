package dht

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lbryio/errors.go"
	"github.com/lbryio/lbry.go/stopOnce"
	"github.com/spf13/cast"

	log "github.com/sirupsen/logrus"
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
const udpMaxMessageLength = 1024 // I think our longest message is ~676 bytes, so I rounded up

const tExpire = 86400 * time.Second    // the time after which a key/value pair expires; this is a time-to-live (TTL) from the original publication date
const tRefresh = 3600 * time.Second    // the time after which an otherwise unaccessed bucket must be refreshed
const tReplicate = 3600 * time.Second  // the interval between Kademlia replication events, when a node is required to publish its entire database
const tRepublish = 86400 * time.Second // the time after which the original publisher must republish a key/value pair

const numBuckets = nodeIDLength * 8
const compactNodeInfoLength = nodeIDLength + 6

const tokenSecretRotationInterval = 5 * time.Minute // how often the token-generating secret is rotated

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
		Address: "0.0.0.0:4444",
		SeedNodes: []string{
			"lbrynet1.lbry.io:4444",
			"lbrynet2.lbry.io:4444",
			"lbrynet3.lbry.io:4444",
		},
	}
}

// DHT represents a DHT node.
type DHT struct {
	// config
	conf *Config
	// local contact
	contact Contact
	// node
	node *Node
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

	contact, err := getContact(config.NodeID, config.Address)
	if err != nil {
		return nil, err
	}

	node, err := NewNode(contact.id)
	if err != nil {
		return nil, err
	}

	d := &DHT{
		conf:    config,
		contact: contact,
		node:    node,
		stop:    stopOnce.New(),
		stopWG:  &sync.WaitGroup{},
		joined:  make(chan struct{}),
	}
	return d, nil
}

// join makes current node join the dht network.
func (dht *DHT) join() {
	defer close(dht.joined) // if anyone's waiting for join to finish, they'll know its done

	log.Debugf("[%s] joining network", dht.node.id.HexShort())

	// ping nodes, which gets their real node IDs and adds them to the routing table
	atLeastOneNodeResponded := false
	for _, addr := range dht.conf.SeedNodes {
		err := dht.Ping(addr)
		if err != nil {
			log.Error(errors.Prefix(fmt.Sprintf("[%s] join", dht.node.id.HexShort()), err))
		} else {
			atLeastOneNodeResponded = true
		}
	}

	if !atLeastOneNodeResponded {
		log.Errorf("[%s] join: no nodes responded to initial ping", dht.node.id.HexShort())
		return
	}

	// now call iterativeFind on yourself
	_, err := dht.Get(dht.node.id)
	if err != nil {
		log.Errorf("[%s] join: %s", dht.node.id.HexShort(), err.Error())
	}
}

// Start starts the dht
func (dht *DHT) Start() error {
	listener, err := net.ListenPacket(network, dht.conf.Address)
	if err != nil {
		return errors.Err(err)
	}
	conn := listener.(*net.UDPConn)

	err = dht.node.Connect(conn)
	if err != nil {
		return err
	}

	dht.join()
	log.Debugf("[%s] DHT ready on %s (%d nodes found during join)", dht.node.id.HexShort(), dht.contact.Addr().String(), dht.node.rt.Count())
	return nil
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
	dht.node.Shutdown()
	log.Debugf("[%s] DHT stopped", dht.node.id.HexShort())
}

// Get returns the list of nodes that have the blob for the given hash
func (dht *DHT) Ping(addr string) error {
	raddr, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return err
	}

	tmpNode := Contact{id: RandomBitmapP(), ip: raddr.IP, port: raddr.Port}
	res := dht.node.Send(tmpNode, Request{Method: pingMethod})
	if res == nil {
		return errors.Err("no response from node %s", addr)
	}

	return nil
}

// Get returns the list of nodes that have the blob for the given hash
func (dht *DHT) Get(hash Bitmap) ([]Contact, error) {
	nf := newContactFinder(dht.node, hash, true)
	res, err := nf.Find()
	if err != nil {
		return nil, err
	}

	if res.Found {
		return res.Contacts, nil
	}
	return nil, nil
}

// Announce announces to the DHT that this node has the blob for the given hash
func (dht *DHT) Announce(hash Bitmap) error {
	nf := newContactFinder(dht.node, hash, false)
	res, err := nf.Find()
	if err != nil {
		return err
	}

	// TODO: if this node is closer than farthest peer, store locally and pop farthest peer

	for _, node := range res.Contacts {
		go dht.storeOnNode(hash, node)
	}

	return nil
}

func (dht *DHT) storeOnNode(hash Bitmap, node Contact) {
	dht.stopWG.Add(1)
	defer dht.stopWG.Done()

	resCh := dht.node.SendAsync(context.Background(), node, Request{
		Method: findValueMethod,
		Arg:    &hash,
	})
	var res *Response

	select {
	case res = <-resCh:
	case <-dht.stop.Chan():
		return
	}

	if res == nil {
		return // request timed out
	}

	dht.node.SendAsync(context.Background(), node, Request{
		Method: storeMethod,
		StoreArgs: &storeArgs{
			BlobHash: hash,
			Value: storeArgsValue{
				Token:  res.Token,
				LbryID: dht.contact.id,
				Port:   dht.contact.port,
			},
		},
	})
}

func (dht *DHT) PrintState() {
	log.Printf("DHT node %s at %s", dht.contact.String(), time.Now().Format(time.RFC822Z))
	log.Printf("Outstanding transactions: %d", dht.node.CountActiveTransactions())
	log.Printf("Stored hashes: %d", dht.node.store.CountStoredHashes())
	log.Printf("Buckets:")
	for _, line := range strings.Split(dht.node.rt.BucketInfo(), "\n") {
		log.Println(line)
	}
}

func printNodeList(list []Contact) {
	for i, n := range list {
		log.Printf("%d) %s", i, n.String())
	}
}

func MakeTestDHT(numNodes int) []*DHT {
	if numNodes < 1 {
		return nil
	}

	ip := "127.0.0.1"
	firstPort := 21000
	dhts := make([]*DHT, numNodes)

	for i := 0; i < numNodes; i++ {
		seeds := []string{}
		if i > 0 {
			seeds = []string{ip + ":" + strconv.Itoa(firstPort)}
		}

		dht, err := New(&Config{Address: ip + ":" + strconv.Itoa(firstPort+i), NodeID: RandomBitmapP().Hex(), SeedNodes: seeds})
		if err != nil {
			panic(err)
		}

		go dht.Start()
		dht.WaitUntilJoined()
		dhts[i] = dht
	}

	return dhts
}

func getContact(nodeID, addr string) (Contact, error) {
	var c Contact
	if nodeID == "" {
		c.id = RandomBitmapP()
	} else {
		c.id = BitmapFromHexP(nodeID)
	}

	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		return c, errors.Err(err)
	} else if ip == "" {
		return c, errors.Err("address does not contain an IP")
	} else if port == "" {
		return c, errors.Err("address does not contain a port")
	}

	c.ip = net.ParseIP(ip)
	if c.ip == nil {
		return c, errors.Err("invalid ip")
	}

	c.port, err = cast.ToIntE(port)
	if err != nil {
		return c, errors.Err(err)
	}

	return c, nil
}
