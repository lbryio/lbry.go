package address

import (
	"errors"
	"./base58"
)

func DecodeAddress(address string) ([address_length]byte, error) {
	decoded, err := base58.DecodeBase58(address, address_length)
	if err != nil {
		return [address_length]byte{}, errors.New("failed to decode")
	}
	buf := [address_length]byte{}
	for i, b := range decoded {
		buf[i] = b
	}
	return ValidateAddress(buf)
}
