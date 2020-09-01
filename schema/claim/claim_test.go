package claim

import "testing"

func TestClaimHelper(t *testing.T) {
	for _, rawClaim := range raw_claims {
		helper, err := DecodeClaimHex(rawClaim, "lbrycrd_main")
		if err != nil {
			t.Error(err)
		}
		_, err = helper.RenderJSON()
		if err != nil {
			t.Error(err)
		}

		_, err = helper.serialized()
		if err != nil {
			t.Error(err)
		}
		_, err = helper.serializedHexString()
		if err != nil {
			t.Error(err)
		}
		_, err = helper.serializedNoSignature()
		if err != nil {
			t.Error(err)
		}
		err = helper.ValidateAddresses("lbrycrd_main")
		if err != nil {
			t.Error(err)
		}
	}
}
