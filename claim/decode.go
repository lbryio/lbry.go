package claim

import (
	"bytes"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	types "github.com/lbryio/types/go"
)

func ToJSON(value []byte) (string, error) {
	c := &types.Claim{}
	err := proto.Unmarshal(value, c)
	if err != nil {
		return "", err
	}

	b := bytes.NewBuffer(nil)
	m := jsonpb.Marshaler{Indent: "  "}
	err = m.Marshal(b, c)

	return b.String(), err
}
