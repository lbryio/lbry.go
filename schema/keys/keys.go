package keys

import (
	"crypto/elliptic"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/btcsuite/btcd/btcec"
)

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

//This type provides compatibility with the btcec package
type ecPrivateKey struct {
	Version       int
	PrivateKey    []byte
	NamedCurveOID asn1.ObjectIdentifier `asn1:"optional,explicit,tag:0"`
	PublicKey     asn1.BitString        `asn1:"optional,explicit,tag:1"`
}

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

func GetPublicKeyFromBytes(pubKeyBytes []byte) (*btcec.PublicKey, error) {
	PKInfo := publicKeyInfo{}
	asn1.Unmarshal(pubKeyBytes, &PKInfo)
	pubkeyBytes1 := []byte(PKInfo.PublicKey.Bytes)
	return btcec.ParsePubKey(pubkeyBytes1, btcec.S256())
}

//Returns a btec.Private key object if provided a correct secp256k1 encoded pem.
func ExtractKeyFromPem(pm string) (*btcec.PrivateKey, *btcec.PublicKey) {
	byta := []byte(pm)
	blck, _ := pem.Decode(byta)
	var ecp ecPrivateKey
	asn1.Unmarshal(blck.Bytes, &ecp)
	return btcec.PrivKeyFromBytes(btcec.S256(), ecp.PrivateKey)
}
