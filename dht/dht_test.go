package dht

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/zeebo/bencode"
)

func TestPing(t *testing.T) {
	log.SetLevel(log.DebugLevel)
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
	blobHashToStore := newRandomBitmap().RawString()

	storeRequest := Request{
		ID:     messageID,
		NodeID: testNodeID.RawString(),
		Method: storeMethod,
		StoreArgs: &storeArgs{
			BlobHash: blobHashToStore,
			NodeID:   testNodeID,
		},
	}
	storeRequest.StoreArgs.Value.Token = "arst"
	storeRequest.StoreArgs.Value.LbryID = testNodeID.RawString()
	storeRequest.StoreArgs.Value.Port = 9999

	_ = "64 " + // start message
		"313A30 693065" + // type: 0
		"313A31 3230   3A 6EB490B5788B63F0F7E6D92352024D0CBDEC2D3A" + // message id
		"313A32 3438   3A 7CE1B831DEC8689E44F80F547D2DEA171F6A625E1A4FF6C6165E645F953103DABEB068A622203F859C6C64658FD3AA3B" + // node id
		"313A33 35     3A 73746F7265" + // method
		"313A34 6C" + // start args list
		"3438 3A 3214D6C2F77FCB5E8D5FC07EDAFBA614F031CE8B2EAB49F924F8143F6DFBADE048D918710072FB98AB1B52B58F4E1468" + // block hash
		"64" + // start value dict
		"363A6C6272796964 3438 3A 7CE1B831DEC8689E44F80F547D2DEA171F6A625E1A4FF6C6165E645F953103DABEB068A622203F859C6C64658FD3AA3B" + // lbry id
		"343A706F7274 69 33333333 65" + // port
		"353A746F6B656E 3438 3A 17C2D8E1E48EF21567FE4AD5C8ED944B798D3B65AB58D0C9122AD6587D1B5FED472EA2CB12284CEFA1C21EFF302322BD" + // token
		"65" + // end value dict
		"3438 3A 7CE1B831DEC8689E44F80F547D2DEA171F6A625E1A4FF6C6165E645F953103DABEB068A622203F859C6C64658FD3AA3B" + // node id
		"693065" + // self store (integer)
		"65" + // end args list
		"65" // end message

	data, err := bencode.EncodeBytes(storeRequest)
	if err != nil {
		t.Error(err)
		return
	}

	conn.toRead <- testUDPPacket{addr: conn.addr, data: data}
	timer := time.NewTimer(3 * time.Second)

	var response map[string]interface{}
	select {
	case <-timer.C:
		t.Error("timeout")
		return
	case resp := <-conn.writes:
		err := bencode.DecodeBytes(resp.data, &response)
		if err != nil {
			t.Error(err)
			return
		}
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

	if len(dht.store.data) != 1 {
		t.Error("dht store has wrong number of items")
	}

	items := dht.store.Get(blobHashToStore)
	if len(items) != 1 {
		t.Error("list created in store, but nothing in list")
	}
	if !items[0].Equals(testNodeID) {
		t.Error("wrong value stored")
	}
}

func TestFindNode(t *testing.T) {
	dhtNodeID := newRandomBitmap()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: ":21216", NodeID: dhtNodeID.Hex()})
	dht.conn = conn
	dht.listen()
	go dht.runHandler()

	data, _ := hex.DecodeString("64313a30693065313a3132303a2afdf2272981651a2c64e39ab7f04ec2d3b5d5d2313a3234383a7ce1b831dec8689e44f80f547d2dea171f6a625e1a4ff6c6165e645f953103dabeb068a622203f859c6c64658fd3aa3b313a33383a66696e644e6f6465313a346c34383a7ce1b831dec8689e44f80f547d2dea171f6a625e1a4ff6c6165e645f953103dabeb068a622203f859c6c64658fd3aa3b6565")

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

		spew.Dump(response)
	}
}

func TestFindValue(t *testing.T) {
	dhtNodeID := newRandomBitmap()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: ":21216", NodeID: dhtNodeID.Hex()})
	dht.conn = conn
	dht.listen()
	go dht.runHandler()

	data, _ := hex.DecodeString("64313a30693065313a3132303a7de8e57d34e316abbb5a8a8da50dcd1ad4c80e0f313a3234383a7ce1b831dec8689e44f80f547d2dea171f6a625e1a4ff6c6165e645f953103dabeb068a622203f859c6c64658fd3aa3b313a33393a66696e6456616c7565313a346c34383aa47624b8e7ee1e54df0c45e2eb858feb0b705bd2a78d8b739be31ba188f4bd6f56b371c51fecc5280d5fd26ba4168e966565")

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

		spew.Dump(response)
	}
}
