package dht

import (
	"math/rand"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestNodeFinder_FindNodes(t *testing.T) {
	dhts := makeDHT(t, 3)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
	}()

	nf := newNodeFinder(dhts[2], newRandomBitmap(), false)
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
		if n.id.Equals(dhts[0].node.id) {
			foundOne = true
		}
		if n.id.Equals(dhts[1].node.id) {
			foundTwo = true
		}
	}

	if !foundOne {
		t.Errorf("did not find node %s", dhts[0].node.id.Hex())
	}
	if !foundTwo {
		t.Errorf("did not find node %s", dhts[1].node.id.Hex())
	}
}

func TestNodeFinder_FindValue(t *testing.T) {
	dhts := makeDHT(t, 3)
	defer func() {
		for i := range dhts {
			dhts[i].Shutdown()
		}
	}()

	blobHashToFind := newRandomBitmap()
	nodeToFind := Node{id: newRandomBitmap(), ip: net.IPv4(1, 2, 3, 4), port: 5678}
	dhts[0].store.Upsert(blobHashToFind, nodeToFind)

	nf := newNodeFinder(dhts[2], blobHashToFind, true)
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

func TestDHT_LargeDHT(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	log.Println("if this takes longer than 20 seconds, its stuck. idk why it gets stuck sometimes, but its a bug.")
	nodes := 100
	dhts := makeDHT(t, nodes)
	defer func() {
		for _, d := range dhts {
			go d.Shutdown()
		}
		time.Sleep(1 * time.Second)
	}()

	wg := &sync.WaitGroup{}
	numIDs := nodes / 2
	ids := make([]bitmap, numIDs)
	for i := 0; i < numIDs; i++ {
		ids[i] = newRandomBitmap()
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

func makeDHT(t *testing.T, numNodes int) []*DHT {
	if numNodes < 1 {
		return nil
	}

	ip := "127.0.0.1"
	firstPort := 21000
	dhts := make([]*DHT, numNodes)

	for i := 0; i < numNodes; i++ {
		seeds := []string{}
		if i > 0 {
			seeds = []string{ip + ":" + strconv.Itoa(firstPort)}
		}

		dht, err := New(&Config{Address: ip + ":" + strconv.Itoa(firstPort+i), NodeID: newRandomBitmap().Hex(), SeedNodes: seeds})
		if err != nil {
			t.Fatal(err)
		}

		go dht.Start()
		dht.WaitUntilJoined()
		dhts[i] = dht
	}

	return dhts
}
