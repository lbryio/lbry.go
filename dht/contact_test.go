package dht

import (
	"net"
	"reflect"
	"testing"

	"github.com/lbryio/lbry.go/v2/dht/bits"
)

func TestCompactEncoding(t *testing.T) {
	c := Contact{
		ID:       bits.FromHexP("1c8aff71b99462464d9eeac639595ab99664be3482cb91a29d87467515c7d9158fe72aa1f1582dab07d8f8b5db277f41"),
		IP:       net.ParseIP("1.2.3.4"),
		PeerPort: int(55<<8 + 66),
	}

	var compact []byte
	compact, err := c.MarshalCompact()
	if err != nil {
		t.Fatal(err)
	}

	if len(compact) != compactNodeInfoLength {
		t.Fatalf("got length of %d; expected %d", len(compact), compactNodeInfoLength)
	}

	if !reflect.DeepEqual(compact, append([]byte{1, 2, 3, 4, 55, 66}, c.ID[:]...)) {
		t.Errorf("compact bytes not encoded correctly")
	}
}
