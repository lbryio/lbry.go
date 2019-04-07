package claim

import (
	"crypto/elliptic"
	"crypto/x509/pkix"
	"encoding/asn1"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/btcsuite/btcd/btcec"
)

func PublicKeyToDER(publicKey *btcec.PublicKey) ([]byte, error) {
	var publicKeyBytes []byte
	var publicKeyAlgorithm pkix.AlgorithmIdentifier
	var err error
	pub := publicKey.ToECDSA()
	publicKeyBytes = elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	//ans1 encoding oid for ecdsa public key https://github.com/golang/go/blob/release-branch.go1.12/src/crypto/x509/x509.go#L457
	publicKeyAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
	//asn1 encoding oid for secp256k1 https://github.com/bitpay/bitpay-go/blob/v2.2.2/key_utils/key_utils.go#L30
	paramBytes, err := asn1.Marshal(asn1.ObjectIdentifier{1, 3, 132, 0, 10})
	if err != nil {
		return nil, errors.Err(err)
	}
	publicKeyAlgorithm.Parameters.FullBytes = paramBytes

	return asn1.Marshal(publicKeyInfo{
		Algorithm: publicKeyAlgorithm,
		PublicKey: asn1.BitString{
			Bytes:     publicKeyBytes,
			BitLength: 8 * len(publicKeyBytes),
		},
	})

}

func (c *ClaimHelper) GetPublicKey() (*btcec.PublicKey, error) {
	if c.GetChannel() == nil {
		return nil, errors.Err("claim is not of type channel, so there is no public key to get")
	}
	return getPublicKeyFromBytes(c.GetChannel().PublicKey)
}

func getPublicKeyFromBytes(pubKeyBytes []byte) (*btcec.PublicKey, error) {
	PKInfo := publicKeyInfo{}
	asn1.Unmarshal(pubKeyBytes, &PKInfo)
	pubkeyBytes1 := []byte(PKInfo.PublicKey.Bytes)
	return btcec.ParsePubKey(pubkeyBytes1, btcec.S256())
}
