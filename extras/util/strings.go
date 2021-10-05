package util

import (
	"encoding/hex"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
	"strings"
)

func StringSplitArg(stringToSplit, separator string) []interface{} {
	split := strings.Split(stringToSplit, separator)
	splitInterface := make([]interface{}, len(split))
	for i, s := range split {
		splitInterface[i] = s
	}
	return splitInterface
}

// NormalizeName Normalize names to remove weird characters and account to capitalization
func NormalizeName(s string) string {
	c := cases.Fold()
	return c.String(norm.NFD.String(s))
}

// ReverseBytesInPlace reverse the bytes. thanks, Satoshi 😒
func ReverseBytesInPlace(s []byte) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// TxIdToTxHash convert the txid to a hash for returning from the hub
func TxIdToTxHash(txid string) []byte {
	t, err := hex.DecodeString(txid)
	if err != nil {
		return nil
	}

	ReverseBytesInPlace(t)

	return t
}

// TxHashToTxId convert the txHash from the response format back to an id
func TxHashToTxId(txHash []byte) string {
	t := make([]byte, len(txHash))
	copy(t, txHash)

	ReverseBytesInPlace(t)

	return hex.EncodeToString(t)

}
