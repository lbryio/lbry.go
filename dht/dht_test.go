package dht

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestNodeFinder_FindNodes(t *testing.T) {
	bs, dhts := TestingCreateDHT(t, 3, true, false)
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
	_, dhts := TestingCreateDHT(t, 3, false, false)
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
	bs, dhts := TestingCreateDHT(t, 3, true, false)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
		bs.Shutdown()
	}()

	blobHashToFind := RandomBitmapP()
	nodeToFind := Contact{ID: RandomBitmapP(), IP: net.IPv4(1, 2, 3, 4), Port: 5678}
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

	if !foundNodes[0].ID.Equals(nodeToFind.ID) {
		t.Fatalf("found node id %s, expected %s", foundNodes[0].ID.Hex(), nodeToFind.ID.Hex())
	}
}

func TestDHT_LargeDHT(t *testing.T) {
	nodes := 100
	bs, dhts := TestingCreateDHT(t, nodes, true, true)
	defer func() {
		for _, d := range dhts {
			go d.Shutdown()
		}
		bs.Shutdown()
		time.Sleep(1 * time.Second)
	}()

	wg := &sync.WaitGroup{}
	ids := make([]Bitmap, nodes)
	for i := range ids {
		ids[i] = RandomBitmapP()
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := dhts[index].Announce(ids[index]); err != nil {
				t.Error("error announcing random bitmap - ", err)
			}
		}(i)
	}
	wg.Wait()

	// check that each node is in at learst 1 other routing table
	rtCounts := make(map[Bitmap]int)
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
	storeCounts := make(map[Bitmap]int)
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
