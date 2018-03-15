package dht

import (
	"testing"

	"github.com/zeebo/bencode"
)

func TestBitmap(t *testing.T) {
	a := bitmap{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23,
		24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35,
		36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
	}
	b := bitmap{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23,
		24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35,
		36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 47, 46,
	}
	c := bitmap{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1,
	}

	if !a.Equals(a) {
		t.Error("bitmap does not equal itself")
	}
	if a.Equals(b) {
		t.Error("bitmap equals another bitmap with different id")
	}

	if !a.Xor(b).Equals(c) {
		t.Error(a.Xor(b))
	}

	if c.PrefixLen() != 375 {
		t.Error(c.PrefixLen())
	}

	if b.Less(a) {
		t.Error("bitmap fails lessThan test")
	}

	id := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if newBitmapFromHex(id).Hex() != id {
		t.Error(newBitmapFromHex(id).Hex())
	}
}

func TestBitmapMarshal(t *testing.T) {
	b := newBitmapFromString("123456789012345678901234567890123456789012345678")
	encoded, err := bencode.EncodeBytes(b)
	if err != nil {
		t.Error(err)
	}

	if string(encoded) != "48:123456789012345678901234567890123456789012345678" {
		t.Error("encoding does not match expected")
	}
}

func TestBitmapMarshalEmbedded(t *testing.T) {
	e := struct {
		A string
		B bitmap
		C int
	}{
		A: "1",
		B: newBitmapFromString("222222222222222222222222222222222222222222222222"),
		C: 3,
	}

	encoded, err := bencode.EncodeBytes(e)
	if err != nil {
		t.Error(err)
	}

	if string(encoded) != "d1:A1:11:B48:2222222222222222222222222222222222222222222222221:Ci3ee" {
		t.Error("encoding does not match expected")
	}
}

func TestBitmapMarshalEmbedded2(t *testing.T) {
	encoded, err := bencode.EncodeBytes([]interface{}{
		newBitmapFromString("333333333333333333333333333333333333333333333333"),
	})
	if err != nil {
		t.Error(err)
	}

	if string(encoded) != "l48:333333333333333333333333333333333333333333333333e" {
		t.Error("encoding does not match expected")
	}
}

func TestBitmap_PrefixLen(t *testing.T) {
	tt := []struct {
		str string
		len int
	}{
		{len: 0, str: "F00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 0, str: "800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 1, str: "700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 1, str: "400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 384, str: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 383, str: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"},
		{len: 382, str: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002"},
		{len: 382, str: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"},
	}

	for _, test := range tt {
		len := newBitmapFromHex(test.str).PrefixLen()
		if len != test.len {
			t.Errorf("got prefix len %d; expected %d for %s", len, test.len, test.str)
		}
	}
}
