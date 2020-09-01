package base58

import (
	"errors"
	"math/big"
)

func DecodeBase58(value string, size int64) ([]byte, error) {
	buf := []byte(value)
	longValue := big.NewInt(0)
	result := make([]byte, size)
	for i := int64(len(buf) - 1); i >= 0; i-- {
		to_add := big.NewInt(0)
		to_add = to_add.Exp(big.NewInt(58), big.NewInt(i), to_add)
		c, err := CharacterIndex(buf[int64(len(buf))-i-1])
		if err != nil {
			return result, err
		}
		to_add = to_add.Mul(c, to_add)
		longValue = longValue.Add(to_add, longValue)
	}
	for i := size - 1; i >= 0; i-- {
		m := big.NewInt(0)
		longValue, m = longValue.DivMod(longValue, big.NewInt(256), m)
		bs := m.Bytes()
		if len(bs) == 0 {
			bs = append(bs, 0x00)
		}
		b := byte(bs[0])
		result[i] = b
	}
	if longValue.Int64() != 0 {
		return result, errors.New("cannot decode to the given size")
	}
	if size != int64(len(result)) {
		return result, errors.New("length mismatch")
	}
	return result, nil
}
