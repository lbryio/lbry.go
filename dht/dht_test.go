package dht

import (
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

// TODO: make a dht with X nodes, have them all join, then ensure that every node appears at least once in another node's routing table

func TestNodeFinder_FindNodes(t *testing.T) {
	dhts := TestingCreateDHT(3)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
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

	if len(foundNodes) != 1 {
		t.Errorf("expected 1 node, found %d", len(foundNodes))
	}

	foundOne := false
	//foundTwo := false

	for _, n := range foundNodes {
		if n.id.Equals(dhts[0].node.id) {
			foundOne = true
		}
		//if n.id.Equals(dhts[1].node.c.id) {
		//	foundTwo = true
		//}
	}

	if !foundOne {
		t.Errorf("did not find first node %s", dhts[0].node.id.Hex())
	}
	//if !foundTwo {
	//	t.Errorf("did not find second node %s", dhts[1].node.c.id.Hex())
	//}
}

func TestNodeFinder_FindValue(t *testing.T) {
	dhts := TestingCreateDHT(3)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
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
	rand.Seed(time.Now().UnixNano())
	log.Println("if this takes longer than 20 seconds, its stuck. idk why it gets stuck sometimes, but its a bug.")
	nodes := 100
	dhts := TestingCreateDHT(nodes)
	defer func() {
		for _, d := range dhts {
			go d.Shutdown()
		}
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
			r := rand.Intn(nodes)
			wg.Add(1)
			defer wg.Done()
			dhts[r].Announce(ids[i])
		}(i)
	}
	wg.Wait()

	dhts[1].PrintState()
}
