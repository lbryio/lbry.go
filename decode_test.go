package lbryschema

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"encoding/hex"
	"testing"
	"./pb"
)

func TestDecodeCertificate(t *testing.T) {
	claim_hex := "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a03420004d015365a40f3e5c03c87227168e5851f44659837bcf6a3398ae633bc37d04ee19baeb26dc888003bd728146dbea39f5344bf8c52cedaf1a3a1623a0166f4a367"
	buf, _ := hex.DecodeString(claim_hex)
	testClaim := &pb.Claim{}
	proto.Unmarshal(buf, testClaim)
	fmt.Println( testClaim)
}

//func TestDecodeAddress(t *testing.T) {
//	DecodeAddress("bUc9gyCJPKu2CBYpTvJ98MdmsLb68utjP6")
//}
