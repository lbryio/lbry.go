package lbrycrd

import (
	"encoding/hex"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
)

// DecodeAddress decodes the string encoding of an address and returns
// the Address if addr is a valid encoding for a known address type.
//
// The bitcoin network the address is associated with is extracted if possible.
// When the address does not encode the network, such as in the case of a raw
// public key, the address will be associated with the passed defaultNet.
func DecodeAddress(addr string, defaultNet *chaincfg.Params) (btcutil.Address, error) {
	// Serialized public keys are either 65 bytes (130 hex chars) if
	// uncompressed/hybrid or 33 bytes (66 hex chars) if compressed.
	if len(addr) == 130 || len(addr) == 66 {
		serializedPubKey, err := hex.DecodeString(addr)
		if err != nil {
			return nil, err
		}
		return btcutil.NewAddressPubKey(serializedPubKey, defaultNet)
	}

	// Switch on decoded length to determine the type.
	decoded, netID, err := base58.CheckDecode(addr)
	if err != nil {
		if err == base58.ErrChecksum {
			return nil, btcutil.ErrChecksumMismatch
		}
		return nil, errors.Err("decoded address[%s] is of unknown format even with default chainparams[%s]", addr, defaultNet.Name)
	}

	switch len(decoded) {
	case ripemd160.Size: // P2PKH or P2SH
		isP2PKH := chaincfg.IsPubKeyHashAddrID(netID)
		isP2SH := chaincfg.IsScriptHashAddrID(netID)
		switch hash160 := decoded; {
		case isP2PKH && isP2SH:
			return nil, btcutil.ErrAddressCollision
		case isP2PKH:
			return btcutil.NewAddressPubKeyHash(hash160, &chaincfg.Params{PubKeyHashAddrID: netID})
		case isP2SH:
			return btcutil.NewAddressScriptHashFromHash(hash160, &chaincfg.Params{ScriptHashAddrID: netID})
		default:
			return nil, btcutil.ErrUnknownAddressType
		}

	default:
		return nil, errors.Err("decoded address is of unknown size")
	}
}
