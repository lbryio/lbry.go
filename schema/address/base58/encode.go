package base58

import (
	"math/big"
)

func EncodeBase58(data []byte) string {
	longValue := big.NewInt(0)
	result := ""
	for i := 0; i < len(data); i++ {
		to_add := big.NewInt(0)
		to_add = to_add.Exp(big.NewInt(256), big.NewInt(int64(i)), to_add)
		to_add = to_add.Mul(big.NewInt(int64(data[24-i])), to_add)
		longValue = longValue.Add(to_add, longValue)
	}
	i := 0
	for {
		m := big.NewInt(0)
		longValue, m = longValue.DivMod(longValue, big.NewInt(58), m)
		bs := m.Bytes()
		if len(bs) == 0 {
			bs = append(bs, 0x00)
		}
		b := b58Characters[bs[0]]
		result = string(b) + result
		if longValue.Int64() == 0 {
			break
		}
		i += 1
	}
	return result
}
