package base58

import "crypto/sha256"

const checksumLength = 4

func VerifyBase58Checksum(v []byte) bool {
	checksum := [checksumLength]byte{}
	for i := range checksum {
		checksum[i] = v[len(v)-checksumLength+i]
	}
	real_checksum := sha256.Sum256(v[:len(v)-checksumLength])
	real_checksum = sha256.Sum256(real_checksum[:])
	for i, c := range checksum {
		if c != real_checksum[i] {
			return false
		}
	}
	return true
}
