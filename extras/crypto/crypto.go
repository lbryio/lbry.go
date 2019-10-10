package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"sort"
	"strings"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

// RandString returns a random alphanumeric string of a given length
func RandString(length int) string {
	buf := make([]byte, length)
	_, err := rand.Reader.Read(buf)
	if err != nil {
		panic(errors.Err(err))
	}

	randStr := base58.Encode(buf)[:length]
	if len(randStr) < length {
		panic(errors.Err("Could not create random string that is long enough"))
	}

	return randStr
}

// Int returns a uniform random value in [0, max). It panics if max <= 0.
func RandInt64(max int64) int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		panic(err)
	}
	return n.Int64()
}

// HashStringSlice returns a hex hash of a slice of strings
func HashStringSlice(data []string) string {
	return hex.EncodeToString(hashStringSliceRaw(data))
}

func hashStringSliceRaw(data []string) []byte {
	sort.Strings(data)
	hash := sha3.Sum256([]byte(strings.Join(data, "")))
	return hash[:]
}
