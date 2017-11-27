package base58

import "crypto/sha256"

const checksum_length = 4

func VerifyBase58Checksum(v []byte) bool {
	checksum := [checksum_length]byte{}
	for i := range checksum {
		checksum[i] = v[len(v)-checksum_length+i]
	}
	real_checksum := sha256.Sum256(v[:len(v)-checksum_length])
	real_checksum = sha256.Sum256(real_checksum[:])
	for i, c := range checksum {
		if c != real_checksum[i] {
			return false
		}
	}
	return true
}
