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
const bucketSize = 20
const numBuckets = nodeIDLength * 8

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
		Address: ":4444",
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
	node := &Node{id: id, addr: config.Address}
	return &DHT{
		conf:         config,
		node:         node,
		routingTable: NewRoutingTable(node),
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
func handleRequest(dht *DHT, addr *net.UDPAddr, request Request) (success bool) {
	if request.NodeID == dht.node.id.RawString() {
		log.Warn("ignoring self-request")
		return
	}

	switch request.Method {
	case pingMethod:
		log.Println("ping")
		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id.RawString(), Data: pingSuccessResponse})
	case storeMethod:
		log.Println("store")
		node := &Node{id: newBitmapFromHex(request.StoreArgs.Value.LbryID), addr: request.StoreArgs.Value.Port}
		dht.store.Insert(newBitmapFromHex(request.StoreArgs.BlobHash))
		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id.RawString(), Data: storeSuccessResponse})
	case findNodeMethod:
		log.Println("findnode")
	case findValueMethod:
		log.Println("findvalue")
		//if len(request.Args) < 1 {
		//	send(dht, addr, Error{ID: request.ID, NodeID: dht.node.id.RawString(), Response: []string{"No target"}})
		//	return
		//}
		//
		//target := request.Args[0]
		//if len(target) != nodeIDLength {
		//	send(dht, addr, Error{ID: request.ID, NodeID: dht.node.id.RawString(), Response: []string{"Invalid target"}})
		//	return
		//}
		//
		//nodes := []findNodeDatum{}
		//targetID := newBitmapFromString(target)
		//
		//no, _ := dht.routingTable.GetNodeKBucktByID(targetID)
		//if no != nil {
		//	nodes = []findNodeDatum{{ID: no.id.RawString(), IP: no.addr.IP.String(), Port: no.addr.Port}}
		//} else {
		//	neighbors := dht.routingTable.GetNeighbors(targetID, dht.K)
		//	for _, n := range neighbors {
		//		nodes = append(nodes, findNodeDatum{ID: n.id.RawString(), IP: n.addr.IP.String(), Port: n.addr.Port})
		//	}
		//}
		//
		//send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id.RawString(), FindNodeData: nodes})

	default:
		//		send(dht, addr, makeError(t, protocolError, "invalid q"))
		log.Errorln("invalid request method")
		return
	}

	node := &Node{id: newBitmapFromString(request.NodeID), addr: addr.String()}
	dht.routingTable.Update(node)
	return true
}

// handleResponse handles responses received from udp.
func handleResponse(dht *DHT, addr *net.UDPAddr, response Response) (success bool) {
	spew.Dump(response)

	//switch trans.request.Method {
	//case pingMethod:
	//case findNodeMethod:
	//	target := trans.request.Args[0]
	//	if findOn(dht, response.FindNodeData, newBitmapFromString(target), findNodeMethod) != nil {
	//		return
	//	}
	//default:
	//	return
	//}

	node := &Node{id: newBitmapFromString(response.NodeID), addr: addr.String()}
	dht.routingTable.Update(node)

	return true
}

// handleError handles errors received from udp.
func handleError(dht *DHT, addr *net.UDPAddr, e Error) (success bool) {
	spew.Dump(e)
	return true
}

// send sends data to the udp.
func send(dht *DHT, addr *net.UDPAddr, data Message) error {
	if req, ok := data.(Request); ok {
		log.Debugf("[%s] query %s: sending request: %s(%s)", dht.node.id.Hex()[:8], hex.EncodeToString([]byte(req.ID))[:8], req.Method, argsToString(req.Args))
	} else if res, ok := data.(Response); ok {
		log.Debugf("[%s] query %s: sending response: %s", dht.node.id.Hex()[:8], hex.EncodeToString([]byte(res.ID))[:8], res.Data)
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
	args := []string{}
	if reflect.TypeOf(argsInt).Kind() == reflect.Slice {
		v := reflect.ValueOf(argsInt)
		for i := 0; i < v.Len(); i++ {
			args = append(args, cast.ToString(v.Index(i).Interface()))
		}
	}
	return args
}

func argsToString(args []string) string {
	for k, v := range args {
		if len(v) == nodeIDLength {
			args[k] = hex.EncodeToString([]byte(v))[:8]
		}
	}
	return strings.Join(args, ", ")
}
