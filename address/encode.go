package address

import (
	"./base58"
)

func EncodeAddress(address [address_length]byte, blockchainName string) (string, error) {
	buf, err := ValidateAddress(address, blockchainName)
	if err != nil {
		return "", err
	}
	return base58.EncodeBase58(buf[:]), nil
}
