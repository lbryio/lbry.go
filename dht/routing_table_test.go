package dht

import (
	"encoding/json"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/lbryio/lbry.go/v3/dht/bits"

	"github.com/sebdah/goldie"
)

func TestBucket_Split(t *testing.T) {
	rt := newRoutingTable(bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))
	if len(rt.buckets) != 1 {
		t.Errorf("there should only be one bucket so far")
	}
	if len(rt.buckets[0].peers) != 0 {
		t.Errorf("there should be no contacts yet")
	}

	var tests = []struct {
		name                  string
		id                    bits.Bitmap
		expectedBucketCount   int
		expectedTotalContacts int
	}{
		//fill first bucket
		{"b1-one", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100"), 1, 1},
		{"b1-two", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200"), 1, 2},
		{"b1-three", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300"), 1, 3},
		{"b1-four", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000400"), 1, 4},
		{"b1-five", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000500"), 1, 5},
		{"b1-six", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600"), 1, 6},
		{"b1-seven", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000700"), 1, 7},
		{"b1-eight", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800"), 1, 8},

		// split off second bucket and fill it
		{"b2-one", bits.FromHexP("001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 9},
		{"b2-two", bits.FromHexP("002000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 10},
		{"b2-three", bits.FromHexP("003000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 11},
		{"b2-four", bits.FromHexP("004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 12},
		{"b2-five", bits.FromHexP("005000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 13},
		{"b2-six", bits.FromHexP("006000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 14},
		{"b2-seven", bits.FromHexP("007000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 15},

		// at this point there are two buckets. the first has 7 contacts, the second has 8

		// inserts into the second bucket should be skipped
		{"dont-split", bits.FromHexP("009000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 15},

		// ... unless the ID is closer than the kth-closest contact
		{"split-kth-closest", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"), 2, 16},

		{"b3-two", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002"), 3, 17},
		{"b3-three", bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"), 3, 18},
	}

	for i, testCase := range tests {
		rt.Update(Contact{testCase.id, net.ParseIP("127.0.0.1"), 8000 + i, 0})

		if len(rt.buckets) != testCase.expectedBucketCount {
			t.Errorf("failed test case %s. there should be %d buckets, got %d", testCase.name, testCase.expectedBucketCount, len(rt.buckets))
		}
		if rt.Count() != testCase.expectedTotalContacts {
			t.Errorf("failed test case %s. there should be %d contacts, got %d", testCase.name, testCase.expectedTotalContacts, rt.Count())
		}
	}

	var testRanges = []struct {
		id       bits.Bitmap
		expected int
	}{
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"), 0},
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005"), 0},
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000410"), 1},
		{bits.FromHexP("0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007f0"), 1},
		{bits.FromHexP("F00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800"), 2},
		{bits.FromHexP("F00000000000000000000000000000000000000000000000000F00000000000000000000000000000000000000000000"), 2},
		{bits.FromHexP("F0000000000000000000000000000000F0000000000000000000000000F0000000000000000000000000000000000000"), 2},
	}

	for _, tt := range testRanges {
		bucket := bucketNumFor(rt, tt.id)
		if bucket != tt.expected {
			t.Errorf("bucketFor(%s, %s) => got %d, expected %d", tt.id.Hex(), rt.id.Hex(), bucket, tt.expected)
		}
	}
}

func bucketNumFor(rt *routingTable, target bits.Bitmap) int {
	if rt.id.Equals(target) {
		panic("routing table does not have a bucket for its own id")
	}
	distance := target.Xor(rt.id)
	for i := range rt.buckets {
		if rt.buckets[i].Range.Contains(distance) {
			return i
		}
	}
	panic("target is not contained in any buckets")
}

func TestBucket_Split_Continuous(t *testing.T) {
	b := newBucket(bits.MaxRange())

	left, right := b.Split()

	if !left.Range.Start.Equals(b.Range.Start) {
		t.Errorf("left bucket start does not align with original bucket start. got %s, expected %s", left.Range.Start, b.Range.Start)
	}

	if !right.Range.End.Equals(b.Range.End) {
		t.Errorf("right bucket end does not align with original bucket end. got %s, expected %s", right.Range.End, b.Range.End)
	}

	leftEndNext := (&big.Int{}).Add(left.Range.End.Big(), big.NewInt(1))
	if !bits.FromBigP(leftEndNext).Equals(right.Range.Start) {
		t.Errorf("there's a gap between left bucket end and right bucket start. end is %s, start is %s", left.Range.End, right.Range.Start)
	}
}

func TestBucket_Split_KthClosest_DoSplit(t *testing.T) {
	rt := newRoutingTable(bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))

	// add 4 low IDs
	rt.Update(Contact{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"), net.ParseIP("127.0.0.1"), 8001, 0})
	rt.Update(Contact{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002"), net.ParseIP("127.0.0.1"), 8002, 0})
	rt.Update(Contact{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"), net.ParseIP("127.0.0.1"), 8003, 0})
	rt.Update(Contact{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004"), net.ParseIP("127.0.0.1"), 8004, 0})

	// add 4 high IDs
	rt.Update(Contact{bits.FromHexP("800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8001, 0})
	rt.Update(Contact{bits.FromHexP("900000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8002, 0})
	rt.Update(Contact{bits.FromHexP("a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8003, 0})
	rt.Update(Contact{bits.FromHexP("b00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8004, 0})

	// split the bucket and fill the high bucket
	rt.Update(Contact{bits.FromHexP("c00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8005, 0})
	rt.Update(Contact{bits.FromHexP("d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8006, 0})
	rt.Update(Contact{bits.FromHexP("e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8007, 0})
	rt.Update(Contact{bits.FromHexP("f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8008, 0})

	// add a high ID. it should split because the high ID is closer than the Kth closest ID
	rt.Update(Contact{bits.FromHexP("910000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.1"), 8009, 0})

	if len(rt.buckets) != 3 {
		t.Errorf("expected 3 buckets, got %d", len(rt.buckets))
	}
	if rt.Count() != 13 {
		t.Errorf("expected 13 contacts, got %d", rt.Count())
	}
}

func TestBucket_Split_KthClosest_DontSplit(t *testing.T) {
	rt := newRoutingTable(bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))

	// add 4 low IDs
	rt.Update(Contact{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"), net.ParseIP("127.0.0.1"), 8001, 0})
	rt.Update(Contact{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002"), net.ParseIP("127.0.0.1"), 8002, 0})
	rt.Update(Contact{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"), net.ParseIP("127.0.0.1"), 8003, 0})
	rt.Update(Contact{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004"), net.ParseIP("127.0.0.1"), 8004, 0})

	// add 4 high IDs
	rt.Update(Contact{bits.FromHexP("800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8001, 0})
	rt.Update(Contact{bits.FromHexP("900000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8002, 0})
	rt.Update(Contact{bits.FromHexP("a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8003, 0})
	rt.Update(Contact{bits.FromHexP("b00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8004, 0})

	// split the bucket and fill the high bucket
	rt.Update(Contact{bits.FromHexP("c00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8005, 0})
	rt.Update(Contact{bits.FromHexP("d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8006, 0})
	rt.Update(Contact{bits.FromHexP("e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8007, 0})
	rt.Update(Contact{bits.FromHexP("f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.2"), 8008, 0})

	// add a really high ID. this should not split because its not closer than the Kth closest ID
	rt.Update(Contact{bits.FromHexP("ffff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), net.ParseIP("127.0.0.1"), 8009, 0})

	if len(rt.buckets) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(rt.buckets))
	}
	if rt.Count() != 12 {
		t.Errorf("expected 12 contacts, got %d", rt.Count())
	}
}

func TestRoutingTable_GetClosest(t *testing.T) {
	n1 := bits.FromHexP("FFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n2 := bits.FromHexP("FFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n3 := bits.FromHexP("111111110000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	rt := newRoutingTable(n1)
	rt.Update(Contact{n2, net.ParseIP("127.0.0.1"), 8001, 0})
	rt.Update(Contact{n3, net.ParseIP("127.0.0.1"), 8002, 0})

	contacts := rt.GetClosest(bits.FromHexP("222222220000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1)
	if len(contacts) != 1 {
		t.Fail()
		return
	}
	if !contacts[0].ID.Equals(n3) {
		t.Error(contacts[0])
	}
	contacts = rt.GetClosest(n2, 10)
	if len(contacts) != 2 {
		t.Error(len(contacts))
		return
	}
	if !contacts[0].ID.Equals(n2) {
		t.Error(contacts[0])
	}
	if !contacts[1].ID.Equals(n3) {
		t.Error(contacts[1])
	}
}

func TestRoutingTable_GetClosest_Empty(t *testing.T) {
	n1 := bits.FromShortHexP("1")
	rt := newRoutingTable(n1)

	contacts := rt.GetClosest(bits.FromShortHexP("a"), 3)
	if len(contacts) != 0 {
		t.Error("there shouldn't be any contacts")
		return
	}
}

func TestRoutingTable_Refresh(t *testing.T) {
	t.Skip("TODO: test routing table refreshing")
}

func TestRoutingTable_MoveToBack(t *testing.T) {
	tt := map[string]struct {
		data     []peer
		index    int
		expected []peer
	}{
		"simpleMove": {
			data:     []peer{{NumFailures: 0}, {NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}},
			index:    1,
			expected: []peer{{NumFailures: 0}, {NumFailures: 2}, {NumFailures: 3}, {NumFailures: 1}},
		},
		"moveFirst": {
			data:     []peer{{NumFailures: 0}, {NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}},
			index:    0,
			expected: []peer{{NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}, {NumFailures: 0}},
		},
		"moveLast": {
			data:     []peer{{NumFailures: 0}, {NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}},
			index:    3,
			expected: []peer{{NumFailures: 0}, {NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}},
		},
		"largeIndex": {
			data:     []peer{{NumFailures: 0}, {NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}},
			index:    27,
			expected: []peer{{NumFailures: 0}, {NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}},
		},
		"negativeIndex": {
			data:     []peer{{NumFailures: 0}, {NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}},
			index:    -12,
			expected: []peer{{NumFailures: 0}, {NumFailures: 1}, {NumFailures: 2}, {NumFailures: 3}},
		},
	}

	for name, test := range tt {
		moveToBack(test.data, test.index)
		expected := make([]string, len(test.expected))
		actual := make([]string, len(test.data))
		for i := range actual {
			actual[i] = strconv.Itoa(test.data[i].NumFailures)
			expected[i] = strconv.Itoa(test.expected[i].NumFailures)
		}

		expJoin := strings.Join(expected, ",")
		actJoin := strings.Join(actual, ",")

		if actJoin != expJoin {
			t.Errorf("%s failed: got %s; expected %s", name, actJoin, expJoin)
		}
	}
}

func TestRoutingTable_Save(t *testing.T) {
	t.Skip("fix me")
	id := bits.FromHexP("1c8aff71b99462464d9eeac639595ab99664be3482cb91a29d87467515c7d9158fe72aa1f1582dab07d8f8b5db277f41")
	rt := newRoutingTable(id)

	for i, b := range rt.buckets {
		for j := 0; j < bucketSize; j++ {
			toAdd := b.Range.Start.Add(bits.FromShortHexP(strconv.Itoa(j)))
			if toAdd.Cmp(b.Range.End) <= 0 {
				rt.Update(Contact{
					ID:   b.Range.Start.Add(bits.FromShortHexP(strconv.Itoa(j))),
					IP:   net.ParseIP("1.2.3." + strconv.Itoa(j)),
					Port: 1 + i*bucketSize + j,
				})
			}
		}
	}

	data, err := json.MarshalIndent(rt, "", "  ")
	if err != nil {
		t.Error(err)
	}

	goldie.Assert(t, t.Name(), data)
}

func TestRoutingTable_Load_ID(t *testing.T) {
	t.Skip("fix me")
	id := "1c8aff71b99462464d9eeac639595ab99664be3482cb91a29d87467515c7d9158fe72aa1f1582dab07d8f8b5db277f41"
	data := []byte(`{"id": "` + id + `","contacts": []}`)

	rt := routingTable{}
	err := json.Unmarshal(data, &rt)
	if err != nil {
		t.Error(err)
	}
	if rt.id.Hex() != id {
		t.Error("id mismatch")
	}
}

func TestRoutingTable_Load_Contacts(t *testing.T) {
	t.Skip("TODO")
}
