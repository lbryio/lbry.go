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

func Normalize(s string) string {
	c := cases.Fold()
	return c.String(norm.NFD.String(s))
}

func ReverseBytes(s []byte) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// convert txid to txHash
func ToHash(txid string) []byte {
	t, err := hex.DecodeString(txid)
	if err != nil {
		return nil
	}

	// reverse the bytes. thanks, Satoshi 😒
	for i, j := 0, len(t)-1; i < j; i, j = i+1, j-1 {
		t[i], t[j] = t[j], t[i]
	}

	return t
}

// convert txHash to txid
func FromHash(txHash []byte) string {
	t := make([]byte, len(txHash))
	copy(t, txHash)

	// reverse the bytes. thanks, Satoshi 😒
	for i, j := 0, len(txHash)-1; i < j; i, j = i+1, j-1 {
		txHash[i], txHash[j] = txHash[j], txHash[i]
	}

	return hex.EncodeToString(t)

}
