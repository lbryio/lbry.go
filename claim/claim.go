package claim

import (
	"github.com/golang/protobuf/proto"
	"encoding/hex"
	"../pb"
	"errors"
)

type Claim struct {
	protobuf pb.Claim
}

func (claim *Claim) LoadFromBytes(raw_claim []byte) (error) {
	if claim.protobuf.String() != "" {
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
	claim.protobuf = *claim_pb
	return nil
}

func (claim *Claim) LoadFromHexString(claim_hex string) (error) {
	buf, err := hex.DecodeString(claim_hex)
	if err != nil {
		return err
	}
	return claim.LoadFromBytes(buf)
}

func DecodeClaimBytes(serialized []byte) (*Claim, error) {
	claim := &Claim{}
	err := claim.LoadFromBytes(serialized)
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func DecodeClaimHex(serialized string) (*Claim, error) {
	claim_bytes, err := hex.DecodeString(serialized)
	if err != nil {
		return nil, err
	}
	return DecodeClaimBytes(claim_bytes)
}
