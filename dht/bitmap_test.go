package dht

import "testing"

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
