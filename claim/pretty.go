package claim

import (
	"github.com/golang/protobuf/jsonpb"
)

func (claim *Claim) RenderJSON() (string, error) {
	marshaler := jsonpb.Marshaler{}
	return marshaler.MarshalToString(&claim.protobuf)
}
