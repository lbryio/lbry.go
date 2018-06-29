package dht

import (
	"encoding/json"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"github.com/lbryio/reflector.go/dht/bits"
	"github.com/sebdah/goldie"
)


func checkBucketCount(rt *routingTable, t *testing.T, correctSize, correctCount, testCaseIndex int) {
	if len(rt.buckets) != correctSize {
		t.Errorf("failed test case %d. there should be %d buckets, got %d", testCaseIndex + 1, correctSize, len(rt.buckets))
	}
	if rt.Count() != correctCount {
		t.Errorf("failed test case %d. there should be %d contacts, got %d", testCaseIndex + 1, correctCount, rt.Count())
	}

}

func checkRangeContinuity(rt *routingTable, t *testing.T) {
	position := big.NewInt(0)
	for i, bucket := range rt.buckets {
		bucketStart := bucket.bucketRange.Start.Big()
		if bucketStart.Cmp(position) != 0 {
			t.Errorf("invalid start of bucket range: %s vs %s", position.String(), bucketStart.String())
		}
		if bucketStart.Cmp(bucket.bucketRange.End.Big()) != -1 {
			t.Error("range start is not less than bucket end")
		}
		position = bucket.bucketRange.End.Big()
		if i != len(rt.buckets) - 1 {
			position.Add(position, big.NewInt(1))
		}
	}
	if position.Cmp(bits.MaxP().Big()) != 0 {
		t.Errorf("range does not cover the whole keyspace, %s vs %s", bits.FromBigP(position).String(), bits.MaxP().String())
	}
}

func TestSplitBuckets(t *testing.T) {
	rt := newRoutingTable(bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))
	if len(rt.buckets) != 1 {
		t.Errorf("there should only be one bucket so far")
	}
	if len(rt.buckets[0].peers) != 0 {
		t.Errorf("there should be no contacts yet")
	}

	var tests = []struct {
		id       bits.Bitmap
		expectedBucketCount int
		expectedTotalContacts int
	}{
		//fill first bucket
		{bits.FromHexP("F00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1, 1},
		{bits.FromHexP("FF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1, 2},
		{bits.FromHexP("FFF000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1, 3},
		{bits.FromHexP("FFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1, 4},
		{bits.FromHexP("FFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1, 5},
		{bits.FromHexP("FFFFFF000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1, 6},
		{bits.FromHexP("FFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1, 7},
		{bits.FromHexP("FFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 1, 8},

		// fill second bucket
		{bits.FromHexP("FFFFFFFFF000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 9},
		{bits.FromHexP("FFFFFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 10},
		{bits.FromHexP("FFFFFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 11},
		{bits.FromHexP("FFFFFFFFFFFF000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 12},
		{bits.FromHexP("FFFFFFFFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 13},
		{bits.FromHexP("FFFFFFFFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 14},
		{bits.FromHexP("FFFFFFFFFFFFFFF000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 15},
		{bits.FromHexP("FFFFFFFFFFFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 16},

		// this should be skipped (no split should occur)
		{bits.FromHexP("FFFFFFFFFFFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2, 16},

		{bits.FromHexP("100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 3, 17},
		{bits.FromHexP("200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 3, 18},
		{bits.FromHexP("300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 3, 19},

		{bits.FromHexP("400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 4, 20},
		{bits.FromHexP("500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 4, 21},
		{bits.FromHexP("600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 4, 22},
		{bits.FromHexP("700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 4, 23},
		{bits.FromHexP("800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 4, 24},
		{bits.FromHexP("900000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 4, 25},
		{bits.FromHexP("A00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 4, 26},
		{bits.FromHexP("B00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 4, 27},
	}
	for i, testCase := range tests {
		rt.Update(Contact{testCase.id, net.ParseIP("127.0.0.1"), 8000 + i})
		checkBucketCount(rt, t, testCase.expectedBucketCount, testCase.expectedTotalContacts, i)
		checkRangeContinuity(rt, t)
	}

	var testRanges = []struct {
		id       bits.Bitmap
		expected int
	}{
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"), 0},
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005"), 0},
		{bits.FromHexP("200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010"), 1},
		{bits.FromHexP("380000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 2},
		{bits.FromHexP("F00000000000000000000000000000000000000000000000000F00000000000000000000000000000000000000000000"), 3},
		{bits.FromHexP("F0000000000000000000000000000000F0000000000000000000000000F0000000000000000000000000000000000000"), 3},
	}

	for _, tt := range testRanges {
		bucket := rt.bucketNumFor(tt.id)
		if bucket != tt.expected {
			t.Errorf("bucketFor(%s, %s) => %d, want %d", tt.id.Hex(), rt.id.Hex(), bucket, tt.expected)
		}
	}

	rt.printBucketInfo()
}

func TestRoutingTable_GetClosest(t *testing.T) {
	n1 := bits.FromHexP("FFFFFFFF0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n2 := bits.FromHexP("FFFFFFF00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	n3 := bits.FromHexP("111111110000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	rt := newRoutingTable(n1)
	rt.Update(Contact{n2, net.ParseIP("127.0.0.1"), 8001})
	rt.Update(Contact{n3, net.ParseIP("127.0.0.1"), 8002})

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
	id := bits.FromHexP("1c8aff71b99462464d9eeac639595ab99664be3482cb91a29d87467515c7d9158fe72aa1f1582dab07d8f8b5db277f41")
	rt := newRoutingTable(id)

	ranges := rt.BucketRanges()

	for i, r := range ranges {
		for j := 0; j < bucketSize; j++ {
			toAdd := r.Start.Add(bits.FromShortHexP(strconv.Itoa(j)))
			if toAdd.Cmp(r.End) <= 0 {
				rt.Update(Contact{
					ID:   r.Start.Add(bits.FromShortHexP(strconv.Itoa(j))),
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
