package dht

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/lbryio/errors.go"
	"github.com/lyoshenka/bencode"
)

// TODO: http://roaringbitmap.org/

type Bitmap [nodeIDLength]byte

func (b Bitmap) RawString() string {
	return string(b[:])
}

// BString returns the bitmap as a string of 0s and 1s
func (b Bitmap) BString() string {
	var buf bytes.Buffer
	for i := 0; i < nodeIDBits; i++ {
		if b.Get(i) {
			buf.WriteString("1")
		} else {
			buf.WriteString("0")
		}
	}
	return buf.String()
}

func (b Bitmap) Hex() string {
	return hex.EncodeToString(b[:])
}

func (b Bitmap) HexShort() string {
	return hex.EncodeToString(b[:4])
}

func (b Bitmap) HexSimplified() string {
	simple := strings.TrimLeft(b.Hex(), "0")
	if simple == "" {
		simple = "0"
	}
	return simple
}

func (b Bitmap) Equals(other Bitmap) bool {
	for k := range b {
		if b[k] != other[k] {
			return false
		}
	}
	return true
}

func (b Bitmap) Less(other interface{}) bool {
	for k := range b {
		if b[k] != other.(Bitmap)[k] {
			return b[k] < other.(Bitmap)[k]
		}
	}
	return false
}

func (b Bitmap) LessOrEqual(other interface{}) bool {
	if bm, ok := other.(Bitmap); ok && b.Equals(bm) {
		return true
	}
	return b.Less(other)
}

func (b Bitmap) Greater(other interface{}) bool {
	for k := range b {
		if b[k] != other.(Bitmap)[k] {
			return b[k] > other.(Bitmap)[k]
		}
	}
	return false
}

func (b Bitmap) GreaterOrEqual(other interface{}) bool {
	if bm, ok := other.(Bitmap); ok && b.Equals(bm) {
		return true
	}
	return b.Greater(other)
}

func (b Bitmap) Copy() Bitmap {
	var ret Bitmap
	copy(ret[:], b[:])
	return ret
}

func (b Bitmap) Xor(other Bitmap) Bitmap {
	var ret Bitmap
	for k := range b {
		ret[k] = b[k] ^ other[k]
	}
	return ret
}

func (b Bitmap) And(other Bitmap) Bitmap {
	var ret Bitmap
	for k := range b {
		ret[k] = b[k] & other[k]
	}
	return ret
}

func (b Bitmap) Or(other Bitmap) Bitmap {
	var ret Bitmap
	for k := range b {
		ret[k] = b[k] | other[k]
	}
	return ret
}

func (b Bitmap) Not() Bitmap {
	var ret Bitmap
	for k := range b {
		ret[k] = ^b[k]
	}
	return ret
}

func (b Bitmap) add(other Bitmap) (Bitmap, bool) {
	var ret Bitmap
	carry := false
	for i := nodeIDBits - 1; i >= 0; i-- {
		bBit := getBit(b[:], i)
		oBit := getBit(other[:], i)
		setBit(ret[:], i, bBit != oBit != carry)
		carry = (bBit && oBit) || (bBit && carry) || (oBit && carry)
	}
	return ret, carry
}

func (b Bitmap) Add(other Bitmap) Bitmap {
	ret, carry := b.add(other)
	if carry {
		panic("overflow in bitmap addition")
	}
	return ret
}

func (b Bitmap) Sub(other Bitmap) Bitmap {
	if b.Less(other) {
		panic("negative bitmaps not supported")
	}
	complement, _ := other.Not().add(BitmapFromShortHexP("1"))
	ret, _ := b.add(complement)
	return ret
}

func (b Bitmap) Get(n int) bool {
	return getBit(b[:], n)
}

func (b Bitmap) Set(n int, one bool) Bitmap {
	ret := b.Copy()
	setBit(ret[:], n, one)
	return ret
}

// PrefixLen returns the number of leading 0 bits
func (b Bitmap) PrefixLen() int {
	for i := range b {
		for j := 0; j < 8; j++ {
			if (b[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}
	return nodeIDBits
}

// Prefix returns a copy of b with the first n bits set to 1 (if `one` is true) or 0 (if `one` is false)
// https://stackoverflow.com/a/23192263/182709
func (b Bitmap) Prefix(n int, one bool) Bitmap {
	ret := b.Copy()

Outer:
	for i := range ret {
		for j := 0; j < 8; j++ {
			if i*8+j < n {
				if one {
					ret[i] |= 1 << uint(7-j)
				} else {
					ret[i] &= ^(1 << uint(7-j))
				}
			} else {
				break Outer
			}
		}
	}

	return ret
}

// Syffix returns a copy of b with the last n bits set to 1 (if `one` is true) or 0 (if `one` is false)
// https://stackoverflow.com/a/23192263/182709
func (b Bitmap) Suffix(n int, one bool) Bitmap {
	ret := b.Copy()

Outer:
	for i := len(ret) - 1; i >= 0; i-- {
		for j := 7; j >= 0; j-- {
			if i*8+j >= nodeIDBits-n {
				if one {
					ret[i] |= 1 << uint(7-j)
				} else {
					ret[i] &= ^(1 << uint(7-j))
				}
			} else {
				break Outer
			}
		}
	}

	return ret
}

func (b Bitmap) MarshalBencode() ([]byte, error) {
	str := string(b[:])
	return bencode.EncodeBytes(str)
}

func (b *Bitmap) UnmarshalBencode(encoded []byte) error {
	var str string
	err := bencode.DecodeBytes(encoded, &str)
	if err != nil {
		return err
	}
	if len(str) != nodeIDLength {
		return errors.Err("invalid bitmap length")
	}
	copy(b[:], str)
	return nil
}

func BitmapFromBytes(data []byte) (Bitmap, error) {
	var bmp Bitmap

	if len(data) != len(bmp) {
		return bmp, errors.Err("invalid bitmap of length %d", len(data))
	}

	copy(bmp[:], data)
	return bmp, nil
}

func BitmapFromBytesP(data []byte) Bitmap {
	bmp, err := BitmapFromBytes(data)
	if err != nil {
		panic(err)
	}
	return bmp
}

func BitmapFromString(data string) (Bitmap, error) {
	return BitmapFromBytes([]byte(data))
}

func BitmapFromStringP(data string) Bitmap {
	bmp, err := BitmapFromString(data)
	if err != nil {
		panic(err)
	}
	return bmp
}

func BitmapFromHex(hexStr string) (Bitmap, error) {
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return Bitmap{}, errors.Err(err)
	}
	return BitmapFromBytes(decoded)
}

func BitmapFromHexP(hexStr string) Bitmap {
	bmp, err := BitmapFromHex(hexStr)
	if err != nil {
		panic(err)
	}
	return bmp
}

func BitmapFromShortHex(hexStr string) (Bitmap, error) {
	return BitmapFromHex(strings.Repeat("0", nodeIDLength*2-len(hexStr)) + hexStr)
}

func BitmapFromShortHexP(hexStr string) Bitmap {
	bmp, err := BitmapFromShortHex(hexStr)
	if err != nil {
		panic(err)
	}
	return bmp
}

func RandomBitmapP() Bitmap {
	var id Bitmap
	_, err := rand.Read(id[:])
	if err != nil {
		panic(err)
	}
	return id
}

func RandomBitmapInRangeP(low, high Bitmap) Bitmap {
	diff := high.Sub(low)
	r := RandomBitmapP()
	for r.Greater(diff) {
		r = r.Sub(diff)
	}
	return r.Add(low)
}

func getBit(b []byte, n int) bool {
	i := n / 8
	j := n % 8
	return b[i]&(1<<uint(7-j)) > 0
}

func setBit(b []byte, n int, one bool) {
	i := n / 8
	j := n % 8
	if one {
		b[i] |= 1 << uint(7-j)
	} else {
		b[i] &= ^(1 << uint(7-j))
	}
}
