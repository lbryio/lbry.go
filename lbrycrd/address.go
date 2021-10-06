package lbrycrd

import (
	"encoding/hex"

	"golang.org/x/crypto/ripemd160"

	"github.com/lbryio/lbcd/chaincfg"
	"github.com/lbryio/lbcutil"
	"github.com/lbryio/lbcutil/base58"

	"github.com/cockroachdb/errors"
)

// DecodeAddress decodes the string encoding of an address and returns
// the Address if addr is a valid encoding for a known address type.
//
// The bitcoin network the address is associated with is extracted if possible.
// When the address does not encode the network, such as in the case of a raw
// public key, the address will be associated with the passed defaultNet.
func DecodeAddress(addr string, defaultNet *chaincfg.Params) (lbcutil.Address, error) {
	// Serialized public keys are either 65 bytes (130 hex chars) if
	// uncompressed/hybrid or 33 bytes (66 hex chars) if compressed.
	if len(addr) == 130 || len(addr) == 66 {
		serializedPubKey, err := hex.DecodeString(addr)
		if err != nil {
			return nil, err
		}
		return lbcutil.NewAddressPubKey(serializedPubKey, defaultNet)
	}

	// Switch on decoded length to determine the type.
	decoded, netID, err := base58.CheckDecode(addr)
	if err != nil {
		if err == base58.ErrChecksum {
			return nil, lbcutil.ErrChecksumMismatch
		}
		return nil, errors.WithStack(errors.Newf("decoded address[%s] is of unknown format even with default chainparams[%s]", addr, defaultNet.Name))
	}

	switch len(decoded) {
	case ripemd160.Size: // P2PKH or P2SH
		isP2PKH := chaincfg.IsPubKeyHashAddrID(netID)
		isP2SH := chaincfg.IsScriptHashAddrID(netID)
		switch hash160 := decoded; {
		case isP2PKH && isP2SH:
			return nil, lbcutil.ErrAddressCollision
		case isP2PKH:
			return lbcutil.NewAddressPubKeyHash(hash160, &chaincfg.Params{PubKeyHashAddrID: netID})
		case isP2SH:
			return lbcutil.NewAddressScriptHashFromHash(hash160, &chaincfg.Params{ScriptHashAddrID: netID})
		default:
			return nil, lbcutil.ErrUnknownAddressType
		}

	default:
		return nil, errors.WithStack(errors.New("decoded address is of unknown size"))
	}
}
