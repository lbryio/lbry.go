package address

import (
	"errors"
	"crypto/sha256"
)

const pubkey_prefix = byte(85)
const script_prefix = byte(122)
const prefix_length = 1
const pubkey_length = 20
const checksum_length = 4
const address_length = prefix_length + pubkey_length + checksum_length
var address_prefixes = [2]byte {pubkey_prefix, script_prefix}


func PrefixIsValid(address [address_length]byte) bool {
	prefix := address[0]
	for _, addr_prefix := range address_prefixes {
		if addr_prefix == prefix {
			return true
		}
	}
	return false
}

func PubKeyIsValid(address [address_length]byte) bool {
	pubkey := address[prefix_length:pubkey_length+prefix_length]
	// TODO: validate this for real
	if len(pubkey) != pubkey_length {
		return false
	}
	return true
}

func ChecksumIsValid(address [address_length]byte) bool {
	checksum := [checksum_length]byte{}
	for i := range checksum {checksum[i] = address[prefix_length+pubkey_length+i]}
	real_checksum := sha256.Sum256(address[:prefix_length+pubkey_length])
	real_checksum = sha256.Sum256(real_checksum[:])
	for i, c := range checksum {
		if c != real_checksum[i] {
			return false
		}
	}
	return true
}

func ValidateAddress(address [address_length]byte) ([address_length]byte, error) {
	if !PrefixIsValid(address) {
		return address, errors.New("invalid prefix")
	}
	if !PubKeyIsValid(address) {
		return address, errors.New("invalid pubkey")
	}
	if !ChecksumIsValid(address) {
		return address, errors.New("invalid address checksum")
	}
	return address, nil
}
