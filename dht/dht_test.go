package dht

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lbryio/lbry.go/v2/dht/bits"
)

func TestNodeFinder_FindNodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow nodeFinder test")
	}

	bs, dhts := TestingCreateNetwork(t, 3, true, false)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
		bs.Shutdown()
	}()

	contacts, found, err := FindContacts(dhts[2].node, bits.Rand(), false, nil)
	if err != nil {
		t.Fatal(err)
	}

	if found {
		t.Fatal("something was found, but it should not have been")
	}

	if len(contacts) != 3 {
		t.Errorf("expected 3 node, found %d", len(contacts))
	}

	foundBootstrap := false
	foundOne := false
	foundTwo := false

	for _, n := range contacts {
		if n.ID.Equals(bs.id) {
			foundBootstrap = true
		}
		if n.ID.Equals(dhts[0].node.id) {
			foundOne = true
		}
		if n.ID.Equals(dhts[1].node.id) {
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
	_, dhts := TestingCreateNetwork(t, 3, false, false)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
	}()

	_, _, err := FindContacts(dhts[2].node, bits.Rand(), false, nil)
	if err == nil {
		t.Fatal("contact finder should have errored saying that there are no contacts in the routing table")
	}
}

func TestNodeFinder_FindValue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow nodeFinder test")
	}

	bs, dhts := TestingCreateNetwork(t, 3, true, false)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
		bs.Shutdown()
	}()

	blobHashToFind := bits.Rand()
	nodeToFind := Contact{ID: bits.Rand(), IP: net.IPv4(1, 2, 3, 4), Port: 5678}
	dhts[0].node.store.Upsert(blobHashToFind, nodeToFind)

	contacts, found, err := FindContacts(dhts[2].node, blobHashToFind, true, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !found {
		t.Fatal("node was not found")
	}

	if len(contacts) != 1 {
		t.Fatalf("expected one node, found %d", len(contacts))
	}

	if !contacts[0].ID.Equals(nodeToFind.ID) {
		t.Fatalf("found node id %s, expected %s", contacts[0].ID.Hex(), nodeToFind.ID.Hex())
	}
}

func TestDHT_LargeDHT(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large DHT test")
	}

	nodes := 100
	bs, dhts := TestingCreateNetwork(t, nodes, true, true)
	defer func() {
		for _, d := range dhts {
			go d.Shutdown()
		}
		bs.Shutdown()
		time.Sleep(1 * time.Second)
	}()

	wg := &sync.WaitGroup{}
	ids := make([]bits.Bitmap, nodes)
	for i := range ids {
		ids[i] = bits.Rand()
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			err := dhts[index].announce(ids[index])
			if err != nil {
				t.Error("error announcing random bitmap - ", err)
			}
		}(i)
	}
	wg.Wait()

	// check that each node is in at learst 1 other routing table
	rtCounts := make(map[bits.Bitmap]int)
	for _, d := range dhts {
		for _, d2 := range dhts {
			if d.node.id.Equals(d2.node.id) {
				continue
			}
			c := d2.node.rt.GetClosest(d.node.id, 1)
			if len(c) > 1 {
				t.Error("rt returned more than one node when only one requested")
			} else if len(c) == 1 && c[0].ID.Equals(d.node.id) {
				rtCounts[d.node.id]++
			}
		}
	}

	for k, v := range rtCounts {
		if v == 0 {
			t.Errorf("%s was not in any routing tables", k.HexShort())
		}
	}

	// check that each ID is stored by at least 3 nodes
	storeCounts := make(map[bits.Bitmap]int)
	for _, d := range dhts {
		for _, id := range ids {
			if len(d.node.store.Get(id)) > 0 {
				storeCounts[id]++
			}
		}
	}

	for k, v := range storeCounts {
		if v == 0 {
			t.Errorf("%s was not stored by any nodes", k.HexShort())
		}
	}
}
