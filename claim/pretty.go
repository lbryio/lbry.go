package claim

import (
	"github.com/golang/protobuf/jsonpb"
)

func marshalToString(claim *ClaimHelper) (string, error) {
	m_pb := &jsonpb.Marshaler{}
	return m_pb.MarshalToString(claim)
}

func (claim *ClaimHelper) RenderJSON() (string, error) {
	return marshalToString(claim)
}

//TODO: encode byte arrays with b58 for addresses and b16 for source hashes instead of the default of b64