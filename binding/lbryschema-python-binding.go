package main

import (
	"../address"
	"../claim"
	"C"
	"encoding/hex"
)

//export VerifySignature
func VerifySignature(claimHex string, certificateHex string, claimAddress string, certificateId string, blockchainName string) bool {
	decodedClaim, err := claim.DecodeClaimHex(claimHex, blockchainName)
	if err != nil {
		return false
	}
	decodedCertificate, err := claim.DecodeClaimHex(certificateHex, blockchainName)
	if err != nil {
		return false
	}
	result, err := decodedClaim.ValidateClaimSignature(decodedCertificate, claimAddress, certificateId, blockchainName)
	if err != nil {
		return false
	}
	return result
}

//export DecodeClaimHex
func DecodeClaimHex(claimHex string, blockchainName string) *C.char {
	decodedClaim, err := claim.DecodeClaimHex(claimHex, blockchainName)
	if err != nil {
		return C.CString("decode error: " + err.Error())
	}
	decoded, err := decodedClaim.RenderJSON()
	if err != nil {
		return C.CString("encode error: " + err.Error())
	}
	return C.CString(decoded)
}

//export SerializeClaimFromJSON
func SerializeClaimFromJSON(claimJSON string, blockchainName string) *C.char {
	decodedClaim, err := claim.DecodeClaimJSON(claimJSON, blockchainName)
	if err != nil {
		return C.CString("decode error: " + err.Error())
	}
	SerializedHex, err := decodedClaim.SerializedHexString()
	if err != nil {
		return C.CString("encode error: " + err.Error())
	}
	return C.CString(SerializedHex)
}

//export DecodeAddress
func DecodeAddress(addressString string, blockchainName string) *C.char {
	addressBytes, err := address.DecodeAddress(addressString, blockchainName)
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString(hex.EncodeToString(addressBytes[:]))
}

//export EncodeAddress
func EncodeAddress(addressChars string, blockchainName string) *C.char {
	addressBytes := [25]byte{}
	if len(addressChars) != 25 {
		return C.CString("error: address is not 25 bytes")
	}
	for i := range addressBytes {
		addressBytes[i] = byte(addressChars[i])
	}
	encodedAddress, err := address.EncodeAddress(addressBytes, blockchainName)
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString(encodedAddress)
}

func main() {}
