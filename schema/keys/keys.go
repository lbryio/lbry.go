package keys

import (
	"crypto/elliptic"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/lbryio/lbcd/btcec"
)

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
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

// This type provides compatibility with the btcec package
type ecPrivateKey struct {
	Version       int
	PrivateKey    []byte
	NamedCurveOID asn1.ObjectIdentifier `asn1:"optional,explicit,tag:0"`
	PublicKey     asn1.BitString        `asn1:"optional,explicit,tag:1"`
}

func PrivateKeyToDER(key *btcec.PrivateKey) ([]byte, error) {
	privateKey := make([]byte, (key.Curve.Params().N.BitLen()+7)/8)
	oid := asn1.ObjectIdentifier{1, 3, 132, 0, 10}
	return asn1.Marshal(ecPrivateKey{
		Version:       1,
		PrivateKey:    key.D.FillBytes(privateKey),
		NamedCurveOID: oid,
		PublicKey:     asn1.BitString{Bytes: elliptic.Marshal(key.Curve, key.X, key.Y)},
	})
}

func GetPublicKeyFromBytes(pubKeyBytes []byte) (*btcec.PublicKey, error) {
	if len(pubKeyBytes) == 33 {
		return btcec.ParsePubKey(pubKeyBytes, btcec.S256())
	}
	PKInfo := publicKeyInfo{}
	_, err := asn1.Unmarshal(pubKeyBytes, &PKInfo)
	if err != nil {
		return nil, errors.Err(err)
	}
	pubkeyBytes1 := PKInfo.PublicKey.Bytes
	return btcec.ParsePubKey(pubkeyBytes1, btcec.S256())
}

func GetPrivateKeyFromBytes(privKeyBytes []byte) (*btcec.PrivateKey, *btcec.PublicKey, error) {
	ecPK := ecPrivateKey{}
	_, err := asn1.Unmarshal(privKeyBytes, &ecPK)
	if err != nil {
		return nil, nil, errors.Err(err)
	}
	priv, publ := btcec.PrivKeyFromBytes(btcec.S256(), ecPK.PrivateKey)
	return priv, publ, nil
}

// Returns a btec.Private key object if provided a correct secp256k1 encoded pem.
func ExtractKeyFromPem(pm string) (*btcec.PrivateKey, *btcec.PublicKey) {
	byta := []byte(pm)
	blck, _ := pem.Decode(byta)
	var ecp ecPrivateKey
	asn1.Unmarshal(blck.Bytes, &ecp)
	return btcec.PrivKeyFromBytes(btcec.S256(), ecp.PrivateKey)
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
