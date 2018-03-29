package dht

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func TestDHT_FindNodes(t *testing.T) {
	//log.SetLevel(log.DebugLevel)

	id1 := newRandomBitmap()
	id2 := newRandomBitmap()
	id3 := newRandomBitmap()

	seedIP := "127.0.0.1:21216"

	dht, err := New(&Config{Address: seedIP, NodeID: id1.Hex()})
	if err != nil {
		t.Fatal(err)
	}
	go dht.Start()

	time.Sleep(1 * time.Second)

	dht2, err := New(&Config{Address: "127.0.0.1:21217", NodeID: id2.Hex(), SeedNodes: []string{seedIP}})
	if err != nil {
		t.Fatal(err)
	}
	go dht2.Start()

	time.Sleep(1 * time.Second) // give dhts a chance to connect

	dht3, err := New(&Config{Address: "127.0.0.1:21218", NodeID: id3.Hex(), SeedNodes: []string{seedIP}})
	if err != nil {
		t.Fatal(err)
	}
	go dht3.Start()

	time.Sleep(1 * time.Second) // give dhts a chance to connect

	spew.Dump(dht3.FindNodes(id2))
}
