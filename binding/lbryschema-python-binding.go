package main
import (
	"C"
	"../claim"
)

//export VerifySignature
func VerifySignature(claimHex string, certificateHex string, claimAddress string, certificateId string) (bool) {
	decodedClaim, err := claim.DecodeClaimHex(claimHex)
	if err != nil {
		return false
	}
	decodedCertificate, err := claim.DecodeClaimHex(certificateHex)
	result, err := decodedClaim.ValidateClaimSignature(decodedCertificate, claimAddress, certificateId)
	if err != nil {
		return false
	}
	return result
}

func main() {}
