package claim

import (
	"../address"
	"crypto/sha256"
	"errors"
)

func GetClaimSignatureDigest(claim_address [25]byte, certificate_id [20]byte, serialized_no_sig []byte) [32]byte {
	combined := []byte{}
	for _, c := range claim_address {combined = append(combined, c)}
	for _, c := range serialized_no_sig {combined = append(combined, c)}
	for _, c := range certificate_id {combined = append(combined, c)}
	digest := sha256.Sum256(combined)
	return [32]byte(digest)
}

func VerifyDigest(signature [64]byte, digest [32]byte) bool {
	/*
    def verify_digest(self, signature, digest, sigdecode=sigdecode_string):
        if len(digest) > self.curve.baselen:
            raise BadDigestError("this curve (%s) is too short "
                                 "for your digest (%d)" % (self.curve.name,
                                                           8*len(digest)))
        number = string_to_number(digest)
        r, s = sigdecode(signature, self.pubkey.order)
        sig = ecdsa.Signature(r, s)
        if self.pubkey.verifies(number, sig):
            return True
        raise BadSignatureError
 	*/
	return true
}

func (claim *Claim) ValidateClaimSignature(claim_address [25]byte, certificate_id [20]byte, signature [64]byte) (bool, error) {
	claim_address, err := address.ValidateAddress(claim_address)
	if err != nil {return false, errors.New("invalid address")}
	serialized_no_sig, err := claim.SerializedNoSignature()
	if err != nil {return false, errors.New("serialization error")}
	claim_digest := GetClaimSignatureDigest(claim_address, certificate_id, serialized_no_sig)
	return VerifyDigest(signature, claim_digest), nil
}
