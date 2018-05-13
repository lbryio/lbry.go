package dht

import (
	"net"
	"reflect"
	"testing"
)

func TestRoutingTable_bucketFor(t *testing.T) {
	rt := newRoutingTable(BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))
	var tests = []struct {
		id       Bitmap
		expected int
	}{
		{BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"), 0},
		{BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002"), 1},
		{BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"), 1},
		{BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004"), 2},
		{BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005"), 2},
		{BitmapFromHexP("00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f"), 3},
		{BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010"), 4},
		{BitmapFromHexP("F00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 383},
		{BitmapFromHexP("F0000000000000000000000000000000F0000000000000000000000000F0000000000000000000000000000000000000"), 383},
	}

	for _, tt := range tests {
		bucket := rt.bucketNumFor(tt.id)
		if bucket != tt.expected {
			t.Errorf("bucketFor(%s, %s) => %d, want %d", tt.id.Hex(), rt.id.Hex(), bucket, tt.expected)
		}
	}
}

func TestRoutingTable(t *testing.T) {
	n1 := BitmapFromHexP("FFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n2 := BitmapFromHexP("FFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n3 := BitmapFromHexP("111111110000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	rt := newRoutingTable(n1)
	rt.Update(Contact{n2, net.ParseIP("127.0.0.1"), 8001})
	rt.Update(Contact{n3, net.ParseIP("127.0.0.1"), 8002})

	contacts := rt.GetClosest(BitmapFromHexP("222222220000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1)
	if len(contacts) != 1 {
		t.Fail()
		return
	}
	if !contacts[0].id.Equals(n3) {
		t.Error(contacts[0])
	}

	contacts = rt.GetClosest(n2, 10)
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
	c := Contact{
		id:   BitmapFromHexP("1c8aff71b99462464d9eeac639595ab99664be3482cb91a29d87467515c7d9158fe72aa1f1582dab07d8f8b5db277f41"),
		ip:   net.ParseIP("1.2.3.4"),
		port: int(55<<8 + 66),
	}

	var compact []byte
	compact, err := c.MarshalCompact()
	if err != nil {
		t.Fatal(err)
	}

	if len(compact) != compactNodeInfoLength {
		t.Fatalf("got length of %d; expected %d", len(compact), compactNodeInfoLength)
	}

	if !reflect.DeepEqual(compact, append([]byte{1, 2, 3, 4, 55, 66}, c.id[:]...)) {
		t.Errorf("compact bytes not encoded correctly")
	}
}

func TestRoutingTableRefresh(t *testing.T) {
	t.Skip("TODO: test routing table refreshing")
}
