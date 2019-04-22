package claim

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"gotest.tools/assert"
)

func TestSign(t *testing.T) {
	privateKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Error(err)
		return
	}
	channel := &ClaimHelper{newChannelClaim(), nil, nil, NoSig, nil}
	pubkeyBytes, err := PublicKeyToDER(privateKey.PubKey())
	if err != nil {
		t.Error(err)
		return
	}
	channel.GetChannel().PublicKey = pubkeyBytes
	claimID := "cf3f7c898af87cc69b06a6ac7899efb9a4878fdb"                      //Fake
	txid := "4c1df9e022e396859175f9bfa69b38e444db10fb53355fa99a0989a83bcdb82f" //Fake
	claimIDHexBytes, err := hex.DecodeString(claimID)
	if err != nil {
		t.Error(err)
		return
	}

	claim := &ClaimHelper{newStreamClaim(), nil, reverseBytes(claimIDHexBytes), WithSig, nil}
	claim.Claim.Title = "Test title"
	claim.Claim.Description = "Test description"
	sig, err := Sign(*privateKey, *channel, *claim, txid)
	if err != nil {
		t.Error(err)
		return
	}

	signatureBytes, err := sig.LBRYSDKEncode()
	if err != nil {
		t.Error(err)
		return
	}

	claim.Signature = signatureBytes

	rawChannel, err := channel.CompileValue()
	if err != nil {
		t.Error(err)
		return
	}
	rawClaim, err := claim.CompileValue()
	if err != nil {
		t.Error(err)
		return
	}

	channel, err = DecodeClaimBytes(rawChannel, "lbrycrd_main")
	if err != nil {
		t.Error(err)
		return
	}
	claim, err = DecodeClaimBytes(rawClaim, "lbrycrd_main")
	if err != nil {
		t.Error(err)
		return
	}

	valid, err := claim.ValidateClaimSignature(channel, txid, claimID, "lbrycrd_main")
	if err != nil {
		t.Error(err)
		return
	}

	assert.Assert(t, valid, "could not verify signature")

}

func TestSignWithV1Channel(t *testing.T) {
	cert_claim_hex := "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a03420004d015365a40f3e5c03c87227168e5851f44659837bcf6a3398ae633bc37d04ee19baeb26dc888003bd728146dbea39f5344bf8c52cedaf1a3a1623a0166f4a367"
	channel, err := DecodeClaimHex(cert_claim_hex, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}
	privateKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Error(err)
		return
	}
	pubkeyBytes, err := PublicKeyToDER(privateKey.PubKey())
	if err != nil {
		t.Error(err)
		return
	}
	channel.GetChannel().PublicKey = pubkeyBytes

	claimID := "251305ca93d4dbedb50dceb282ebcb7b07b7ac64"
	txid := "4c1df9e022e396859175f9bfa69b38e444db10fb53355fa99a0989a83bcdb82f" //Fake
	claimIDHexBytes, err := hex.DecodeString(claimID)
	if err != nil {
		t.Error(err)
		return
	}

	claim := &ClaimHelper{newStreamClaim(), nil, reverseBytes(claimIDHexBytes), WithSig, nil}
	claim.Claim.Title = "Test title"
	claim.Claim.Description = "Test description"
	sig, err := Sign(*privateKey, *channel, *claim, txid)
	if err != nil {
		t.Error(err)
		return
	}

	signatureBytes, err := sig.LBRYSDKEncode()
	if err != nil {
		t.Error(err)
		return
	}

	claim.Signature = signatureBytes
	compiledClaim, err := claim.CompileValue()
	if err != nil {
		t.Error(err)
	}
	claim, err = DecodeClaimBytes(compiledClaim, "lbrycrd_main")

	valid, err := claim.ValidateClaimSignature(channel, txid, claimID, "lbrycrd_main")
	if err != nil {
		t.Error(err)
		return
	}

	assert.Assert(t, valid, "could not verify signature")

}
