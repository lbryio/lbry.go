package address

import (
	"errors"
	"github.com/lbryio/lbryschema.go/address/base58"
)

func DecodeAddress(address string, blockchainName string) ([addressLength]byte, error) {
	decoded, err := base58.DecodeBase58(address, addressLength)
	if err != nil {
		return [addressLength]byte{}, errors.New("failed to decode")
	}
	buf := [addressLength]byte{}
	for i, b := range decoded {
		buf[i] = b
	}

	return ValidateAddress(buf, blockchainName)
}
