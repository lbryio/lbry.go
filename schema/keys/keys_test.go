package keys

import (
	"bytes"
	"encoding/hex"
	"encoding/pem"
	"testing"

	"github.com/lbryio/lbcd/btcec"
	"gotest.tools/assert"
)

// The purpose of this test, is to make sure the function converts btcec.PublicKey to DER format the same way
// lbry SDK does as this is the bytes that are put into protobuf and the same bytes are used for verify signatures.
// Making sure these
func TestPublicKeyToDER(t *testing.T) {
	publicKeyHex := "3056301006072a8648ce3d020106052b8104000a03420004d015365a40f3e5c03c87227168e5851f44659837bcf6a3398ae633bc37d04ee19baeb26dc888003bd728146dbea39f5344bf8c52cedaf1a3a1623a0166f4a367"
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		t.Error(err)
	}

	p1, err := GetPublicKeyFromBytes(pubKeyBytes)
	if err != nil {
		t.Error(err)
	}

	pubkeyBytes2, err := PublicKeyToDER(p1)
	if err != nil {
		t.Error(err)
	}
	for i, b := range pubKeyBytes {
		assert.Assert(t, b == pubkeyBytes2[i], "DER format in bytes must match!")
	}

	p2, err := GetPublicKeyFromBytes(pubkeyBytes2)
	if err != nil {
		t.Error(err)
	}
	assert.Assert(t, p1.IsEqual(p2), "The keys produced must be the same key!")
}

func TestPrivateKeyToDER(t *testing.T) {
	private1, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := PrivateKeyToDER(private1)
	if err != nil {
		t.Fatal(err)
	}
	private2, _, err := GetPrivateKeyFromBytes(bytes)
	if err != nil {
		t.Error(err)
	}
	if !private1.ToECDSA().Equal(private2.ToECDSA()) {
		t.Error("private keys dont match")
	}
}

func TestGetPrivateKeyFromBytes(t *testing.T) {
	private, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := PrivateKeyToDER(private)
	private2, _, err := GetPrivateKeyFromBytes(bytes)
	if !private.ToECDSA().Equal(private2.ToECDSA()) {
		t.Error("private keys dont match")
	}
}

func TestEncodePEMAndBack(t *testing.T) {
	private1, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Fatal(err)
	}
	b := bytes.NewBuffer(nil)
	derBytes, err := PrivateKeyToDER(private1)
	if err != nil {
		t.Fatal(err)
	}
	err = pem.Encode(b, &pem.Block{Type: "PRIVATE KEY", Bytes: derBytes})
	if err != nil {
		t.Fatal(err)
	}
	println(string(b.Bytes()))
	private2, _ := ExtractKeyFromPem(string(b.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !private1.ToECDSA().Equal(private2.ToECDSA()) {
		t.Error("private keys dont match")
	}
}
