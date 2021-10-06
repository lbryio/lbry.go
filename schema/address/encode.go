package address

import (
	"github.com/lbryio/lbry.go/v3/schema/address/base58"
)

func EncodeAddress(address [addressLength]byte, blockchainName string) (string, error) {
	buf, err := ValidateAddress(address, blockchainName)
	if err != nil {
		return "", err
	}
	return base58.EncodeBase58(buf[:]), nil
}
