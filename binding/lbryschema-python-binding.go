package main
import (
	"C"
	"../claim"
)

//export VerifySignature
func VerifySignature(claimHex string, certificateHex string, claimAddress string, certificateId string) bool {
	decodedClaim, err := claim.DecodeClaimHex(claimHex)
	if err != nil {
		return false
	}
	decodedCertificate, err := claim.DecodeClaimHex(certificateHex)
	if err != nil {
		return false
	}
	result, err := decodedClaim.ValidateClaimSignature(decodedCertificate, claimAddress, certificateId)
	if err != nil {
		return false
	}
	if result == false {
		return false
	}
	return true
}

//export DecodeClaimHex
func DecodeClaimHex(claimHex string) *C.char {
	decodedClaim, err := claim.DecodeClaimHex(claimHex)
	if err != nil {
		return C.CString("decode error")
	}
	decoded, err := decodedClaim.RenderJSON()
	if err != nil {
		return C.CString("encode error")
	}
	return C.CString(decoded)
}

func main() {}
