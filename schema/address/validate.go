package address

import (
	"github.com/cockroachdb/errors"
	"github.com/lbryio/lbry.go/v3/schema/address/base58"
)

const lbrycrdMainPubkeyPrefix = byte(85)
const lbrycrdMainScriptPrefix = byte(122)

const lbrycrdTestnetPubkeyPrefix = byte(111)
const lbrycrdTestnetScriptPrefix = byte(196)

const lbrycrdRegtestPubkeyPrefix = byte(111)
const lbrycrdRegtestScriptPrefix = byte(196)

const prefixLength = 1
const pubkeyLength = 20
const checksumLength = 4
const addressLength = prefixLength + pubkeyLength + checksumLength

const lbrycrdMain = "lbrycrd_main"
const lbrycrdTestnet = "lbrycrd_testnet"
const lbrycrdRegtest = "lbrycrd_regtest"

var addressPrefixes = map[string][2]byte{
	lbrycrdMain:    {lbrycrdMainPubkeyPrefix, lbrycrdMainScriptPrefix},
	lbrycrdTestnet: {lbrycrdTestnetPubkeyPrefix, lbrycrdTestnetScriptPrefix},
	lbrycrdRegtest: {lbrycrdRegtestPubkeyPrefix, lbrycrdRegtestScriptPrefix},
}

func PrefixIsValid(address [addressLength]byte, blockchainName string) bool {
	prefix := address[0]
	for _, addrPrefix := range addressPrefixes[blockchainName] {
		if addrPrefix == prefix {
			return true
		}
	}
	return false
}

func PubKeyIsValid(address [addressLength]byte) bool {
	pubkey := address[prefixLength : pubkeyLength+prefixLength]
	// TODO: validate this for real
	if len(pubkey) != pubkeyLength {
		return false
	}
	return true
}

func ChecksumIsValid(address [addressLength]byte) bool {
	return base58.VerifyBase58Checksum(address[:])
}

func ValidateAddress(address [addressLength]byte, blockchainName string) ([addressLength]byte, error) {
	if blockchainName != lbrycrdMain && blockchainName != lbrycrdTestnet && blockchainName != lbrycrdRegtest {
		return address, errors.WithStack(errors.New("invalid blockchain name"))
	}
	if !PrefixIsValid(address, blockchainName) {
		return address, errors.WithStack(errors.New("invalid prefix"))
	}
	if !PubKeyIsValid(address) {
		return address, errors.WithStack(errors.New("invalid pubkey"))
	}
	if !ChecksumIsValid(address) {
		return address, errors.WithStack(errors.New("invalid address checksum"))
	}
	return address, nil
}
