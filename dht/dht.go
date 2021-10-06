package dht

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/lbryio/lbry.go/v3/dht/bits"
	"github.com/lbryio/lbry.go/v3/extras/stop"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

var log *logrus.Logger

func UseLogger(l *logrus.Logger) {
	log = l
}

func init() {
	log = logrus.StandardLogger()
	//log.SetFormatter(&log.TextFormatter{ForceColors: true})
	//log.SetLevel(log.DebugLevel)
}

// DHT represents a DHT node.
type DHT struct {
	// config
	conf *Config
	// local contact
	contact Contact
	// node
	node *Node
	// stopGroup to shut down DHT
	grp *stop.Group
	// channel is closed when DHT joins network
	joined chan struct{}
	// cache for store tokens
	tokenCache *tokenCache
	// hashes that need to be put into the announce queue or removed from the queue
	announceAddRemove chan queueEdit
}

// New returns a DHT pointer. If config is nil, then config will be set to the default config.
func New(config *Config) *DHT {
	if config == nil {
		config = NewStandardConfig()
	}

	d := &DHT{
		conf:              config,
		grp:               stop.New(),
		joined:            make(chan struct{}),
		announceAddRemove: make(chan queueEdit),
	}
	return d
}

func (dht *DHT) connect(conn UDPConn) error {
	contact, err := getContact(dht.conf.NodeID, dht.conf.Address)
	if err != nil {
		return err
	}

	dht.contact = contact
	dht.node = NewNode(contact.ID)
	dht.tokenCache = newTokenCache(dht.node, tokenSecretRotationInterval)

	return dht.node.Connect(conn)
}

// Start starts the dht
func (dht *DHT) Start() error {
	listener, err := net.ListenPacket(Network, dht.conf.Address)
	if err != nil {
		return errors.WithStack(err)
	}
	conn := listener.(*net.UDPConn)

	err = dht.connect(conn)
	if err != nil {
		return err
	}

	dht.join()
	log.Infof("[%s] DHT ready on %s (%d nodes found during join)",
		dht.node.id.HexShort(), dht.contact.Addr().String(), dht.node.rt.Count())

	dht.grp.Add(1)
	go func() {
		dht.runAnnouncer()
		dht.grp.Done()
	}()

	if dht.conf.RPCPort > 0 {
		dht.grp.Add(1)
		go func() {
			dht.runRPCServer(dht.conf.RPCPort)
			dht.grp.Done()
		}()
	}

	return nil
}

// join makes current node join the dht network.
func (dht *DHT) join() {
	defer close(dht.joined) // if anyone's waiting for join to finish, they'll know its done

	log.Infof("[%s] joining DHT network", dht.node.id.HexShort())

	// ping nodes, which gets their real node IDs and adds them to the routing table
	atLeastOneNodeResponded := false
	for _, addr := range dht.conf.SeedNodes {
		err := dht.Ping(addr)
		if err != nil {
			log.Error(errors.WithMessage(err, fmt.Sprintf("[%s] join", dht.node.id.HexShort())))
		} else {
			atLeastOneNodeResponded = true
		}
	}

	if !atLeastOneNodeResponded {
		log.Errorf("[%s] join: no nodes responded to initial ping", dht.node.id.HexShort())
		return
	}

	// now call iterativeFind on yourself
	_, _, err := FindContacts(dht.node, dht.node.id, false, dht.grp.Child())
	if err != nil {
		log.Errorf("[%s] join: %s", dht.node.id.HexShort(), err.Error())
	}

	// TODO: after joining, refresh all buckets further away than our closest neighbor
	// http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#join
}

// WaitUntilJoined blocks until the node joins the network.
func (dht *DHT) WaitUntilJoined() {
	if dht.joined == nil {
		panic("dht not initialized")
	}
	<-dht.joined
}

// Shutdown shuts down the dht
func (dht *DHT) Shutdown() {
	log.Debugf("[%s] DHT shutting down", dht.node.id.HexShort())
	dht.grp.StopAndWait()
	dht.node.Shutdown()
	log.Debugf("[%s] DHT stopped", dht.node.id.HexShort())
}

// Ping pings a given address, creates a temporary contact for sending a message, and returns an error if communication
// fails.
func (dht *DHT) Ping(addr string) error {
	raddr, err := net.ResolveUDPAddr(Network, addr)
	if err != nil {
		return err
	}

	tmpNode := Contact{ID: bits.Rand(), IP: raddr.IP, Port: raddr.Port}
	res := dht.node.Send(tmpNode, Request{Method: pingMethod}, SendOptions{skipIDCheck: true})
	if res == nil {
		return errors.WithStack(errors.Newf("no response from node %s", addr))
	}

	return nil
}

// Get returns the list of nodes that have the blob for the given hash
func (dht *DHT) Get(hash bits.Bitmap) ([]Contact, error) {
	contacts, found, err := FindContacts(dht.node, hash, true, dht.grp.Child())
	if err != nil {
		return nil, err
	}

	if found {
		return contacts, nil
	}
	return nil, nil
}

// PrintState prints the current state of the DHT including address, nr outstanding transactions, stored hashes as well
// as current bucket information.
func (dht *DHT) PrintState() {
	log.Printf("DHT node %s at %s", dht.contact.String(), time.Now().Format(time.RFC822Z))
	log.Printf("Outstanding transactions: %d", dht.node.CountActiveTransactions())
	log.Printf("Stored hashes: %d", dht.node.store.CountStoredHashes())
	log.Printf("Buckets:")
	for _, line := range strings.Split(dht.node.rt.BucketInfo(), "\n") {
		log.Println(line)
	}
}

func (dht DHT) ID() bits.Bitmap {
	return dht.contact.ID
}

func getContact(nodeID, addr string) (Contact, error) {
	var c Contact
	if nodeID == "" {
		c.ID = bits.Rand()
	} else {
		c.ID = bits.FromHexP(nodeID)
	}

	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		return c, errors.WithStack(err)
	} else if ip == "" {
		return c, errors.WithStack(errors.New("address does not contain an IP"))
	} else if port == "" {
		return c, errors.WithStack(errors.New("address does not contain a port"))
	}

	c.IP = net.ParseIP(ip)
	if c.IP == nil {
		return c, errors.WithStack(errors.New("invalid ip"))
	}

	c.Port, err = cast.ToIntE(port)
	if err != nil {
		return c, errors.WithStack(err)
	}

	return c, nil
}
