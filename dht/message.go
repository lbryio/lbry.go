package dht

import (
	"crypto/rand"
	"encoding/hex"
	"reflect"
	"strings"

	"github.com/lbryio/errors.go"

	"github.com/lyoshenka/bencode"
	"github.com/spf13/cast"
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

type messageID [messageIDLength]byte

func (m messageID) HexShort() string {
	return hex.EncodeToString(m[:])[:8]
}

func (m *messageID) UnmarshalBencode(encoded []byte) error {
	var str string
	err := bencode.DecodeBytes(encoded, &str)
	if err != nil {
		return err
	}
	copy(m[:], str)
	return nil
}

func (m messageID) MarshalBencode() ([]byte, error) {
	str := string(m[:])
	return bencode.EncodeBytes(str)
}

func newMessageID() messageID {
	var m messageID
	_, err := rand.Read(m[:])
	if err != nil {
		panic(err)
	}
	return m
}

type Request struct {
	ID        messageID
	NodeID    bitmap
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
		ID     messageID          `bencode:"1"`
		NodeID bitmap             `bencode:"2"`
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

type storeArgsValue struct {
	Token  string `bencode:"token"`
	LbryID bitmap `bencode:"lbryid"`
	Port   int    `bencode:"port"`
}

type storeArgs struct {
	BlobHash  string
	Value     storeArgsValue
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

type Response struct {
	ID           messageID
	NodeID       bitmap
	Data         string
	FindNodeData []Node
	FindValueKey string
}

func (r Response) ArgsDebug() string {
	if r.Data != "" {
		return r.Data
	}

	str := "contacts "
	if r.FindValueKey != "" {
		str = "value for " + hex.EncodeToString([]byte(r.FindValueKey))[:8] + " "
	}

	str += "|"
	for _, c := range r.FindNodeData {
		str += c.Addr().String() + ":" + c.id.HexShort() + ","
	}
	str = strings.TrimRight(str, ",") + "|"
	return str
}

func (r Response) MarshalBencode() ([]byte, error) {
	data := map[string]interface{}{
		headerTypeField:      responseType,
		headerMessageIDField: r.ID,
		headerNodeIDField:    r.NodeID,
	}
	if r.Data != "" {
		data[headerPayloadField] = r.Data
	} else if r.FindValueKey != "" {
		var contacts [][]byte
		for _, n := range r.FindNodeData {
			compact, err := n.MarshalCompact()
			if err != nil {
				return nil, err
			}
			contacts = append(contacts, compact)
		}
		data[headerPayloadField] = map[string][][]byte{r.FindValueKey: contacts}
	} else {
		data[headerPayloadField] = map[string][]Node{"contacts": r.FindNodeData}
	}

	return bencode.EncodeBytes(data)
}

func (r *Response) UnmarshalBencode(b []byte) error {
	var raw struct {
		ID     messageID          `bencode:"1"`
		NodeID bitmap             `bencode:"2"`
		Data   bencode.RawMessage `bencode:"3"`
	}
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return err
	}

	r.ID = raw.ID
	r.NodeID = raw.NodeID

	err = bencode.DecodeBytes(raw.Data, &r.Data)
	if err != nil {
		var rawData map[string]bencode.RawMessage
		err = bencode.DecodeBytes(raw.Data, &rawData)
		if err != nil {
			return err
		}

		if contacts, ok := rawData["contacts"]; ok {
			err = bencode.DecodeBytes(contacts, &r.FindNodeData)
			if err != nil {
				return err
			}
		} else {
			for k, v := range rawData {
				r.FindValueKey = k
				var compactNodes [][]byte
				err = bencode.DecodeBytes(v, &compactNodes)
				if err != nil {
					return err
				}
				for _, compact := range compactNodes {
					var uncompactedNode Node
					err = uncompactedNode.UnmarshalCompact(compact)
					if err != nil {
						return err
					}
					r.FindNodeData = append(r.FindNodeData, uncompactedNode)
				}
				break
			}
		}
	}

	return nil
}

type Error struct {
	ID            messageID
	NodeID        bitmap
	ExceptionType string
	Response      []string
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

func (e *Error) UnmarshalBencode(b []byte) error {
	var raw struct {
		ID            messageID   `bencode:"1"`
		NodeID        bitmap      `bencode:"2"`
		ExceptionType string      `bencode:"3"`
		Args          interface{} `bencode:"4"`
	}
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return err
	}

	e.ID = raw.ID
	e.NodeID = raw.NodeID
	e.ExceptionType = raw.ExceptionType

	if reflect.TypeOf(raw.Args).Kind() == reflect.Slice {
		v := reflect.ValueOf(raw.Args)
		for i := 0; i < v.Len(); i++ {
			e.Response = append(e.Response, cast.ToString(v.Index(i).Interface()))
		}
	}

	return nil
}
