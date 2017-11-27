package claim

import (
	"../address"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"math/big"
)

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

const SECP256k1 = "SECP256k1"

func GetClaimSignatureDigest(claimAddress [25]byte, certificateId [20]byte, serializedNoSig []byte) [32]byte {
	var combined []byte
	for _, c := range claimAddress {
		combined = append(combined, c)
	}
	for _, c := range serializedNoSig {
		combined = append(combined, c)
	}
	for _, c := range certificateId {
		combined = append(combined, c)
	}
	digest := sha256.Sum256(combined)
	return [32]byte(digest)
}

func (c *ClaimHelper) GetCertificatePublicKey() (*btcec.PublicKey, error) {
	derBytes := c.GetCertificate().GetPublicKey()
	pub := publicKeyInfo{}
	asn1.Unmarshal(derBytes, &pub)
	pubkey_bytes := []byte(pub.PublicKey.Bytes)
	p, err := btcec.ParsePubKey(pubkey_bytes, btcec.S256())
	if err != nil {
		return &btcec.PublicKey{}, err
	}
	return p, err
}

func (c *ClaimHelper) VerifyDigest(certificate *ClaimHelper, signature [64]byte, digest [32]byte) bool {
	public_key, err := certificate.GetCertificatePublicKey()
	if err != nil {
		return false
	}

	if c.PublisherSignature.SignatureType.String() == SECP256k1 {
		R := &big.Int{}
		S := &big.Int{}
		R.SetBytes(signature[0:32])
		S.SetBytes(signature[32:64])
		return ecdsa.Verify(public_key.ToECDSA(), digest[:], R, S)
	}
	return false
}

func (c *ClaimHelper) ValidateClaimSignatureBytes(certificate *ClaimHelper, claimAddress [25]byte, certificateId [20]byte, blockchainName string) (bool, error) {
	signature := c.GetPublisherSignature()
	if signature == nil {
		return false, errors.New("claim does not have a signature")
	}
	signatureSlice := signature.GetSignature()
	signatureBytes := [64]byte{}
	for i := range signatureBytes {
		signatureBytes[i] = signatureSlice[i]
	}

	claimAddress, err := address.ValidateAddress(claimAddress, blockchainName)
	if err != nil {
		return false, errors.New("invalid address")
	}

	serializedNoSig, err := c.SerializedNoSignature()
	if err != nil {
		return false, errors.New("serialization error")
	}

	claimDigest := GetClaimSignatureDigest(claimAddress, certificateId, serializedNoSig)
	return c.VerifyDigest(certificate, signatureBytes, claimDigest), nil
}

func (c *ClaimHelper) ValidateClaimSignature(certificate *ClaimHelper, claimAddress string, certificateId string, blockchainName string) (bool, error) {
	addressBytes, err := address.DecodeAddress(claimAddress, blockchainName)
	if err != nil {
		return false, err
	}
	certificateIdSlice, err := hex.DecodeString(certificateId)
	if err != nil {
		return false, err
	}
	certificateIdBytes := [20]byte{}
	for i := range certificateIdBytes {
		certificateIdBytes[i] = certificateIdSlice[i]
	}
	return c.ValidateClaimSignatureBytes(certificate, addressBytes, certificateIdBytes, blockchainName)
}
