package claim

import (
	"encoding/hex"

	"github.com/lbryio/lbry.go/extras/errors"
	legacy "github.com/lbryio/types/v1/go"
	pb "github.com/lbryio/types/v2/go"

	"github.com/golang/protobuf/proto"
)

func (c *ClaimHelper) serialized() ([]byte, error) {
	serialized := c.String()
	if serialized == "" {
		return nil, errors.Err("not initialized")
	}

	if c.LegacyClaim != nil {
		return proto.Marshal(c.getLegacyProtobuf())
	}

	return proto.Marshal(c.getProtobuf())
}

func (c *ClaimHelper) getProtobuf() *pb.Claim {
	if c.GetChannel() != nil {
		return &pb.Claim{Type: &pb.Claim_Channel{Channel: c.GetChannel()}}
	} else if c.GetStream() != nil {
		return &pb.Claim{Type: &pb.Claim_Stream{Stream: c.GetStream()}}
	}

	return nil
}

func (c *ClaimHelper) getLegacyProtobuf() *legacy.Claim {
	v := c.LegacyClaim.GetVersion()
	t := c.LegacyClaim.GetClaimType()
	return &legacy.Claim{
		Version:            &v,
		ClaimType:          &t,
		Stream:             c.LegacyClaim.GetStream(),
		Certificate:        c.LegacyClaim.GetCertificate(),
		PublisherSignature: c.LegacyClaim.GetPublisherSignature()}
}

func (c *ClaimHelper) serializedHexString() (string, error) {
	serialized, err := c.serialized()
	if err != nil {
		return "", err
	}
	serialized_hex := hex.EncodeToString(serialized)
	return serialized_hex, nil
}

func (c *ClaimHelper) serializedNoSignature() ([]byte, error) {
	if c.String() == "" {
		return nil, errors.Err("not initialized")
	}
	if c.Signature == nil {
		serialized, err := c.serialized()
		if err != nil {
			return nil, err
		}
		return serialized, nil
	} else {
		if c.LegacyClaim != nil {
			clone := &legacy.Claim{}
			proto.Merge(clone, c.getLegacyProtobuf())
			proto.ClearAllExtensions(clone.PublisherSignature)
			clone.PublisherSignature = nil
			return proto.Marshal(clone)
		}
		clone := &pb.Claim{}
		proto.Merge(clone, c.getProtobuf())
		return proto.Marshal(clone)
	}
}
