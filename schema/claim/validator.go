package claim

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbryschema.go/address"
)

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

const SECP256k1 = "SECP256k1"

//const NIST256p = "NIST256p"
//const NIST384p = "NIST384p"

func getClaimSignatureDigest(bytes ...[]byte) [32]byte {

	var combined []byte
	for _, b := range bytes {
		combined = append(combined, b...)
	}
	digest := sha256.Sum256(combined)
	return [32]byte(digest)
}

func (c *ClaimHelper) VerifyDigest(certificate *ClaimHelper, signature [64]byte, digest [32]byte) bool {
	if certificate == nil {
		return false
	}

	R := &big.Int{}
	S := &big.Int{}
	R.SetBytes(signature[0:32])
	S.SetBytes(signature[32:64])
	pk, err := certificate.GetPublicKey()
	if err != nil {
		return false
	}
	return ecdsa.Verify(pk.ToECDSA(), digest[:], R, S)
}

func (c *ClaimHelper) ValidateClaimSignature(certificate *ClaimHelper, k string, certificateId string, blockchainName string) (bool, error) {
	if c.LegacyClaim != nil {
		return c.validateV1ClaimSignature(certificate, k, certificateId, blockchainName)
	}

	return c.validateClaimSignature(certificate, k, certificateId, blockchainName)
}

func (c *ClaimHelper) validateClaimSignature(certificate *ClaimHelper, firstInputTxHash, certificateId string, blockchainName string) (bool, error) {
	certificateIdSlice, err := hex.DecodeString(certificateId)
	if err != nil {
		return false, errors.Err(err)
	}
	certificateIdSlice = reverseBytes(certificateIdSlice)
	firstInputTxIDBytes, err := hex.DecodeString(firstInputTxHash)
	if err != nil {
		return false, errors.Err(err)
	}

	signature := c.Signature
	if signature == nil {
		return false, errors.Err("claim does not have a signature")
	}
	signatureBytes := [64]byte{}
	for i, b := range signature {
		signatureBytes[i] = b
	}

	claimDigest := getClaimSignatureDigest(firstInputTxIDBytes, certificateIdSlice, c.Payload)
	return c.VerifyDigest(certificate, signatureBytes, claimDigest), nil
}

func (c *ClaimHelper) validateV1ClaimSignature(certificate *ClaimHelper, claimAddy string, certificateId string, blockchainName string) (bool, error) {
	addressBytes, err := address.DecodeAddress(claimAddy, blockchainName)
	if err != nil {
		return false, err
	}
	//For V1 claim_id was incorrectly stored for claim signing.
	// So the bytes are not reversed like they are supposed to be (Endianess)
	certificateIdSlice, err := hex.DecodeString(certificateId)
	if err != nil {
		return false, err
	}

	signature := c.Signature
	if signature == nil {
		return false, errors.Err("claim does not have a signature")
	}
	signatureBytes := [64]byte{}
	for i := range signatureBytes {
		signatureBytes[i] = signature[i]
	}

	claimAddress, err := address.ValidateAddress(addressBytes, blockchainName)
	if err != nil {
		return false, errors.Err("invalid address")
	}

	serializedNoSig, err := c.serializedNoSignature()
	if err != nil {
		return false, errors.Err("serialization error")
	}

	claimDigest := getClaimSignatureDigest(claimAddress[:], serializedNoSig, certificateIdSlice)
	return c.VerifyDigest(certificate, signatureBytes, claimDigest), nil
}

func GetOutpointHash(txid string, vout uint32) (string, error) {
	txidBytes, err := hex.DecodeString(txid)
	if err != nil {
		return "", errors.Err(err)
	}
	var voutBytes = make([]byte, 4)
	binary.LittleEndian.PutUint32(voutBytes, vout)
	return hex.EncodeToString(append(reverseBytes(txidBytes), voutBytes...)), nil
}
