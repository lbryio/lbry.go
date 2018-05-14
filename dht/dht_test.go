package dht

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lbryio/lbry.go/crypto"
)

// TODO: make a dht with X nodes, have them all join, then ensure that every node appears at least once in another node's routing table

func TestNodeFinder_FindNodes(t *testing.T) {
	bs, dhts := TestingCreateDHT(3, true, false)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
		bs.Shutdown()
	}()

	nf := newContactFinder(dhts[2].node, RandomBitmapP(), false)
	res, err := nf.Find()
	if err != nil {
		t.Fatal(err)
	}
	foundNodes, found := res.Contacts, res.Found

	if found {
		t.Fatal("something was found, but it should not have been")
	}

	if len(foundNodes) != 3 {
		t.Errorf("expected 3 node, found %d", len(foundNodes))
	}

	foundBootstrap := false
	foundOne := false
	foundTwo := false

	for _, n := range foundNodes {
		if n.id.Equals(bs.id) {
			foundBootstrap = true
		}
		if n.id.Equals(dhts[0].node.id) {
			foundOne = true
		}
		if n.id.Equals(dhts[1].node.id) {
			foundTwo = true
		}
	}

	if !foundBootstrap {
		t.Errorf("did not find bootstrap node %s", bs.id.Hex())
	}
	if !foundOne {
		t.Errorf("did not find first node %s", dhts[0].node.id.Hex())
	}
	if !foundTwo {
		t.Errorf("did not find second node %s", dhts[1].node.id.Hex())
	}
}

func TestNodeFinder_FindNodes_NoBootstrap(t *testing.T) {
	_, dhts := TestingCreateDHT(3, false, false)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
	}()

	nf := newContactFinder(dhts[2].node, RandomBitmapP(), false)
	_, err := nf.Find()
	if err == nil {
		t.Fatal("contact finder should have errored saying that there are no contacts in the routing table")
	}
}

func TestNodeFinder_FindValue(t *testing.T) {
	bs, dhts := TestingCreateDHT(3, true, false)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
		bs.Shutdown()
	}()

	blobHashToFind := RandomBitmapP()
	nodeToFind := Contact{id: RandomBitmapP(), ip: net.IPv4(1, 2, 3, 4), port: 5678}
	dhts[0].node.store.Upsert(blobHashToFind, nodeToFind)

	nf := newContactFinder(dhts[2].node, blobHashToFind, true)
	res, err := nf.Find()
	if err != nil {
		t.Fatal(err)
	}
	foundNodes, found := res.Contacts, res.Found

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

func TestDHT_LargeDHT(t *testing.T) {
	nodes := 100
	bs, dhts := TestingCreateDHT(nodes, true, true)
	defer func() {
		for _, d := range dhts {
			go d.Shutdown()
		}
		bs.Shutdown()
		time.Sleep(1 * time.Second)
	}()

	wg := &sync.WaitGroup{}
	numIDs := nodes / 2
	ids := make([]Bitmap, numIDs)
	for i := 0; i < numIDs; i++ {
		ids[i] = RandomBitmapP()
	}
	for i := 0; i < numIDs; i++ {
		go func(i int) {
			wg.Add(1)
			defer wg.Done()
			dhts[int(crypto.RandInt64(int64(nodes)))].Announce(ids[i])
		}(i)
	}
	wg.Wait()

	dhts[len(dhts)-1].PrintState()
}
