package dht

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/reflector.go/dht/bits"
)

var testingDHTIP = "127.0.0.1"
var testingDHTFirstPort = 21000

// TestingCreateDHT initializes a testable DHT network with a specific number of nodes, with bootstrap and concurrent options.
func TestingCreateDHT(t *testing.T, numNodes int, bootstrap, concurrent bool) (*BootstrapNode, []*DHT) {
	var bootstrapNode *BootstrapNode
	var seeds []string

	if bootstrap {
		bootstrapAddress := testingDHTIP + ":" + strconv.Itoa(testingDHTFirstPort)
		seeds = []string{bootstrapAddress}
		bootstrapNode = NewBootstrapNode(bits.Rand(), 0, bootstrapDefaultRefreshDuration)
		listener, err := net.ListenPacket(network, bootstrapAddress)
		if err != nil {
			panic(err)
		}
		if err := bootstrapNode.Connect(listener.(*net.UDPConn)); err != nil {
			t.Error("error connecting bootstrap node - ", err)
		}
	}

	if numNodes < 1 {
		return bootstrapNode, nil
	}

	firstPort := testingDHTFirstPort + 1
	dhts := make([]*DHT, numNodes)

	for i := 0; i < numNodes; i++ {
		dht, err := New(&Config{Address: testingDHTIP + ":" + strconv.Itoa(firstPort+i), NodeID: bits.Rand().Hex(), SeedNodes: seeds})
		if err != nil {
			panic(err)
		}

		go func() {
			if err := dht.Start(); err != nil {
				t.Error("error starting dht - ", err)
			}
		}()
		if !concurrent {
			dht.WaitUntilJoined()
		}
		dhts[i] = dht
	}

	if concurrent {
		for _, d := range dhts {
			d.WaitUntilJoined()
		}
	}

	return bootstrapNode, dhts
}

type timeoutErr struct {
	error
}

func (t timeoutErr) Timeout() bool {
	return true
}

func (t timeoutErr) Temporary() bool {
	return true
}

// TODO: just use a normal net.Conn instead of this mock conn

type testUDPPacket struct {
	data []byte
	addr *net.UDPAddr
}

type testUDPConn struct {
	addr   *net.UDPAddr
	toRead chan testUDPPacket
	writes chan testUDPPacket

	readDeadline time.Time
}

func newTestUDPConn(addr string) *testUDPConn {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		panic("addr needs ip and port")
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}
	return &testUDPConn{
		addr:   &net.UDPAddr{IP: net.IP(parts[0]), Port: port},
		toRead: make(chan testUDPPacket),
		writes: make(chan testUDPPacket),
	}
}

func (t testUDPConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	var timeoutCh <-chan time.Time
	if !t.readDeadline.IsZero() {
		timeoutCh = time.After(time.Until(t.readDeadline))
	}

	select {
	case packet, ok := <-t.toRead:
		if !ok {
			return 0, nil, errors.Err("conn closed")
		}
		n := copy(b, packet.data)
		return n, packet.addr, nil
	case <-timeoutCh:
		return 0, nil, timeoutErr{errors.Err("timeout")}
	}
}

func (t testUDPConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	t.writes <- testUDPPacket{data: b, addr: addr}
	return len(b), nil
}

func (t *testUDPConn) SetReadDeadline(tm time.Time) error {
	t.readDeadline = tm
	return nil
}

func (t *testUDPConn) SetWriteDeadline(tm time.Time) error {
	return nil
}

func (t *testUDPConn) Close() error {
	close(t.toRead)
	t.writes = nil
	return nil
}

func verifyResponse(t *testing.T, resp map[string]interface{}, id messageID, dhtNodeID string) {
	if len(resp) != 4 {
		t.Errorf("expected 4 response fields, got %d", len(resp))
	}

	_, ok := resp[headerTypeField]
	if !ok {
		t.Error("missing type field")
	} else {
		rType, ok := resp[headerTypeField].(int64)
		if !ok {
			t.Error("type is not an integer")
		} else if rType != responseType {
			t.Error("unexpected response type")
		}
	}

	_, ok = resp[headerMessageIDField]
	if !ok {
		t.Error("missing message id field")
	} else {
		rMessageID, ok := resp[headerMessageIDField].(string)
		if !ok {
			t.Error("message ID is not a string")
		} else if rMessageID != string(id[:]) {
			t.Error("unexpected message ID")
		}
		if len(rMessageID) != messageIDLength {
			t.Errorf("message ID should be %d chars long", messageIDLength)
		}
	}

	_, ok = resp[headerNodeIDField]
	if !ok {
		t.Error("missing node id field")
	} else {
		rNodeID, ok := resp[headerNodeIDField].(string)
		if !ok {
			t.Error("node ID is not a string")
		} else if rNodeID != dhtNodeID {
			t.Error("unexpected node ID")
		}
		if len(rNodeID) != nodeIDLength {
			t.Errorf("node ID should be %d chars long", nodeIDLength)
		}
	}
}

func verifyContacts(t *testing.T, contacts []interface{}, nodes []Contact) {
	if len(contacts) != len(nodes) {
		t.Errorf("got %d contacts; expected %d", len(contacts), len(nodes))
		return
	}

	foundNodes := make(map[string]bool)

	for _, c := range contacts {
		contact, ok := c.([]interface{})
		if !ok {
			t.Error("contact is not a list")
			return
		}

		if len(contact) != 3 {
			t.Error("contact must be 3 items")
			return
		}

		var currNode Contact
		currNodeFound := false

		id, ok := contact[0].(string)
		if !ok {
			t.Error("contact id is not a string")
		} else {
			if _, ok := foundNodes[id]; ok {
				t.Errorf("contact %s appears multiple times", id)
				continue
			}
			for _, n := range nodes {
				if n.ID.String() == id {
					currNode = n
					currNodeFound = true
					foundNodes[id] = true
					break
				}
			}
			if !currNodeFound {
				t.Errorf("unexpected contact %s", id)
				continue
			}
		}

		ip, ok := contact[1].(string)
		if !ok {
			t.Error("contact IP is not a string")
		} else if !currNode.IP.Equal(net.ParseIP(ip)) {
			t.Errorf("contact IP mismatch. got %s; expected %s", ip, currNode.IP.String())
		}

		port, ok := contact[2].(int64)
		if !ok {
			t.Error("contact port is not an int")
		} else if int(port) != currNode.Port {
			t.Errorf("contact port mismatch. got %d; expected %d", port, currNode.Port)
		}
	}
}

func verifyCompactContacts(t *testing.T, contacts []interface{}, nodes []Contact) {
	if len(contacts) != len(nodes) {
		t.Errorf("got %d contacts; expected %d", len(contacts), len(nodes))
		return
	}

	foundNodes := make(map[string]bool)

	for _, c := range contacts {
		compact, ok := c.(string)
		if !ok {
			t.Error("contact is not a string")
			return
		}

		contact := Contact{}
		err := contact.UnmarshalCompact([]byte(compact))
		if err != nil {
			t.Error(err)
			return
		}

		var currNode Contact
		currNodeFound := false

		if _, ok := foundNodes[contact.ID.Hex()]; ok {
			t.Errorf("contact %s appears multiple times", contact.ID.Hex())
			continue
		}
		for _, n := range nodes {
			if n.ID.Equals(contact.ID) {
				currNode = n
				currNodeFound = true
				foundNodes[contact.ID.Hex()] = true
				break
			}
		}
		if !currNodeFound {
			t.Errorf("unexpected contact %s", contact.ID.Hex())
			continue
		}

		if !currNode.IP.Equal(contact.IP) {
			t.Errorf("contact IP mismatch. got %s; expected %s", contact.IP.String(), currNode.IP.String())
		}

		if contact.Port != currNode.Port {
			t.Errorf("contact port mismatch. got %d; expected %d", contact.Port, currNode.Port)
		}
	}
}
