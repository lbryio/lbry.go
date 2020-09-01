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
	claim := &pb.Claim{
		Title:       c.GetTitle(),
		Description: c.GetDescription(),
		Thumbnail:   c.GetThumbnail(),
		Tags:        c.GetTags(),
		Languages:   c.GetLanguages(),
		Locations:   c.GetLocations(),
	}
	if c.GetChannel() != nil {
		claim.Type = &pb.Claim_Channel{Channel: c.GetChannel()}
	} else if c.GetStream() != nil {
		claim.Type = &pb.Claim_Stream{Stream: c.GetStream()}
	} else if c.GetCollection() != nil {
		claim.Type = &pb.Claim_Collection{Collection: c.GetCollection()}
	} else if c.GetRepost() != nil {
		claim.Type = &pb.Claim_Repost{Repost: c.GetRepost()}
	}

	return claim
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
