package claim

import (
	"../pb"
	"errors"
	"github.com/golang/protobuf/proto"
	"encoding/hex"
)

func (claim *Claim) Serialized() ([]byte, error) {
	serialized := claim.protobuf.String()
	if serialized == "" {
		return nil, errors.New("not initialized")
	}
	return proto.Marshal(&claim.protobuf)
}

func (claim *Claim) SerializedHexString() (string, error) {
	serialized, err := claim.Serialized()
	if err != nil {
		return "", err
	}
	serialized_hex := hex.EncodeToString(serialized)
	return serialized_hex, nil
}

func (claim *Claim) SerializedNoSignature() ([]byte, error) {
	if claim.protobuf.String() == "" {
		return nil, errors.New("not initialized")
	}
	if claim.protobuf.GetPublisherSignature() == nil {
		serialized, err := claim.Serialized()
		if err != nil {
			return nil, err
		}
		return serialized, nil
	} else {
		clone := &pb.Claim{}
		proto.Merge(clone, &claim.protobuf)
		proto.ClearAllExtensions(clone.PublisherSignature)
		clone.PublisherSignature = nil
		return proto.Marshal(clone)
	}
}
