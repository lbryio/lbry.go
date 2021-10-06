package stake

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/cockroachdb/errors"
	"github.com/lbryio/lbry.go/v3/schema/address"
	"github.com/lbryio/lbry.go/v3/schema/keys"

	"github.com/lbryio/lbcd/btcec"
)

func Sign(privKey btcec.PrivateKey, channel Helper, claim Helper, k string) (*keys.Signature, error) {
	if channel.Claim.GetChannel() == nil {
		return nil, errors.WithStack(errors.New("claim as channel is not of type channel"))
	}
	if claim.LegacyClaim != nil {
		return claim.signV1(privKey, channel, k)
	}

	return claim.sign(privKey, channel, k)
}

func (c *Helper) sign(privKey btcec.PrivateKey, channel Helper, firstInputTxID string) (*keys.Signature, error) {

	txidBytes, err := hex.DecodeString(firstInputTxID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	metadataBytes, err := c.serialized()
	if err != nil {
		return nil, err
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
		return nil, errors.WithStack(err)
	}

	return &keys.Signature{*sig}, nil

}

func (c *Helper) signV1(privKey btcec.PrivateKey, channel Helper, claimAddress string) (*keys.Signature, error) {
	metadataBytes, err := c.serializedNoSignature()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	addressBytes, err := address.DecodeAddress(claimAddress, "lbrycrd_main")
	if err != nil {
		return nil, errors.WithMessage(err, "V1 signing requires claim address and the decode failed")
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
		return nil, errors.WithStack(err)
	}

	return &keys.Signature{Signature: *sig}, nil
}

// rev reverses a byte slice. useful for switching endian-ness
func reverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for left, right := 0, len(b)-1; left < right; left, right = left+1, right-1 {
		r[left], r[right] = b[right], b[left]
	}
	return r
}
