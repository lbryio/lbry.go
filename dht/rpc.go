package dht

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/lyoshenka/bencode"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

func newMessageID() string {
	buf := make([]byte, messageIDLength)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return string(buf)
}

// handlePacket handles packets received from udp.
func handlePacket(dht *DHT, pkt packet) {
	//log.Debugf("[%s] Received message from %s:%s (%d bytes) %s", dht.node.id.HexShort(), pkt.raddr.IP.String(), strconv.Itoa(pkt.raddr.Port), len(pkt.data), hex.EncodeToString(pkt.data))

	var data map[string]interface{}
	err := bencode.DecodeBytes(pkt.data, &data)
	if err != nil {
		log.Errorf("[%s] error decoding data: %s: (%d bytes) %s", dht.node.id.HexShort(), err.Error(), len(pkt.data), hex.EncodeToString(pkt.data))
		return
	}

	msgType, ok := data[headerTypeField]
	if !ok {
		log.Errorf("[%s] decoded data has no message type: %s", dht.node.id.HexShort(), spew.Sdump(data))
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
		log.Debugf("[%s] query %s: received request from %s: %s(%s)", dht.node.id.HexShort(), hex.EncodeToString([]byte(request.ID))[:8], hex.EncodeToString([]byte(request.NodeID))[:8], request.Method, argsToString(request.Args))
		handleRequest(dht, pkt.raddr, request)

	case responseType:
		response := Response{}
		err = bencode.DecodeBytes(pkt.data, &response)
		if err != nil {
			log.Errorln(err)
			return
		}
		log.Debugf("[%s] query %s: received response from %s: %s", dht.node.id.HexShort(), hex.EncodeToString([]byte(response.ID))[:8], hex.EncodeToString([]byte(response.NodeID))[:8], response.ArgsDebug())
		handleResponse(dht, pkt.raddr, response)

	case errorType:
		e := Error{
			ID:            data[headerMessageIDField].(string),
			NodeID:        data[headerNodeIDField].(string),
			ExceptionType: data[headerPayloadField].(string),
			Response:      getArgs(data[headerArgsField]),
		}
		log.Debugf("[%s] query %s: received error from %s: %s", dht.node.id.HexShort(), hex.EncodeToString([]byte(e.ID))[:8], hex.EncodeToString([]byte(e.NodeID))[:8], e.ExceptionType)
		handleError(dht, pkt.raddr, e)

	default:
		log.Errorf("[%s] invalid message type: %s", dht.node.id.HexShort(), msgType)
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
		dht.store.Upsert(request.StoreArgs.BlobHash, Node{id: request.StoreArgs.NodeID, ip: addr.IP, port: request.StoreArgs.Value.Port})
		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id.RawString(), Data: storeSuccessResponse})
	case findNodeMethod:
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

	node := Node{id: newBitmapFromString(request.NodeID), ip: addr.IP, port: addr.Port}
	dht.rt.Update(node)
}

func doFindNodes(dht *DHT, addr *net.UDPAddr, request Request) {
	nodeID := newBitmapFromString(request.Args[0])
	closestNodes := dht.rt.GetClosest(nodeID, bucketSize)
	if len(closestNodes) > 0 {
		response := Response{ID: request.ID, NodeID: dht.node.id.RawString(), FindNodeData: make([]Node, len(closestNodes))}
		for i, n := range closestNodes {
			response.FindNodeData[i] = n
		}
		send(dht, addr, response)
	} else {
		log.Warn("no nodes in routing table")
	}
}

// handleResponse handles responses received from udp.
func handleResponse(dht *DHT, addr *net.UDPAddr, response Response) {
	tx := dht.tm.Find(response.ID, addr)
	if tx != nil {
		tx.res <- &response
	}

	node := Node{id: newBitmapFromString(response.NodeID), ip: addr.IP, port: addr.Port}
	dht.rt.Update(node)
}

// handleError handles errors received from udp.
func handleError(dht *DHT, addr *net.UDPAddr, e Error) {
	spew.Dump(e)
	node := Node{id: newBitmapFromString(e.NodeID), ip: addr.IP, port: addr.Port}
	dht.rt.Update(node)
}

// send sends data to a udp address
func send(dht *DHT, addr *net.UDPAddr, data Message) error {
	encoded, err := bencode.EncodeBytes(data)
	if err != nil {
		return err
	}

	if req, ok := data.(Request); ok {
		log.Debugf("[%s] query %s: sending request to %s (%d bytes) %s(%s)",
			dht.node.id.HexShort(), hex.EncodeToString([]byte(req.ID))[:8], addr.String(), len(encoded), req.Method, argsToString(req.Args))
	} else if res, ok := data.(Response); ok {
		log.Debugf("[%s] query %s: sending response to %s (%d bytes) %s",
			dht.node.id.HexShort(), hex.EncodeToString([]byte(res.ID))[:8], addr.String(), len(encoded), res.ArgsDebug())
	} else {
		log.Debugf("[%s] (%d bytes) %s", dht.node.id.HexShort(), len(encoded), spew.Sdump(data))
	}

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
