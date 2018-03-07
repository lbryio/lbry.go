package dht

import (
	"testing"
	"time"

	"github.com/zeebo/bencode"
)

func TestPing(t *testing.T) {
	dhtNodeID := newRandomBitmap()
	testNodeID := newRandomBitmap()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: ":21216", NodeID: dhtNodeID.Hex()})
	dht.conn = conn
	dht.listen()
	go dht.runHandler()

	messageID := newRandomBitmap().RawString()

	data, err := bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      requestType,
		headerMessageIDField: messageID,
		headerNodeIDField:    testNodeID.RawString(),
		headerPayloadField:   "ping",
		headerArgsField:      []string{},
	})
	if err != nil {
		panic(err)
	}

	conn.toRead <- testUDPPacket{addr: conn.addr, data: data}
	timer := time.NewTimer(3 * time.Second)

	select {
	case <-timer.C:
		t.Error("timeout")
	case resp := <-conn.writes:
		var response map[string]interface{}
		err := bencode.DecodeBytes(resp.data, &response)
		if err != nil {
			t.Error(err)
			return
		}

		if len(response) != 4 {
			t.Errorf("expected 4 response fields, got %d", len(response))
		}

		_, ok := response[headerTypeField]
		if !ok {
			t.Error("missing type field")
		} else {
			rType, ok := response[headerTypeField].(int64)
			if !ok {
				t.Error("type is not an integer")
			} else if rType != responseType {
				t.Error("unexpected response type")
			}
		}

		_, ok = response[headerMessageIDField]
		if !ok {
			t.Error("missing message id field")
		} else {
			rMessageID, ok := response[headerMessageIDField].(string)
			if !ok {
				t.Error("message ID is not a string")
			} else if rMessageID != messageID {
				t.Error("unexpected message ID")
			}
		}

		_, ok = response[headerNodeIDField]
		if !ok {
			t.Error("missing node id field")
		} else {
			rNodeID, ok := response[headerNodeIDField].(string)
			if !ok {
				t.Error("node ID is not a string")
			} else if rNodeID != dhtNodeID.RawString() {
				t.Error("unexpected node ID")
			}
		}

		_, ok = response[headerPayloadField]
		if !ok {
			t.Error("missing payload field")
		} else {
			rNodeID, ok := response[headerPayloadField].(string)
			if !ok {
				t.Error("payload is not a string")
			} else if rNodeID != pingSuccessResponse {
				t.Error("did not pong")
			}
		}
	}
}

func TestStore(t *testing.T) {
	dhtNodeID := newRandomBitmap()
	testNodeID := newRandomBitmap()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: ":21216", NodeID: dhtNodeID.Hex()})
	dht.conn = conn
	dht.listen()
	go dht.runHandler()

	messageID := newRandomBitmap().RawString()
	idToStore := newRandomBitmap().RawString()

	data, err := bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      requestType,
		headerMessageIDField: messageID,
		headerNodeIDField:    testNodeID.RawString(),
		headerPayloadField:   "store",
		headerArgsField:      []string{idToStore},
	})
	if err != nil {
		panic(err)
	}

	conn.toRead <- testUDPPacket{addr: conn.addr, data: data}
	timer := time.NewTimer(3 * time.Second)

	select {
	case <-timer.C:
		t.Error("timeout")
	case resp := <-conn.writes:
		var response map[string]interface{}
		err := bencode.DecodeBytes(resp.data, &response)
		if err != nil {
			t.Error(err)
			return
		}

		if len(response) != 4 {
			t.Errorf("expected 4 response fields, got %d", len(response))
		}

		_, ok := response[headerTypeField]
		if !ok {
			t.Error("missing type field")
		} else {
			rType, ok := response[headerTypeField].(int64)
			if !ok {
				t.Error("type is not an integer")
			} else if rType != responseType {
				t.Error("unexpected response type")
			}
		}

		_, ok = response[headerMessageIDField]
		if !ok {
			t.Error("missing message id field")
		} else {
			rMessageID, ok := response[headerMessageIDField].(string)
			if !ok {
				t.Error("message ID is not a string")
			} else if rMessageID != messageID {
				t.Error("unexpected message ID")
			}
		}

		_, ok = response[headerNodeIDField]
		if !ok {
			t.Error("missing node id field")
		} else {
			rNodeID, ok := response[headerNodeIDField].(string)
			if !ok {
				t.Error("node ID is not a string")
			} else if rNodeID != dhtNodeID.RawString() {
				t.Error("unexpected node ID")
			}
		}

		_, ok = response[headerPayloadField]
		if !ok {
			t.Error("missing payload field")
		} else {
			rNodeID, ok := response[headerPayloadField].(string)
			if !ok {
				t.Error("payload is not a string")
			} else if rNodeID != storeSuccessResponse {
				t.Error("did not return OK")
			}
		}
	}
}
