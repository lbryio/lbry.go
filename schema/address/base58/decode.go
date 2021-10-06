package base58

import (
	"math/big"

	"github.com/cockroachdb/errors"
)

func DecodeBase58(value string, size int64) ([]byte, error) {
	buf := []byte(value)
	longValue := big.NewInt(0)
	result := make([]byte, size)
	for i := int64(len(buf) - 1); i >= 0; i-- {
		toAdd := big.NewInt(0)
		toAdd = toAdd.Exp(big.NewInt(58), big.NewInt(i), toAdd)
		c, err := CharacterIndex(buf[int64(len(buf))-i-1])
		if err != nil {
			return result, err
		}
		toAdd = toAdd.Mul(c, toAdd)
		longValue = longValue.Add(toAdd, longValue)
	}
	for i := size - 1; i >= 0; i-- {
		m := big.NewInt(0)
		longValue, m = longValue.DivMod(longValue, big.NewInt(256), m)
		bs := m.Bytes()
		if len(bs) == 0 {
			bs = append(bs, 0x00)
		}
		result[i] = bs[0]
	}
	if longValue.Int64() != 0 {
		return result, errors.New("cannot decode to the given size")
	}
	if size != int64(len(result)) {
		return result, errors.New("length mismatch")
	}
	return result, nil
}
