package address

import (
	"github.com/lbryio/lbryschema.go/address/base58"
)

func EncodeAddress(address [addressLength]byte, blockchainName string) (string, error) {
	buf, err := ValidateAddress(address, blockchainName)
	if err != nil {
		return "", err
	}
	return base58.EncodeBase58(buf[:]), nil
}
