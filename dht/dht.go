package dht

import (
	"encoding/hex"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/zeebo/bencode"
)

const network = "udp4"

const alpha = 3         // this is the constant alpha in the spec
const nodeIDLength = 48 // bytes. this is the constant B in the spec
const bucketSize = 8    // this is the constant k in the spec

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

// DHT represents a DHT node.
type DHT struct {
	conf         *Config
	conn         UDPConn
	node         *Node
	routingTable *RoutingTable
	packets      chan packet
	store        *peerStore
}

// New returns a DHT pointer. If config is nil, then config will be set to the default config.
func New(config *Config) *DHT {
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
		panic(err)
	} else if ip == "" {
		panic("address does not contain an IP")
	} else if port == "" {
		panic("address does not contain a port")
	}

	portInt, err := cast.ToIntE(port)
	if err != nil {
		panic(err)
	}

	node := &Node{id: id, ip: net.ParseIP(ip), port: portInt}
	if node.ip == nil {
		panic("invalid ip")
	}
	return &DHT{
		conf:         config,
		node:         node,
		routingTable: newRoutingTable(node),
		packets:      make(chan packet),
		store:        newPeerStore(),
	}
}

// init initializes global variables.
func (dht *DHT) init() {
	log.Info("Initializing DHT on " + dht.conf.Address)
	log.Infof("Node ID is %s", dht.node.id.Hex())
	listener, err := net.ListenPacket(network, dht.conf.Address)
	if err != nil {
		panic(err)
	}

	dht.conn = listener.(*net.UDPConn)
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
			handle(dht, pkt)
		}
	}
}

// Run starts the dht.
func (dht *DHT) Run() {
	dht.init()
	dht.listen()
	dht.join()
	log.Info("DHT ready")
	dht.runHandler()
}

// handle handles packets received from udp.
func handle(dht *DHT, pkt packet) {
	//log.Infof("Received message from %s:%s : %s\n", pkt.raddr.IP.String(), strconv.Itoa(pkt.raddr.Port), hex.EncodeToString(pkt.data))

	var data map[string]interface{}
	err := bencode.DecodeBytes(pkt.data, &data)
	if err != nil {
		log.Errorf("error decoding data: %s\n%s", err, pkt.data)
		return
	}

	msgType, ok := data[headerTypeField]
	if !ok {
		log.Errorf("decoded data has no message type: %s", data)
		return
	}

	switch msgType.(int64) {
	case requestType:
		request := Request{}
		err = bencode.DecodeBytes(pkt.data, &request)
		if err != nil {
			log.Errorln(err)
			return
		}
		log.Debugf("[%s] query %s: received request from %s: %s(%s)", dht.node.id.Hex()[:8], hex.EncodeToString([]byte(request.ID))[:8], hex.EncodeToString([]byte(request.NodeID))[:8], request.Method, argsToString(request.Args))
		handleRequest(dht, pkt.raddr, request)

	case responseType:
		response := Response{}
		err = bencode.DecodeBytes(pkt.data, &response)
		if err != nil {
			return
		}
		log.Debugf("[%s] query %s: received response from %s: %s", dht.node.id.Hex()[:8], hex.EncodeToString([]byte(response.ID))[:8], hex.EncodeToString([]byte(response.NodeID))[:8], response.Data)
		handleResponse(dht, pkt.raddr, response)

	case errorType:
		e := Error{
			ID:            data[headerMessageIDField].(string),
			NodeID:        data[headerNodeIDField].(string),
			ExceptionType: data[headerPayloadField].(string),
			Response:      getArgs(data[headerArgsField]),
		}
		log.Debugf("[%s] query %s: received error from %s: %s", dht.node.id.Hex()[:8], hex.EncodeToString([]byte(e.ID))[:8], hex.EncodeToString([]byte(e.NodeID))[:8], e.ExceptionType)
		handleError(dht, pkt.raddr, e)

	default:
		log.Errorf("Invalid message type: %s", msgType)
		return
	}
}

// handleRequest handles the requests received from udp.
func handleRequest(dht *DHT, addr *net.UDPAddr, request Request) {
	if request.NodeID == dht.node.id.RawString() {
		log.Warn("ignoring self-request")
		return
	}

	switch request.Method {
	case pingMethod:
		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id.RawString(), Data: pingSuccessResponse})
	case storeMethod:
		if request.StoreArgs.BlobHash == "" {
			log.Errorln("blobhash is empty")
			return // nothing to store
		}
		// TODO: we should be sending the IP in the request, not just using the sender's IP
		// TODO: should we be using StoreArgs.NodeID or StoreArgs.Value.LbryID ???
		dht.store.Insert(request.StoreArgs.BlobHash, Node{id: request.StoreArgs.NodeID, ip: addr.IP, port: request.StoreArgs.Value.Port})
		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id.RawString(), Data: storeSuccessResponse})
	case findNodeMethod:
		log.Println("findnode")
		if len(request.Args) < 1 {
			log.Errorln("nothing to find")
			return
		}
		if len(request.Args[0]) != nodeIDLength {
			log.Errorln("invalid node id")
			return
		}
		doFindNodes(dht, addr, request)
	case findValueMethod:
		log.Println("findvalue")
		if len(request.Args) < 1 {
			log.Errorln("nothing to find")
			return
		}
		if len(request.Args[0]) != nodeIDLength {
			log.Errorln("invalid node id")
			return
		}

		if nodes := dht.store.Get(request.Args[0]); len(nodes) > 0 {
			response := Response{ID: request.ID, NodeID: dht.node.id.RawString()}
			response.FindValueKey = request.Args[0]
			response.FindNodeData = nodes
			send(dht, addr, response)
		} else {
			doFindNodes(dht, addr, request)
		}

	default:
		//		send(dht, addr, makeError(t, protocolError, "invalid q"))
		log.Errorln("invalid request method")
		return
	}

	node := &Node{id: newBitmapFromString(request.NodeID), ip: addr.IP, port: addr.Port}
	dht.routingTable.Update(node)
}

func doFindNodes(dht *DHT, addr *net.UDPAddr, request Request) {
	nodeID := newBitmapFromString(request.Args[0])
	closestNodes := dht.routingTable.FindClosest(nodeID, bucketSize)
	if len(closestNodes) > 0 {
		response := Response{ID: request.ID, NodeID: dht.node.id.RawString(), FindNodeData: make([]Node, len(closestNodes))}
		for i, n := range closestNodes {
			response.FindNodeData[i] = *n
		}
		send(dht, addr, response)
	}
}

// handleResponse handles responses received from udp.
func handleResponse(dht *DHT, addr *net.UDPAddr, response Response) {
	spew.Dump(response)

	// TODO: find transaction by message id, pass along response

	node := &Node{id: newBitmapFromString(response.NodeID), ip: addr.IP, port: addr.Port}
	dht.routingTable.Update(node)
}

// handleError handles errors received from udp.
func handleError(dht *DHT, addr *net.UDPAddr, e Error) {
	spew.Dump(e)
	node := &Node{id: newBitmapFromString(e.NodeID), ip: addr.IP, port: addr.Port}
	dht.routingTable.Update(node)
}

// send sends data to the udp.
func send(dht *DHT, addr *net.UDPAddr, data Message) error {
	if req, ok := data.(Request); ok {
		log.Debugf("[%s] query %s: sending request: %s(%s)", dht.node.id.Hex()[:8], hex.EncodeToString([]byte(req.ID))[:8], req.Method, argsToString(req.Args))
	} else if res, ok := data.(Response); ok {
		log.Debugf("[%s] query %s: sending response: %s", dht.node.id.Hex()[:8], hex.EncodeToString([]byte(res.ID))[:8], spew.Sdump(res.Data))
	} else {
		log.Debugf("[%s] %s", spew.Sdump(data))
	}
	encoded, err := bencode.EncodeBytes(data)
	if err != nil {
		return err
	}
	//log.Infof("Encoded: %s", string(encoded))

	dht.conn.SetWriteDeadline(time.Now().Add(time.Second * 15))

	_, err = dht.conn.WriteToUDP(encoded, addr)
	return err
}

func getArgs(argsInt interface{}) []string {
	var args []string
	if reflect.TypeOf(argsInt).Kind() == reflect.Slice {
		v := reflect.ValueOf(argsInt)
		for i := 0; i < v.Len(); i++ {
			args = append(args, cast.ToString(v.Index(i).Interface()))
		}
	}
	return args
}

func argsToString(args []string) string {
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)
	for k, v := range argsCopy {
		if len(v) == nodeIDLength {
			argsCopy[k] = hex.EncodeToString([]byte(v))[:8]
		}
	}
	return strings.Join(argsCopy, ", ")
}
