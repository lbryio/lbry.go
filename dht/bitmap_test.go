package dht

import (
	"fmt"
	"testing"

	"github.com/lyoshenka/bencode"
)

func TestBitmap(t *testing.T) {
	a := Bitmap{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23,
		24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35,
		36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
	}
	b := Bitmap{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23,
		24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35,
		36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 47, 46,
	}
	c := Bitmap{
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
	if BitmapFromHexP(id).Hex() != id {
		t.Error(BitmapFromHexP(id).Hex())
	}
}

func TestBitmap_GetBit(t *testing.T) {
	tt := []struct {
		hex      string
		bit      int
		expected bool
		panic    bool
	}{
		//{hex: "0", bit: 385, one: true, expected: "1", panic:true}, // should error
		//{hex: "0", bit: 384, one: true, expected: "1", panic:true},
		{bit: 383, expected: false, panic: false},
		{bit: 382, expected: true, panic: false},
		{bit: 381, expected: false, panic: false},
		{bit: 380, expected: true, panic: false},
	}

	b := BitmapFromShortHexP("a")

	for _, test := range tt {
		actual := getBit(b[:], test.bit)
		if test.expected != actual {
			t.Errorf("getting bit %d of %s: expected %t, got %t", test.bit, b.HexSimplified(), test.expected, actual)
		}
	}
}

func TestBitmap_SetBit(t *testing.T) {
	tt := []struct {
		hex      string
		bit      int
		one      bool
		expected string
		panic    bool
	}{
		{hex: "0", bit: 383, one: true, expected: "1", panic: false},
		{hex: "0", bit: 382, one: true, expected: "2", panic: false},
		{hex: "0", bit: 381, one: true, expected: "4", panic: false},
		{hex: "0", bit: 385, one: true, expected: "1", panic: true},
		{hex: "0", bit: 384, one: true, expected: "1", panic: true},
	}

	for _, test := range tt {
		expected := BitmapFromShortHexP(test.expected)
		actual := BitmapFromShortHexP(test.hex)
		if test.panic {
			assertPanic(t, fmt.Sprintf("setting bit %d to %t", test.bit, test.one), func() { setBit(actual[:], test.bit, test.one) })
		} else {
			setBit(actual[:], test.bit, test.one)
			if !expected.Equals(actual) {
				t.Errorf("setting bit %d to %t: expected %s, got %s", test.bit, test.one, test.expected, actual.HexSimplified())
			}

		}
	}
}

func TestBitmap_FromHexShort(t *testing.T) {
	tt := []struct {
		short string
		long  string
	}{
		{short: "", long: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{short: "0", long: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{short: "00000", long: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{short: "9473745bc", long: "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000009473745bc"},
		{short: "09473745bc", long: "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000009473745bc"},
		{short: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			long: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
	}

	for _, test := range tt {
		short := BitmapFromShortHexP(test.short)
		long := BitmapFromHexP(test.long)
		if !short.Equals(long) {
			t.Errorf("short hex %s: expected %s, got %s", test.short, long.Hex(), short.Hex())
		}
	}
}

func TestBitmapMarshal(t *testing.T) {
	b := BitmapFromStringP("123456789012345678901234567890123456789012345678")
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
		B Bitmap
		C int
	}{
		A: "1",
		B: BitmapFromStringP("222222222222222222222222222222222222222222222222"),
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
		BitmapFromStringP("333333333333333333333333333333333333333333333333"),
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
		hex string
		len int
	}{
		{len: 0, hex: "F00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 0, hex: "800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 1, hex: "700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 1, hex: "400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 384, hex: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{len: 383, hex: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"},
		{len: 382, hex: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002"},
		{len: 382, hex: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"},
	}

	for _, test := range tt {
		len := BitmapFromHexP(test.hex).PrefixLen()
		if len != test.len {
			t.Errorf("got prefix len %d; expected %d for %s", len, test.len, test.hex)
		}
	}
}

func TestBitmap_Prefix(t *testing.T) {
	allOne := BitmapFromHexP("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	zerosTT := []struct {
		zeros    int
		expected string
	}{
		{zeros: -123, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{zeros: 0, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{zeros: 1, expected: "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{zeros: 69, expected: "000000000000000007ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{zeros: 383, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"},
		{zeros: 384, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{zeros: 400, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
	}

	for _, test := range zerosTT {
		expected := BitmapFromHexP(test.expected)
		actual := allOne.Prefix(test.zeros, false)
		if !actual.Equals(expected) {
			t.Errorf("%d zeros: got %s; expected %s", test.zeros, actual.Hex(), expected.Hex())
		}
	}

	for i := 0; i < nodeIDLength*8; i++ {
		b := allOne.Prefix(i, false)
		if b.PrefixLen() != i {
			t.Errorf("got prefix len %d; expected %d for %s", b.PrefixLen(), i, b.Hex())
		}
	}

	allZero := BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

	onesTT := []struct {
		ones     int
		expected string
	}{
		{ones: -123, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ones: 0, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ones: 1, expected: "800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ones: 69, expected: "fffffffffffffffff8000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ones: 383, expected: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"},
		{ones: 384, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ones: 400, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	}

	for _, test := range onesTT {
		expected := BitmapFromHexP(test.expected)
		actual := allZero.Prefix(test.ones, true)
		if !actual.Equals(expected) {
			t.Errorf("%d ones: got %s; expected %s", test.ones, actual.Hex(), expected.Hex())
		}
	}
}

func TestBitmap_Suffix(t *testing.T) {
	allOne := BitmapFromHexP("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	zerosTT := []struct {
		zeros    int
		expected string
	}{
		{zeros: -123, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{zeros: 0, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{zeros: 1, expected: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"},
		{zeros: 69, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe00000000000000000"},
		{zeros: 383, expected: "800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{zeros: 384, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{zeros: 400, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
	}

	for _, test := range zerosTT {
		expected := BitmapFromHexP(test.expected)
		actual := allOne.Suffix(test.zeros, false)
		if !actual.Equals(expected) {
			t.Errorf("%d zeros: got %s; expected %s", test.zeros, actual.Hex(), expected.Hex())
		}
	}

	for i := 0; i < nodeIDLength*8; i++ {
		b := allOne.Prefix(i, false)
		if b.PrefixLen() != i {
			t.Errorf("got prefix len %d; expected %d for %s", b.PrefixLen(), i, b.Hex())
		}
	}

	allZero := BitmapFromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

	onesTT := []struct {
		ones     int
		expected string
	}{
		{ones: -123, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ones: 0, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ones: 1, expected: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"},
		{ones: 69, expected: "0000000000000000000000000000000000000000000000000000000000000000000000000000001fffffffffffffffff"},
		{ones: 383, expected: "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ones: 384, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ones: 400, expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	}

	for _, test := range onesTT {
		expected := BitmapFromHexP(test.expected)
		actual := allZero.Suffix(test.ones, true)
		if !actual.Equals(expected) {
			t.Errorf("%d ones: got %s; expected %s", test.ones, actual.Hex(), expected.Hex())
		}
	}
}

func TestBitmap_Add(t *testing.T) {
	tt := []struct {
		a, b, sum string
		panic     bool
	}{
		{"0", "0", "0", false},
		{"0", "1", "1", false},
		{"1", "0", "1", false},
		{"1", "1", "2", false},
		{"8", "4", "c", false},
		{"1000", "0010", "1010", false},
		{"1111", "1111", "2222", false},
		{"ffff", "1", "10000", false},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", false},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1", "", true},
	}

	for _, test := range tt {
		a := BitmapFromShortHexP(test.a)
		b := BitmapFromShortHexP(test.b)
		expected := BitmapFromShortHexP(test.sum)
		if test.panic {
			assertPanic(t, fmt.Sprintf("adding %s and %s", test.a, test.b), func() { a.Add(b) })
		} else {
			actual := a.Add(b)
			if !expected.Equals(actual) {
				t.Errorf("adding %s and %s; expected %s, got %s", test.a, test.b, test.sum, actual.HexSimplified())
			}
		}
	}
}

func TestBitmap_Sub(t *testing.T) {
	tt := []struct {
		a, b, sum string
		panic     bool
	}{
		{"0", "0", "0", false},
		{"1", "0", "1", false},
		{"1", "1", "0", false},
		{"8", "4", "4", false},
		{"f", "9", "6", false},
		{"f", "e", "1", false},
		{"10", "f", "1", false},
		{"2222", "1111", "1111", false},
		{"ffff", "1", "fffe", false},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", false},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0", false},
		{"0", "1", "", true},
	}

	for _, test := range tt {
		a := BitmapFromShortHexP(test.a)
		b := BitmapFromShortHexP(test.b)
		expected := BitmapFromShortHexP(test.sum)
		if test.panic {
			assertPanic(t, fmt.Sprintf("subtracting %s - %s", test.a, test.b), func() { a.Sub(b) })
		} else {
			actual := a.Sub(b)
			if !expected.Equals(actual) {
				t.Errorf("subtracting %s - %s; expected %s, got %s", test.a, test.b, test.sum, actual.HexSimplified())
			}
		}
	}
}
