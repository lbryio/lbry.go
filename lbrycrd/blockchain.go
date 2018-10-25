package lbrycrd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"golang.org/x/crypto/ripemd160"
)

// rev reverses a byte slice. useful for switching endian-ness
func rev(b []byte) []byte {
	r := make([]byte, len(b))
	for left, right := 0, len(b)-1; left < right; left, right = left+1, right-1 {
		r[left], r[right] = b[right], b[left]
	}
	return r
}

func ClaimIDFromOutpoint(txid string, nout int) (string, error) {
	// convert transaction id to byte array
	txidBytes, err := hex.DecodeString(txid)
	if err != nil {
		return "", err
	}

	// reverse (make big-endian)
	txidBytes = rev(txidBytes)

	// append nout
	noutBytes := make([]byte, 4) // num bytes in uint32
	binary.BigEndian.PutUint32(noutBytes, uint32(nout))
	txidBytes = append(txidBytes, noutBytes...)

	// sha256 it
	s := sha256.New()
	s.Write(txidBytes)

	// ripemd it
	r := ripemd160.New()
	r.Write(s.Sum(nil))

	// reverse (make little-endian)
	res := rev(r.Sum(nil))

	return hex.EncodeToString(res), nil
}
