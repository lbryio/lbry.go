package address

import (
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/schema/address/base58"
)

func DecodeAddress(address string, blockchainName string) ([addressLength]byte, error) {
	decoded, err := base58.DecodeBase58(address, addressLength)
	if err != nil {
		return [addressLength]byte{}, errors.Err("failed to decode")
	}
	buf := [addressLength]byte{}
	for i, b := range decoded {
		buf[i] = b
	}

	return ValidateAddress(buf, blockchainName)
}
