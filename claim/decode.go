package claim

import (
	"bytes"

	types "github.com/lbryio/types/v2/go"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func ToJSON(value []byte) (string, error) {
	c := &types.Claim{}
	err := proto.Unmarshal(value, c)
	if err != nil {
		return "", err
	}

	b := bytes.NewBuffer(nil)
	m := protojson.MarshalOptions{Indent: "  "}
	b, err = m.Marshal(c)

	return b.String(), err
}
