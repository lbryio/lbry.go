package stake

import (
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/jsonpb"
)

func marshalToString(c *StakeHelper) (string, error) {
	m_pb := &jsonpb.Marshaler{}
	if c.IsSupport() {
		return m_pb.MarshalToString(c.Support)
	}
	return m_pb.MarshalToString(c.Claim)
}

func (c *StakeHelper) RenderJSON() (string, error) {
	r, err := marshalToString(c)
	if err != nil {
		fmt.Println("err")
		return "", err
	}
	var dat map[string]interface{}
	err = json.Unmarshal([]byte(r), &dat)
	if err != nil {
		return "", err
	}
	return r, nil
}

//TODO: encode byte arrays with b58 for addresses and b16 for source hashes instead of the default of b64
