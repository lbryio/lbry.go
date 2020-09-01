package claim

import (
	"testing"

	"gotest.tools/assert"
)

func TestV1ValidateClaimSignature(t *testing.T) {
	cert_claim_hex := "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a03420004d015365a40f3e5c03c87227168e5851f44659837bcf6a3398ae633bc37d04ee19baeb26dc888003bd728146dbea39f5344bf8c52cedaf1a3a1623a0166f4a367"
	signed_claim_hex := "080110011ad7010801128f01080410011a0c47616d65206f66206c696665221047616d65206f66206c696665206769662a0b4a6f686e20436f6e776179322e437265617469766520436f6d6d6f6e73204174747269627574696f6e20342e3020496e7465726e6174696f6e616c38004224080110011a195569c917f18bf5d2d67f1346aa467b218ba90cdbf2795676da250000803f4a0052005a001a41080110011a30b6adf6e2a62950407ea9fb045a96127b67d39088678d2f738c359894c88d95698075ee6203533d3c204330713aa7acaf2209696d6167652f6769662a5c080110031a40c73fe1be4f1743c2996102eec6ce0509e03744ab940c97d19ddb3b25596206367ab1a3d2583b16c04d2717eeb983ae8f84fee2a46621ffa5c4726b30174c6ff82214251305ca93d4dbedb50dceb282ebcb7b07b7ac65"

	signed_claim, err := DecodeClaimHex(signed_claim_hex, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}
	cert_claim, err := DecodeClaimHex(cert_claim_hex, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}

	claim_addr := "bSkUov7HMWpYBiXackDwRnR5ishhGHvtJt"
	cert_id := "251305ca93d4dbedb50dceb282ebcb7b07b7ac65"

	result, err := signed_claim.ValidateClaimSignature(cert_claim, claim_addr, cert_id, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}
	if result != true {
		t.Error("failed to validate signature:", result)
	}
}

func TestV1FailToValidateClaimSignature(t *testing.T) {
	cert_claim_hex := "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a03420004d015365a40f3e5c03c87227168e5851f44659837bcf6a3398ae633bc37d04ee19baeb26dc888003bd728146dbea39f5344bf8c52cedaf1a3a1623a0166f4a367"
	signed_claim_hex := "080110011ad7010801128f01080410011a0c47616d65206f66206c696665221047616d65206f66206c696665206769662a0b4a6f686e20436f6e776179322e437265617469766520436f6d6d6f6e73204174747269627574696f6e20342e3020496e7465726e6174696f6e616c38004224080110011a195569c917f18bf5d2d67f1346aa467b218ba90cdbf2795676da250000803f4a0052005a001a41080110011a30b6adf6e2a62950407ea9fb045a96127b67d39088678d2f738c359894c88d95698075ee6203533d3c204330713aa7acaf2209696d6167652f6769662a5c080110031a40c73fe1be4f1743c2996102eec6ce0509e03744ab940c97d19ddb3b25596206367ab1a3d2583b16c04d2717eeb983ae8f84fee2a46621ffa5c4726b30174c6ff82214251305ca93d4dbedb50dceb282ebcb7b07b7ac65"

	signed_claim, err := DecodeClaimHex(signed_claim_hex, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}
	cert_claim, err := DecodeClaimHex(cert_claim_hex, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}

	claim_addr := "bSkUov7HMWpYBiXackDwRnR5ishhGHvtJt"
	cert_id := "251305ca93d4dbedb50dceb282ebcb7b07b7ac64"

	result, err := signed_claim.ValidateClaimSignature(cert_claim, claim_addr, cert_id, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}
	if result != false {
		t.Error("failed to validate signature:", result)
	}
}

func TestV2ValidateClaimSignature(t *testing.T) {
	cert_claim_hex := "00125a0a583056301006072a8648ce3d020106052b8104000a034200045a0343c155302280da01ae0001b7295241eb03c42a837acf92ccb9680892f7db50fd1d3c14b28bb594e304f05fc4ae7c1f222a85d1d1a3461b3cfb9906f66cb5"
	signed_claim_hex := "015cb78e424a34fbf79b67f9107430427aa62373e69b4998a29ecec8f14a9e0a213a043ced8064c069d7e464b5fd3ccb92b45bd59b15c0e1bb27e3c366d43f86a9a6b5ad42647a1aad69a73ac50b19ae3ec978c2c70aa2010a99010a301c662f19abc461e7eddecf165adfa7fca569e209773f3db31241c1e297f0a8d5b3e4768828b065fbeb1d6776f61073f6121b3031202d20556e6d6173746572656420496d70756c7365732e377a187a22146170706c69636174696f6e2f782d6578742d377a32302eb61ea475017e28c013616a56c1219ba90dc35fffff453d9675146f648f66634e0d1516528d37aba9f5801229d9f2181a044e6f6e6542087465737420707562520062020801"

	signed_claim, err := DecodeClaimHex(signed_claim_hex, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}
	cert_claim, err := DecodeClaimHex(cert_claim_hex, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}

	firstInputTxHash, err := GetOutpointHash("becb96a4a2e66bd24f083772fe9da904654ea9b5f07cc5bfbee233355911ddb1", uint32(0))
	if err != nil {
		t.Error(err)
	}
	cert_id := "e67323a67a42307410f9679bf7fb344a428eb75c"

	result, err := signed_claim.ValidateClaimSignature(cert_claim, firstInputTxHash, cert_id, "lbrycrd_main")
	if err != nil {
		t.Error(err)
	}
	if result != true {
		t.Error("failed to validate signature:", result)
	}

}

func TestGetOutpointHash(t *testing.T) {
	hash, err := GetOutpointHash("dc3dcf2f94d3c91e454ac2474802e20f26b30705372dda43890c811d918aef64", 1)
	if err != nil {
		t.Error(err)
	}
	assert.Assert(t, hash == "64ef8a911d810c8943da2d370507b3260fe2024847c24a451ec9d3942fcf3ddc01000000", uint(1))
}
