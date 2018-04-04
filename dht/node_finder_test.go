package dht

import (
	"net"
	"testing"
	"time"
)

func TestNodeFinder_FindNodes(t *testing.T) {
	id1 := newRandomBitmap()
	id2 := newRandomBitmap()
	id3 := newRandomBitmap()

	seedIP := "127.0.0.1:21216"

	dht1, err := New(&Config{Address: seedIP, NodeID: id1.Hex()})
	if err != nil {
		t.Fatal(err)
	}
	go dht1.Start()
	defer dht1.Shutdown()

	time.Sleep(1 * time.Second) // give dhts a chance to connect

	dht2, err := New(&Config{Address: "127.0.0.1:21217", NodeID: id2.Hex(), SeedNodes: []string{seedIP}})
	if err != nil {
		t.Fatal(err)
	}
	go dht2.Start()
	defer dht2.Shutdown()

	time.Sleep(1 * time.Second) // give dhts a chance to connect

	dht3, err := New(&Config{Address: "127.0.0.1:21218", NodeID: id3.Hex(), SeedNodes: []string{seedIP}})
	if err != nil {
		t.Fatal(err)
	}
	go dht3.Start()
	defer dht3.Shutdown()

	time.Sleep(1 * time.Second) // give dhts a chance to connect

	nf := newNodeFinder(dht3, newRandomBitmap(), false)
	res, err := nf.Find()
	if err != nil {
		t.Fatal(err)
	}
	foundNodes, found := res.Nodes, res.Found

	if found {
		t.Fatal("something was found, but it should not have been")
	}

	if len(foundNodes) != 2 {
		t.Errorf("expected 2 nodes, found %d", len(foundNodes))
	}

	foundOne := false
	foundTwo := false

	for _, n := range foundNodes {
		if n.id.Equals(id1) {
			foundOne = true
		}
		if n.id.Equals(id2) {
			foundTwo = true
		}
	}

	if !foundOne {
		t.Errorf("did not find node %s", id1.Hex())
	}
	if !foundTwo {
		t.Errorf("did not find node %s", id2.Hex())
	}
}

func TestNodeFinder_FindValue(t *testing.T) {
	id1 := newRandomBitmap()
	id2 := newRandomBitmap()
	id3 := newRandomBitmap()

	seedIP := "127.0.0.1:21216"

	dht1, err := New(&Config{Address: seedIP, NodeID: id1.Hex()})
	if err != nil {
		t.Fatal(err)
	}
	go dht1.Start()
	defer dht1.Shutdown()

	time.Sleep(1 * time.Second)

	dht2, err := New(&Config{Address: "127.0.0.1:21217", NodeID: id2.Hex(), SeedNodes: []string{seedIP}})
	if err != nil {
		t.Fatal(err)
	}
	go dht2.Start()
	defer dht2.Shutdown()

	time.Sleep(1 * time.Second) // give dhts a chance to connect

	dht3, err := New(&Config{Address: "127.0.0.1:21218", NodeID: id3.Hex(), SeedNodes: []string{seedIP}})
	if err != nil {
		t.Fatal(err)
	}
	go dht3.Start()
	defer dht3.Shutdown()

	time.Sleep(1 * time.Second) // give dhts a chance to connect

	blobHashToFind := newRandomBitmap()
	nodeToFind := Node{id: newRandomBitmap(), ip: net.IPv4(1, 2, 3, 4), port: 5678}
	dht1.store.Upsert(blobHashToFind, nodeToFind)

	nf := newNodeFinder(dht3, blobHashToFind, true)
	res, err := nf.Find()
	if err != nil {
		t.Fatal(err)
	}
	foundNodes, found := res.Nodes, res.Found

	if !found {
		t.Fatal("node was not found")
	}

	if len(foundNodes) != 1 {
		t.Fatalf("expected one node, found %d", len(foundNodes))
	}

	if !foundNodes[0].id.Equals(nodeToFind.id) {
		t.Fatalf("found node id %s, expected %s", foundNodes[0].id.Hex(), nodeToFind.id.Hex())
	}
}
