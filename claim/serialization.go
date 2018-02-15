package claim

import (
	"github.com/lbryio/lbryschema.go/pb"
	"encoding/hex"
	"errors"
	"github.com/golang/protobuf/proto"
)

func (c *ClaimHelper) Serialized() ([]byte, error) {
	serialized := c.String()
	if serialized == "" {
		return nil, errors.New("not initialized")
	}
	v := c.GetVersion()
	t := c.GetClaimType()

	return proto.Marshal(
		&pb.Claim{
			Version:            &v,
			ClaimType:          &t,
			Stream:             c.GetStream(),
			Certificate:        c.GetCertificate(),
			PublisherSignature: c.GetPublisherSignature()})
}

func (c *ClaimHelper) GetProtobuf() *pb.Claim {
	v := c.GetVersion()
	t := c.GetClaimType()

	return &pb.Claim{
		Version:            &v,
		ClaimType:          &t,
		Stream:             c.GetStream(),
		Certificate:        c.GetCertificate(),
		PublisherSignature: c.GetPublisherSignature()}
}

func (c *ClaimHelper) SerializedHexString() (string, error) {
	serialized, err := c.Serialized()
	if err != nil {
		return "", err
	}
	serialized_hex := hex.EncodeToString(serialized)
	return serialized_hex, nil
}

func (c *ClaimHelper) SerializedNoSignature() ([]byte, error) {
	if c.String() == "" {
		return nil, errors.New("not initialized")
	}
	if c.GetPublisherSignature() == nil {
		serialized, err := c.Serialized()
		if err != nil {
			return nil, err
		}
		return serialized, nil
	} else {
		clone := &pb.Claim{}
		proto.Merge(clone, c.GetProtobuf())
		proto.ClearAllExtensions(clone.PublisherSignature)
		clone.PublisherSignature = nil
		return proto.Marshal(clone)
	}
}
