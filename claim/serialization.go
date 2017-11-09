package claim

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"encoding/hex"
	"../pb"
)

func (claim *ClaimHelper) Serialized() ([]byte, error) {
	serialized := claim.String()
	if serialized == "" {
		return nil, errors.New("not initialized")
	}
	v := claim.GetVersion()
	t := claim.GetClaimType()

	return proto.Marshal(
		&pb.Claim{
			Version: &v,
			ClaimType: &t,
			Stream: claim.GetStream(),
			Certificate: claim.GetCertificate(),
			PublisherSignature: claim.GetPublisherSignature()})
}

func (claim *ClaimHelper) GetProtobuf() (*pb.Claim) {
	v := claim.GetVersion()
	t := claim.GetClaimType()

	return &pb.Claim{
		Version: &v,
		ClaimType: &t,
		Stream: claim.GetStream(),
		Certificate: claim.GetCertificate(),
		PublisherSignature: claim.GetPublisherSignature()}
}

func (claim *ClaimHelper) SerializedHexString() (string, error) {
	serialized, err := claim.Serialized()
	if err != nil {
		return "", err
	}
	serialized_hex := hex.EncodeToString(serialized)
	return serialized_hex, nil
}

func (claim *ClaimHelper) SerializedNoSignature() ([]byte, error) {
	if claim.String() == "" {
		return nil, errors.New("not initialized")
	}
	if claim.GetPublisherSignature() == nil {
		serialized, err := claim.Serialized()
		if err != nil {
			return nil, err
		}
		return serialized, nil
	} else {
		clone := &pb.Claim{}
		proto.Merge(clone, claim.GetProtobuf())
		proto.ClearAllExtensions(clone.PublisherSignature)
		clone.PublisherSignature = nil
		return proto.Marshal(clone)
	}
}
