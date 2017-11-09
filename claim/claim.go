package claim

import (
	"github.com/golang/protobuf/proto"
	"encoding/hex"
	"../pb"
	"errors"
)

type ClaimHelper struct {
	*pb.Claim
}

func (claim *ClaimHelper) LoadFromBytes(raw_claim []byte) (error) {
	if claim.String() != "" {
		return errors.New("already initialized")
	}
	if len(raw_claim) < 1 {
		return errors.New("there is nothing to decode")
	}

	claim_pb := &pb.Claim{}
	err := proto.Unmarshal(raw_claim, claim_pb)
	if err != nil {
		return err
	}
	*claim = ClaimHelper{claim_pb}

	return nil
}

func (claim *ClaimHelper) LoadFromHexString(claim_hex string) (error) {
	buf, err := hex.DecodeString(claim_hex)
	if err != nil {
		return err
	}
	return claim.LoadFromBytes(buf)
}

func DecodeClaimBytes(serialized []byte) (*ClaimHelper, error) {
	claim := &ClaimHelper{&pb.Claim{}}
	err := claim.LoadFromBytes(serialized)
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func DecodeClaimHex(serialized string) (*ClaimHelper, error) {
	claim_bytes, err := hex.DecodeString(serialized)
	if err != nil {
		return nil, err
	}
	return DecodeClaimBytes(claim_bytes)
}

func (m *ClaimHelper) GetStream() *pb.Stream {
	if m != nil {
		return m.Stream
	}
	return nil
}

func (m *ClaimHelper) GetCertificate() *pb.Certificate {
	if m != nil {
		return m.Certificate
	}
	return nil
}

func (m *ClaimHelper) GetPublisherSignature() *pb.Signature {
	if m != nil {
		return m.PublisherSignature
	}
	return nil
}
