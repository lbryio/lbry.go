package dht

import (
	"net"
	"testing"
	"time"

	"github.com/lbryio/lbry.go/v3/dht/bits"

	"github.com/lyoshenka/bencode"
)

func TestPing(t *testing.T) {
	dhtNodeID := bits.Rand()
	testNodeID := bits.Rand()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: "127.0.0.1:21216", NodeID: dhtNodeID.Hex()})

	err := dht.connect(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer dht.Shutdown()

	messageID := newMessageID()

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
			t.Fatal(err)
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
			} else if rMessageID != string(messageID[:]) {
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
	dhtNodeID := bits.Rand()
	testNodeID := bits.Rand()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: "127.0.0.1:21216", NodeID: dhtNodeID.Hex()})

	err := dht.connect(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer dht.Shutdown()

	messageID := newMessageID()
	blobHashToStore := bits.Rand()

	storeRequest := Request{
		ID:     messageID,
		NodeID: testNodeID,
		Method: storeMethod,
		StoreArgs: &storeArgs{
			BlobHash: blobHashToStore,
			Value: storeArgsValue{
				Token:  dht.node.tokens.Get(testNodeID, conn.addr),
				LbryID: testNodeID,
				Port:   9999,
			},
			NodeID: testNodeID,
		},
	}

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
		t.Fatal(err)
	}

	conn.toRead <- testUDPPacket{addr: conn.addr, data: data}
	timer := time.NewTimer(3 * time.Second)

	var response map[string]interface{}
	select {
	case <-timer.C:
		t.Fatal("timeout")
	case resp := <-conn.writes:
		err := bencode.DecodeBytes(resp.data, &response)
		if err != nil {
			t.Fatal(err)
		}
	}

	verifyResponse(t, response, messageID, dhtNodeID.RawString())

	_, ok := response[headerPayloadField]
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

	if dht.node.store.CountStoredHashes() != 1 {
		t.Error("dht store has wrong number of items")
	}

	items := dht.node.store.Get(blobHashToStore)
	if len(items) != 1 {
		t.Error("list created in store, but nothing in list")
	}
	if !items[0].ID.Equals(testNodeID) {
		t.Error("wrong value stored")
	}
}

func TestFindNode(t *testing.T) {
	dhtNodeID := bits.Rand()
	testNodeID := bits.Rand()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: "127.0.0.1:21216", NodeID: dhtNodeID.Hex()})

	err := dht.connect(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer dht.Shutdown()

	nodesToInsert := 3
	var nodes []Contact
	for i := 0; i < nodesToInsert; i++ {
		n := Contact{ID: bits.Rand(), IP: net.ParseIP("127.0.0.1"), Port: 10000 + i}
		nodes = append(nodes, n)
		dht.node.rt.Update(n)
	}

	messageID := newMessageID()
	blobHashToFind := bits.Rand()

	request := Request{
		ID:     messageID,
		NodeID: testNodeID,
		Method: findNodeMethod,
		Arg:    &blobHashToFind,
	}

	data, err := bencode.EncodeBytes(request)
	if err != nil {
		t.Fatal(err)
	}

	conn.toRead <- testUDPPacket{addr: conn.addr, data: data}
	timer := time.NewTimer(3 * time.Second)

	var response map[string]interface{}
	select {
	case <-timer.C:
		t.Fatal("timeout")
	case resp := <-conn.writes:
		err := bencode.DecodeBytes(resp.data, &response)
		if err != nil {
			t.Fatal(err)
		}
	}

	verifyResponse(t, response, messageID, dhtNodeID.RawString())

	_, ok := response[headerPayloadField]
	if !ok {
		t.Fatal("missing payload field")
	}

	contacts, ok := response[headerPayloadField].([]interface{})
	if !ok {
		t.Fatal("payload is not a list")
	}

	verifyContacts(t, contacts, nodes)
}

func TestFindValueExisting(t *testing.T) {
	dhtNodeID := bits.Rand()
	testNodeID := bits.Rand()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: "127.0.0.1:21216", NodeID: dhtNodeID.Hex()})

	err := dht.connect(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer dht.Shutdown()

	nodesToInsert := 3
	for i := 0; i < nodesToInsert; i++ {
		n := Contact{ID: bits.Rand(), IP: net.ParseIP("127.0.0.1"), Port: 10000 + i}
		dht.node.rt.Update(n)
	}

	//data, _ := hex.DecodeString("64313a30693065313a3132303a7de8e57d34e316abbb5a8a8da50dcd1ad4c80e0f313a3234383a7ce1b831dec8689e44f80f547d2dea171f6a625e1a4ff6c6165e645f953103dabeb068a622203f859c6c64658fd3aa3b313a33393a66696e6456616c7565313a346c34383aa47624b8e7ee1e54df0c45e2eb858feb0b705bd2a78d8b739be31ba188f4bd6f56b371c51fecc5280d5fd26ba4168e966565")

	messageID := newMessageID()
	valueToFind := bits.Rand()

	nodeToFind := Contact{ID: bits.Rand(), IP: net.ParseIP("1.2.3.4"), PeerPort: 1286}
	dht.node.store.Upsert(valueToFind, nodeToFind)
	dht.node.store.Upsert(valueToFind, nodeToFind)
	dht.node.store.Upsert(valueToFind, nodeToFind)

	request := Request{
		ID:     messageID,
		NodeID: testNodeID,
		Method: findValueMethod,
		Arg:    &valueToFind,
	}

	data, err := bencode.EncodeBytes(request)
	if err != nil {
		t.Fatal(err)
	}

	conn.toRead <- testUDPPacket{addr: conn.addr, data: data}
	timer := time.NewTimer(3 * time.Second)

	var response map[string]interface{}
	select {
	case <-timer.C:
		t.Fatal("timeout")
	case resp := <-conn.writes:
		err := bencode.DecodeBytes(resp.data, &response)
		if err != nil {
			t.Fatal(err)
		}
	}

	verifyResponse(t, response, messageID, dhtNodeID.RawString())

	_, ok := response[headerPayloadField]
	if !ok {
		t.Fatal("missing payload field")
	}

	payload, ok := response[headerPayloadField].(map[string]interface{})
	if !ok {
		t.Fatal("payload is not a dictionary")
	}

	compactContacts, ok := payload[valueToFind.RawString()]
	if !ok {
		t.Fatal("payload is missing key for search value")
	}

	contacts, ok := compactContacts.([]interface{})
	if !ok {
		t.Fatal("search results are not a list")
	}

	verifyCompactContacts(t, contacts, []Contact{nodeToFind})
}

func TestFindValueFallbackToFindNode(t *testing.T) {
	dhtNodeID := bits.Rand()
	testNodeID := bits.Rand()

	conn := newTestUDPConn("127.0.0.1:21217")

	dht := New(&Config{Address: "127.0.0.1:21216", NodeID: dhtNodeID.Hex()})

	err := dht.connect(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer dht.Shutdown()

	nodesToInsert := 3
	var nodes []Contact
	for i := 0; i < nodesToInsert; i++ {
		n := Contact{ID: bits.Rand(), IP: net.ParseIP("127.0.0.1"), Port: 10000 + i}
		nodes = append(nodes, n)
		dht.node.rt.Update(n)
	}

	messageID := newMessageID()
	valueToFind := bits.Rand()

	request := Request{
		ID:     messageID,
		NodeID: testNodeID,
		Method: findValueMethod,
		Arg:    &valueToFind,
	}

	data, err := bencode.EncodeBytes(request)
	if err != nil {
		t.Fatal(err)
	}

	conn.toRead <- testUDPPacket{addr: conn.addr, data: data}
	timer := time.NewTimer(3 * time.Second)

	var response map[string]interface{}
	select {
	case <-timer.C:
		t.Fatal("timeout")
	case resp := <-conn.writes:
		err := bencode.DecodeBytes(resp.data, &response)
		if err != nil {
			t.Fatal(err)
		}
	}

	verifyResponse(t, response, messageID, dhtNodeID.RawString())

	_, ok := response[headerPayloadField]
	if !ok {
		t.Fatal("missing payload field")
	}

	payload, ok := response[headerPayloadField].(map[string]interface{})
	if !ok {
		t.Fatal("payload is not a dictionary")
	}

	contactsList, ok := payload[contactsField]
	if !ok {
		t.Fatal("payload is missing 'contacts' key")
	}

	contacts, ok := contactsList.([]interface{})
	if !ok {
		t.Fatal("'contacts' is not a list")
	}

	verifyContacts(t, contacts, nodes)
}
