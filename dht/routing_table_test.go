package dht

import (
	"net"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestRoutingTable(t *testing.T) {
	n1 := newBitmapFromHex("FFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n2 := newBitmapFromHex("FFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n3 := newBitmapFromHex("111111110000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	rt := newRoutingTable(&Node{n1, net.ParseIP("127.0.0.1"), 8000})
	rt.Update(&Node{n2, net.ParseIP("127.0.0.1"), 8001})
	rt.Update(&Node{n3, net.ParseIP("127.0.0.1"), 8002})

	contacts := rt.FindClosest(newBitmapFromHex("222222220000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1)
	if len(contacts) != 1 {
		t.Fail()
		return
	}
	if !contacts[0].id.Equals(n3) {
		t.Error(contacts[0])
	}

	contacts = rt.FindClosest(n2, 10)
	if len(contacts) != 2 {
		t.Error(len(contacts))
		return
	}
	if !contacts[0].id.Equals(n2) {
		t.Error(contacts[0])
	}
	if !contacts[1].id.Equals(n3) {
		t.Error(contacts[1])
	}
}

func TestCompactEncoding(t *testing.T) {
	n := Node{
		id:   newBitmapFromHex("1c8aff71b99462464d9eeac639595ab99664be3482cb91a29d87467515c7d9158fe72aa1f1582dab07d8f8b5db277f41"),
		ip:   net.ParseIP("255.1.0.155"),
		port: 66666,
	}

	var compact []byte
	compact, err := n.MarshalCompact()
	if err != nil {
		t.Fatal(err)
	}

	if len(compact) != nodeIDLength+6 {
		t.Fatalf("got length of %d; expected %d", len(compact), nodeIDLength+6)
	}

	spew.Dump(compact)
}
