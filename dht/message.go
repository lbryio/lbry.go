package dht

import (
	"github.com/lbryio/errors.go"

	"github.com/spf13/cast"
	"github.com/zeebo/bencode"
)

const (
	pingMethod      = "ping"
	storeMethod     = "store"
	findNodeMethod  = "findNode"
	findValueMethod = "findValue"
)

const (
	pingSuccessResponse  = "pong"
	storeSuccessResponse = "OK"
)

const (
	requestType  = 0
	responseType = 1
	errorType    = 2
)

const (
	// these are strings because bencode requires bytestring keys
	headerTypeField      = "0"
	headerMessageIDField = "1" // message id is 20 bytes long
	headerNodeIDField    = "2" // node id is 48 bytes long
	headerPayloadField   = "3"
	headerArgsField      = "4"
)

type Message interface {
	bencode.Marshaler
}

type Request struct {
	ID        string
	NodeID    string
	Method    string
	Args      []string
	StoreArgs *storeArgs
}

func (r Request) MarshalBencode() ([]byte, error) {
	var args interface{}
	if r.StoreArgs != nil {
		args = r.StoreArgs
	} else {
		args = r.Args
	}
	return bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      requestType,
		headerMessageIDField: r.ID,
		headerNodeIDField:    r.NodeID,
		headerPayloadField:   r.Method,
		headerArgsField:      args,
	})
}

func (r *Request) UnmarshalBencode(b []byte) error {
	var raw struct {
		ID     string             `bencode:"1"`
		NodeID string             `bencode:"2"`
		Method string             `bencode:"3"`
		Args   bencode.RawMessage `bencode:"4"`
	}
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return errors.Prefix("request unmarshal", err)
	}

	r.ID = raw.ID
	r.NodeID = raw.NodeID
	r.Method = raw.Method

	if r.Method == storeMethod {
		r.StoreArgs = &storeArgs{} // bencode wont find the unmarshaler on a null pointer. need to fix it.
		err = bencode.DecodeBytes(raw.Args, &r.StoreArgs)
	} else {
		err = bencode.DecodeBytes(raw.Args, &r.Args)
	}
	if err != nil {
		return errors.Prefix("request unmarshal", err)
	}

	return nil
}

type storeArgs struct {
	BlobHash string
	Value    struct {
		Token  string `bencode:"token"`
		LbryID string `bencode:"lbryid"`
		Port   int    `bencode:"port"`
	}
	NodeID    bitmap
	SelfStore bool // this is an int on the wire
}

func (s storeArgs) MarshalBencode() ([]byte, error) {
	encodedValue, err := bencode.EncodeString(s.Value)
	if err != nil {
		return nil, err
	}

	selfStoreStr := 0
	if s.SelfStore {
		selfStoreStr = 1
	}

	return bencode.EncodeBytes([]interface{}{
		s.BlobHash,
		bencode.RawMessage(encodedValue),
		s.NodeID,
		selfStoreStr,
	})
}

func (s *storeArgs) UnmarshalBencode(b []byte) error {
	var argsInt []bencode.RawMessage
	err := bencode.DecodeBytes(b, &argsInt)
	if err != nil {
		return errors.Prefix("storeArgs unmarshal", err)
	}

	if len(argsInt) != 4 {
		return errors.Err("unexpected number of fields for store args. got " + cast.ToString(len(argsInt)))
	}

	err = bencode.DecodeBytes(argsInt[0], &s.BlobHash)
	if err != nil {
		return errors.Prefix("storeArgs unmarshal", err)
	}

	err = bencode.DecodeBytes(argsInt[1], &s.Value)
	if err != nil {
		return errors.Prefix("storeArgs unmarshal", err)
	}

	err = bencode.DecodeBytes(argsInt[2], &s.NodeID)
	if err != nil {
		return errors.Prefix("storeArgs unmarshal", err)
	}

	var selfStore int
	err = bencode.DecodeBytes(argsInt[3], &selfStore)
	if err != nil {
		return errors.Prefix("storeArgs unmarshal", err)
	}
	if selfStore == 0 {
		s.SelfStore = false
	} else if selfStore == 1 {
		s.SelfStore = true
	} else {
		return errors.Err("selfstore must be 1 or 0")
	}

	return nil
}

type findNodeDatum struct {
	ID   bitmap
	IP   string
	Port int
}

func (f *findNodeDatum) UnmarshalBencode(b []byte) error {
	var contact []bencode.RawMessage
	err := bencode.DecodeBytes(b, &contact)
	if err != nil {
		return err
	}

	if len(contact) != 3 {
		return errors.Err("invalid-sized contact")
	}

	err = bencode.DecodeBytes(contact[0], &f.ID)
	if err != nil {
		return err
	}
	err = bencode.DecodeBytes(contact[1], &f.IP)
	if err != nil {
		return err
	}
	err = bencode.DecodeBytes(contact[2], &f.Port)
	if err != nil {
		return err
	}

	return nil
}

type Response struct {
	ID           string
	NodeID       string
	Data         string
	FindNodeData []findNodeDatum
}

func (r Response) MarshalBencode() ([]byte, error) {
	data := map[string]interface{}{
		headerTypeField:      responseType,
		headerMessageIDField: r.ID,
		headerNodeIDField:    r.NodeID,
	}
	if r.Data != "" {
		data[headerPayloadField] = r.Data
	} else {
		var nodes []interface{}
		for _, n := range r.FindNodeData {
			nodes = append(nodes, []interface{}{n.ID, n.IP, n.Port})
		}
		data[headerPayloadField] = nodes
	}

	return bencode.EncodeBytes(data)
}

func (r *Response) UnmarshalBencode(b []byte) error {
	var raw struct {
		ID     string             `bencode:"1"`
		NodeID string             `bencode:"2"`
		Data   bencode.RawMessage `bencode:"2"`
	}
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return err
	}

	r.ID = raw.ID
	r.NodeID = raw.NodeID

	err = bencode.DecodeBytes(raw.Data, &r.Data)
	if err != nil {
		err = bencode.DecodeBytes(raw.Data, r.FindNodeData)
		if err != nil {
			return err
		}
	}

	return nil
}

type Error struct {
	ID            string
	NodeID        string
	Response      []string
	ExceptionType string
}

func (e Error) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      errorType,
		headerMessageIDField: e.ID,
		headerNodeIDField:    e.NodeID,
		headerPayloadField:   e.ExceptionType,
		headerArgsField:      e.Response,
	})
}
