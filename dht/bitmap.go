package dht

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/lbryio/errors.go"
	"github.com/lyoshenka/bencode"
)

type Bitmap [nodeIDLength]byte

func (b Bitmap) RawString() string {
	return string(b[:])
}

func (b Bitmap) Hex() string {
	return hex.EncodeToString(b[:])
}

func (b Bitmap) HexShort() string {
	return hex.EncodeToString(b[:4])
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

func (b Bitmap) Xor(other Bitmap) Bitmap {
	var ret Bitmap
	for k := range b {
		ret[k] = b[k] ^ other[k]
	}
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
	return numBuckets
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

func RandomBitmapP() Bitmap {
	var id Bitmap
	_, err := rand.Read(id[:])
	if err != nil {
		panic(err)
	}
	return id
}
