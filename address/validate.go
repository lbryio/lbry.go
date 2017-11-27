package address

import (
	"./base58"
	"errors"
)

const lbrycrd_main_pubkey_prefix = byte(85)
const lbrycrd_main_script_prefix = byte(122)
const lbrycrd_testnet_pubkey_prefix = byte(111)
const lbrycrd_testnet_script_prefix = byte(196)
const lbrycrd_regtest_pubkey_prefix = byte(111)
const lbrycrd_regtest_script_prefix = byte(196)

const prefix_length = 1
const pubkey_length = 20
const checksum_length = 4
const address_length = prefix_length + pubkey_length + checksum_length
const lbrycrd_main = "lbrycrd_main"
const lbrycrd_testnet = "lbrycrd_testnet"
const lbrycrd_regtest = "lbrycrd_regtest"

var address_prefixes = map[string][2]byte{}

func SetPrefixes() {
	address_prefixes[lbrycrd_main] = [2]byte{lbrycrd_main_pubkey_prefix, lbrycrd_main_script_prefix}
	address_prefixes[lbrycrd_testnet] = [2]byte{lbrycrd_testnet_pubkey_prefix, lbrycrd_testnet_script_prefix}
	address_prefixes[lbrycrd_regtest] = [2]byte{lbrycrd_regtest_pubkey_prefix, lbrycrd_regtest_script_prefix}
}

func PrefixIsValid(address [address_length]byte, blockchainName string) bool {
	SetPrefixes()
	prefix := address[0]
	for _, addr_prefix := range address_prefixes[blockchainName] {
		if addr_prefix == prefix {
			return true
		}
	}
	return false
}

func PubKeyIsValid(address [address_length]byte) bool {
	pubkey := address[prefix_length : pubkey_length+prefix_length]
	// TODO: validate this for real
	if len(pubkey) != pubkey_length {
		return false
	}
	return true
}

func AddressChecksumIsValid(address [address_length]byte) bool {
	return base58.VerifyBase58Checksum(address[:])
}

func ValidateAddress(address [address_length]byte, blockchainName string) ([address_length]byte, error) {
	if blockchainName != lbrycrd_main && blockchainName != lbrycrd_testnet && blockchainName != lbrycrd_regtest {
		return address, errors.New("invalid blockchain name")
	}
	if !PrefixIsValid(address, blockchainName) {
		return address, errors.New("invalid prefix")
	}
	if !PubKeyIsValid(address) {
		return address, errors.New("invalid pubkey")
	}
	if !AddressChecksumIsValid(address) {
		return address, errors.New("invalid address checksum")
	}
	return address, nil
}
