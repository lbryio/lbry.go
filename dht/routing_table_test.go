package dht

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/lbryio/reflector.go/dht/bits"
	"github.com/sebdah/goldie"
)

func TestRoutingTable_bucketFor(t *testing.T) {
	rt := newRoutingTable(bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))
	var tests = []struct {
		id       bits.Bitmap
		expected int
	}{
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"), 0},
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002"), 1},
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"), 1},
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004"), 2},
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005"), 2},
		{bits.FromHexP("00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f"), 3},
		{bits.FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010"), 4},
		{bits.FromHexP("F00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), 383},
		{bits.FromHexP("F0000000000000000000000000000000F0000000000000000000000000F0000000000000000000000000000000000000"), 383},
	}

	for _, tt := range tests {
		bucket := rt.bucketNumFor(tt.id)
		if bucket != tt.expected {
			t.Errorf("bucketFor(%s, %s) => %d, want %d", tt.id.Hex(), rt.id.Hex(), bucket, tt.expected)
		}
	}
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

func TestRoutingTable_BucketRanges(t *testing.T) {
	id := bits.FromHexP("1c8aff71b99462464d9eeac639595ab99664be3482cb91a29d87467515c7d9158fe72aa1f1582dab07d8f8b5db277f41")
	ranges := newRoutingTable(id).BucketRanges()
	if !ranges[0].Start.Equals(ranges[0].End) {
		t.Error("first bucket should only fit exactly one id")
	}
	for i := 0; i < 1000; i++ {
		randID := bits.Rand()
		found := -1
		for i, r := range ranges {
			if r.Start.Cmp(randID) <= 0 && r.End.Cmp(randID) >= 0 {
				if found >= 0 {
					t.Errorf("%s appears in buckets %d and %d", randID.Hex(), found, i)
				} else {
					found = i
				}
			}
		}
		if found < 0 {
			t.Errorf("%s did not appear in any bucket", randID.Hex())
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
