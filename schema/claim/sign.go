package claim

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/v2/schema/address"

	"github.com/btcsuite/btcd/btcec"
)

func Sign(privKey btcec.PrivateKey, channel ClaimHelper, claim ClaimHelper, k string) (*Signature, error) {
	if channel.GetChannel() == nil {
		return nil, errors.Err("claim as channel is not of type channel")
	}
	if claim.LegacyClaim != nil {
		return claim.signV1(privKey, channel, k)
	}

	return claim.sign(privKey, channel, k)
}

func (c *ClaimHelper) sign(privKey btcec.PrivateKey, channel ClaimHelper, firstInputTxID string) (*Signature, error) {

	txidBytes, err := hex.DecodeString(firstInputTxID)
	if err != nil {
		return nil, errors.Err(err)
	}

	metadataBytes, err := c.serialized()
	if err != nil {
		return nil, errors.Err(err)
	}

	var digest []byte
	digest = append(digest, txidBytes...)
	digest = append(digest, c.ClaimID...)
	digest = append(digest, metadataBytes...)
	hash := sha256.Sum256(digest)
	hashBytes := make([]byte, len(hash))
	for i, b := range hash {
		hashBytes[i] = b
	}

	sig, err := privKey.Sign(hashBytes)
	if err != nil {
		return nil, errors.Err(err)
	}

	return &Signature{*sig}, nil

}

func (c *ClaimHelper) signV1(privKey btcec.PrivateKey, channel ClaimHelper, claimAddress string) (*Signature, error) {
	metadataBytes, err := c.serializedNoSignature()
	if err != nil {
		return nil, errors.Err(err)
	}

	addressBytes, err := address.DecodeAddress(claimAddress, "lbrycrd_main")
	if err != nil {
		return nil, errors.Prefix("V1 signing requires claim address and the decode failed with: ", err)
	}

	var digest []byte

	address := make([]byte, len(addressBytes))
	for i, b := range addressBytes {
		address[i] = b
	}

	digest = append(digest, address...)
	digest = append(digest, metadataBytes...)
	digest = append(digest, channel.ClaimID...)

	hash := sha256.Sum256(digest)
	hashBytes := make([]byte, len(hash))
	for i, b := range hash {
		hashBytes[i] = b
	}

	sig, err := privKey.Sign(hashBytes)
	if err != nil {
		return nil, errors.Err(err)
	}

	return &Signature{*sig}, nil
}

type Signature struct {
	btcec.Signature
}

func (s *Signature) LBRYSDKEncode() ([]byte, error) {
	if s.R == nil || s.S == nil {
		return nil, errors.Err("invalid signature, both S & R are nil")
	}
	rBytes := s.R.Bytes()
	sBytes := s.S.Bytes()

	return append(rBytes, sBytes...), nil
}

// rev reverses a byte slice. useful for switching endian-ness
func reverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for left, right := 0, len(b)-1; left < right; left, right = left+1, right-1 {
		r[left], r[right] = b[right], b[left]
	}
	return r
}
