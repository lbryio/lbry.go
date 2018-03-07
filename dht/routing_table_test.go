package dht

import "testing"

func TestRoutingTable(t *testing.T) {
	n1 := newBitmapFromHex("FFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n2 := newBitmapFromHex("FFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n3 := newBitmapFromHex("111111110000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	rt := NewRoutingTable(&Node{n1, "localhost:8000"})
	rt.Update(&Node{n2, "localhost:8001"})
	rt.Update(&Node{n3, "localhost:8002"})

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
