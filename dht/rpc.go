package dht

import (
	"encoding/hex"
	"net"
	"time"

	"github.com/lbryio/errors.go"
	"github.com/lbryio/lbry.go/util"

	"github.com/davecgh/go-spew/spew"
	"github.com/lyoshenka/bencode"
	log "github.com/sirupsen/logrus"
)

// handlePacket handles packets received from udp.
func handlePacket(dht *DHT, pkt packet) {
	//log.Debugf("[%s] Received message from %s:%s (%d bytes) %s", dht.node.id.HexShort(), pkt.raddr.IP.String(), strconv.Itoa(pkt.raddr.Port), len(pkt.data), hex.EncodeToString(pkt.data))

	if !util.InSlice(string(pkt.data[0:5]), []string{"d1:0i", "di0ei"}) {
		log.Errorf("[%s] data is not a well-formatted dict: (%d bytes) %s", dht.node.id.HexShort(), len(pkt.data), hex.EncodeToString(pkt.data))
		return
	}

	// TODO: test this stuff more thoroughly

	// the following is a bit of a hack, but it lets us avoid decoding every message twice
	// it depends on the data being a dict with 0 as the first key (so it starts with "d1:0i") and the message type as the first value

	switch pkt.data[5] {
	case '0' + requestType:
		request := Request{}
		err := bencode.DecodeBytes(pkt.data, &request)
		if err != nil {
			log.Errorf("[%s] error decoding request: %s: (%d bytes) %s", dht.node.id.HexShort(), err.Error(), len(pkt.data), hex.EncodeToString(pkt.data))
			return
		}
		log.Debugf("[%s] query %s: received request from %s: %s(%s)", dht.node.id.HexShort(), request.ID.HexShort(), request.NodeID.HexShort(), request.Method, request.ArgsDebug())
		handleRequest(dht, pkt.raddr, request)

	case '0' + responseType:
		response := Response{}
		err := bencode.DecodeBytes(pkt.data, &response)
		if err != nil {
			log.Errorf("[%s] error decoding response: %s: (%d bytes) %s", dht.node.id.HexShort(), err.Error(), len(pkt.data), hex.EncodeToString(pkt.data))
			return
		}
		log.Debugf("[%s] query %s: received response from %s: %s", dht.node.id.HexShort(), response.ID.HexShort(), response.NodeID.HexShort(), response.ArgsDebug())
		handleResponse(dht, pkt.raddr, response)

	case '0' + errorType:
		e := Error{}
		err := bencode.DecodeBytes(pkt.data, &e)
		if err != nil {
			log.Errorf("[%s] error decoding error: %s: (%d bytes) %s", dht.node.id.HexShort(), err.Error(), len(pkt.data), hex.EncodeToString(pkt.data))
			return
		}
		log.Debugf("[%s] query %s: received error from %s: %s", dht.node.id.HexShort(), e.ID.HexShort(), e.NodeID.HexShort(), e.ExceptionType)
		handleError(dht, pkt.raddr, e)

	default:
		log.Errorf("[%s] invalid message type: %s", dht.node.id.HexShort(), pkt.data[5])
		return
	}
}

// handleRequest handles the requests received from udp.
func handleRequest(dht *DHT, addr *net.UDPAddr, request Request) {
	if request.NodeID.Equals(dht.node.id) {
		log.Warn("ignoring self-request")
		return
	}

	switch request.Method {
	case pingMethod:
		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id, Data: pingSuccessResponse})
	case storeMethod:
		// TODO: we should be sending the IP in the request, not just using the sender's IP
		// TODO: should we be using StoreArgs.NodeID or StoreArgs.Value.LbryID ???
		dht.store.Upsert(request.StoreArgs.BlobHash, Node{id: request.StoreArgs.NodeID, ip: addr.IP, port: request.StoreArgs.Value.Port})
		send(dht, addr, Response{ID: request.ID, NodeID: dht.node.id, Data: storeSuccessResponse})
	case findNodeMethod:
		if request.Arg == nil {
			log.Errorln("request is missing arg")
			return
		}
		doFindNodes(dht, addr, request)
	case findValueMethod:
		if request.Arg == nil {
			log.Errorln("request is missing arg")
			return
		}

		if nodes := dht.store.Get(*request.Arg); len(nodes) > 0 {
			send(dht, addr, Response{
				ID:           request.ID,
				NodeID:       dht.node.id,
				FindValueKey: request.Arg.RawString(),
				FindNodeData: nodes,
			})
		} else {
			doFindNodes(dht, addr, request)
		}

	default:
		//		send(dht, addr, makeError(t, protocolError, "invalid q"))
		log.Errorln("invalid request method")
		return
	}

	node := Node{id: request.NodeID, ip: addr.IP, port: addr.Port}
	dht.rt.Update(node)
}

func doFindNodes(dht *DHT, addr *net.UDPAddr, request Request) {
	closestNodes := dht.rt.GetClosest(*request.Arg, bucketSize)
	if len(closestNodes) > 0 {
		response := Response{ID: request.ID, NodeID: dht.node.id, FindNodeData: make([]Node, len(closestNodes))}
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
		tx.res <- response
	}

	node := Node{id: response.NodeID, ip: addr.IP, port: addr.Port}
	dht.rt.Update(node)
}

// handleError handles errors received from udp.
func handleError(dht *DHT, addr *net.UDPAddr, e Error) {
	spew.Dump(e)
	node := Node{id: e.NodeID, ip: addr.IP, port: addr.Port}
	dht.rt.Update(node)
}

// send sends data to a udp address
func send(dht *DHT, addr *net.UDPAddr, data Message) error {
	encoded, err := bencode.EncodeBytes(data)
	if err != nil {
		return errors.Err(err)
	}

	if req, ok := data.(Request); ok {
		log.Debugf("[%s] query %s: sending request to %s (%d bytes) %s(%s)",
			dht.node.id.HexShort(), req.ID.HexShort(), addr.String(), len(encoded), req.Method, req.ArgsDebug())
	} else if res, ok := data.(Response); ok {
		log.Debugf("[%s] query %s: sending response to %s (%d bytes) %s",
			dht.node.id.HexShort(), res.ID.HexShort(), addr.String(), len(encoded), res.ArgsDebug())
	} else {
		log.Debugf("[%s] (%d bytes) %s", dht.node.id.HexShort(), len(encoded), spew.Sdump(data))
	}

	dht.conn.SetWriteDeadline(time.Now().Add(time.Second * 15))

	_, err = dht.conn.WriteToUDP(encoded, addr)
	return errors.Err(err)
}
