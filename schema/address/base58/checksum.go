package base58

import "crypto/sha256"

const checksumLength = 4

func VerifyBase58Checksum(v []byte) bool {
	checksum := [checksumLength]byte{}
	for i := range checksum {
		checksum[i] = v[len(v)-checksumLength+i]
	}
	realChecksum := sha256.Sum256(v[:len(v)-checksumLength])
	realChecksum = sha256.Sum256(realChecksum[:])
	for i, c := range checksum {
		if c != realChecksum[i] {
			return false
		}
	}
	return true
}
