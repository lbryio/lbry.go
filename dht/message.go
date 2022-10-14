package dht

import (
	"crypto/rand"
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"

	"github.com/lbryio/lbry.go/v3/dht/bits"

	"github.com/cockroachdb/errors"
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
	contactsField        = "contacts"
	pageField            = "p"
	tokenField           = "token"
	protocolVersionField = "protocolVersion"
)

// Message is a DHT message
type Message interface {
	bencode.Marshaler
}

type messageID [messageIDLength]byte

// HexShort returns the first 8 hex characters of the hex encoded message id.
func (m messageID) HexShort() string {
	return hex.EncodeToString(m[:])[:8]
}

// UnmarshalBencode takes a byte slice and unmarshals the message id.
func (m *messageID) UnmarshalBencode(encoded []byte) error {
	var str string
	err := bencode.DecodeBytes(encoded, &str)
	if err != nil {
		return errors.Wrap(err, "")
	}
	copy(m[:], str)
	return nil
}

// MarshalBencode returns the encoded byte slice of the message id.
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

// Request represents a DHT request message
type Request struct {
	ID              messageID
	NodeID          bits.Bitmap
	Method          string
	Arg             *bits.Bitmap
	StoreArgs       *storeArgs
	ProtocolVersion int
}

// MarshalBencode returns the serialized byte slice representation of the request
func (r Request) MarshalBencode() ([]byte, error) {
	var args interface{}
	if r.StoreArgs != nil {
		args = r.StoreArgs
	} else if r.Arg != nil {
		args = []bits.Bitmap{*r.Arg}
	} else {
		args = []string{} // request must always have keys 0-4, so we use an empty list for PING
	}
	b, err := bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      requestType,
		headerMessageIDField: r.ID,
		headerNodeIDField:    r.NodeID,
		headerPayloadField:   r.Method,
		headerArgsField:      args,
	})
	return b, errors.Wrap(err, "bencode")
}

// UnmarshalBencode unmarshals the serialized byte slice into the appropriate fields of the request.
func (r *Request) UnmarshalBencode(b []byte) error {
	var raw struct {
		ID     messageID          `bencode:"1"`
		NodeID bits.Bitmap        `bencode:"2"`
		Method string             `bencode:"3"`
		Args   bencode.RawMessage `bencode:"4"`
	}
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return errors.Wrap(err, "request unmarshal")
	}

	r.ID = raw.ID
	r.NodeID = raw.NodeID
	r.Method = raw.Method

	if r.Method == storeMethod {
		r.StoreArgs = &storeArgs{} // bencode wont find the unmarshaler on a null pointer. need to fix it.
		err = bencode.DecodeBytes(raw.Args, &r.StoreArgs)
		if err != nil {
			return errors.Wrap(err, "request unmarshal")
		}
	} else if len(raw.Args) > 2 { // 2 because an empty list is `le`
		r.Arg, r.ProtocolVersion, err = processArgsAndProtoVersion(raw.Args)
		if err != nil {
			return errors.WithMessage(err, "request unmarshal")
		}
	}

	return nil
}

func processArgsAndProtoVersion(raw bencode.RawMessage) (arg *bits.Bitmap, version int, err error) {
	var args []bencode.RawMessage
	err = bencode.DecodeBytes(raw, &args)
	if err != nil {
		return nil, 0, errors.Wrap(err, "")
	}

	if len(args) == 0 {
		return nil, 0, nil
	}

	var extras map[string]int
	err = bencode.DecodeBytes(args[len(args)-1], &extras)
	if err == nil {
		if v, exists := extras[protocolVersionField]; exists {
			version = v
			args = args[:len(args)-1]
		}
	}

	if len(args) > 0 {
		var b bits.Bitmap
		err = bencode.DecodeBytes(args[0], &b)
		if err != nil {
			return nil, 0, errors.Wrap(err, "")
		}
		arg = &b
	}

	return arg, version, nil
}

func (r Request) argsDebug() string {
	if r.StoreArgs != nil {
		return r.StoreArgs.BlobHash.HexShort() + ", " + r.StoreArgs.Value.LbryID.HexShort() + ":" + strconv.Itoa(r.StoreArgs.Value.Port)
	} else if r.Arg != nil {
		return r.Arg.HexShort()
	}
	return ""
}

type storeArgsValue struct {
	Token  string      `bencode:"token"`
	LbryID bits.Bitmap `bencode:"lbryid"`
	Port   int         `bencode:"port"`
}

type storeArgs struct {
	BlobHash  bits.Bitmap
	Value     storeArgsValue
	NodeID    bits.Bitmap // original publisher id? I think this is getting fixed in the new dht stuff
	SelfStore bool        // this is an int on the wire
}

// MarshalBencode returns the serialized byte slice representation of the storage arguments.
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

// UnmarshalBencode unmarshals the serialized byte slice into the appropriate fields of the store arguments.
func (s *storeArgs) UnmarshalBencode(b []byte) error {
	var argsInt []bencode.RawMessage
	err := bencode.DecodeBytes(b, &argsInt)
	if err != nil {
		return errors.Wrap(err, "storeArgs unmarshal")
	}

	if len(argsInt) != 4 {
		return errors.Wrap(errors.Newf("unexpected number of fields for store args. got %d", len(argsInt)), "")
	}

	err = bencode.DecodeBytes(argsInt[0], &s.BlobHash)
	if err != nil {
		return errors.Wrap(err, "storeArgs unmarshal")
	}

	err = bencode.DecodeBytes(argsInt[1], &s.Value)
	if err != nil {
		return errors.Wrap(err, "storeArgs unmarshal")
	}

	err = bencode.DecodeBytes(argsInt[2], &s.NodeID)
	if err != nil {
		return errors.Wrap(err, "storeArgs unmarshal")
	}

	var selfStore int
	err = bencode.DecodeBytes(argsInt[3], &selfStore)
	if err != nil {
		return errors.Wrap(err, "storeArgs unmarshal")
	}
	if selfStore == 0 {
		s.SelfStore = false
	} else if selfStore == 1 {
		s.SelfStore = true
	} else {
		return errors.Wrap(errors.New("selfstore must be 1 or 0"), "")
	}

	return nil
}

// Response represents a DHT response message
type Response struct {
	ID              messageID
	NodeID          bits.Bitmap
	Data            string
	Contacts        []Contact
	FindValueKey    string
	Token           string
	ProtocolVersion int
	Page            uint8
}

func (r Response) argsDebug() string {
	if r.Data != "" {
		return r.Data
	}

	str := "contacts "
	if r.FindValueKey != "" {
		str = "value for " + hex.EncodeToString([]byte(r.FindValueKey))[:8] + " "
	}

	str += "|"
	for _, c := range r.Contacts {
		str += c.String() + ","
	}
	str = strings.TrimRight(str, ",") + "|"

	if r.Token != "" {
		str += " token: " + hex.EncodeToString([]byte(r.Token))[:8]
	}

	return str
}

// MarshalBencode returns the serialized byte slice representation of the response.
func (r Response) MarshalBencode() ([]byte, error) {
	data := map[string]interface{}{
		headerTypeField:      responseType,
		headerMessageIDField: r.ID,
		headerNodeIDField:    r.NodeID,
	}

	if r.Data != "" {
		// ping or store
		data[headerPayloadField] = r.Data
	} else if r.FindValueKey != "" {
		// findValue success
		if r.Token == "" {
			return nil, errors.WithStack(errors.New("response to findValue must have a token"))
		}

		var contacts [][]byte
		for _, c := range r.Contacts {
			compact, err := c.MarshalCompact()
			if err != nil {
				return nil, err
			}
			contacts = append(contacts, compact)
		}
		data[headerPayloadField] = map[string]interface{}{
			r.FindValueKey: contacts,
			tokenField:     r.Token,
		}
	} else if r.Token != "" {
		// findValue failure falling back to findNode
		data[headerPayloadField] = map[string]interface{}{
			contactsField: r.Contacts,
			tokenField:    r.Token,
		}
	} else {
		// straight up findNode
		data[headerPayloadField] = r.Contacts
	}

	return bencode.EncodeBytes(data)
}

// UnmarshalBencode unmarshals the serialized byte slice into the appropriate fields of the store arguments.
func (r *Response) UnmarshalBencode(b []byte) error {
	var raw struct {
		ID     messageID          `bencode:"1"`
		NodeID bits.Bitmap        `bencode:"2"`
		Data   bencode.RawMessage `bencode:"3"`
	}
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return err
	}

	r.ID = raw.ID
	r.NodeID = raw.NodeID

	// maybe data is a string (response to ping or store)?
	err = bencode.DecodeBytes(raw.Data, &r.Data)
	if err == nil {
		return nil
	}

	// maybe data is a list of contacts (response to findNode)?
	err = bencode.DecodeBytes(raw.Data, &r.Contacts)
	if err == nil {
		return nil
	}

	// it must be a response to findValue
	var rawData map[string]bencode.RawMessage
	err = bencode.DecodeBytes(raw.Data, &rawData)
	if err != nil {
		return err
	}

	if token, ok := rawData[tokenField]; ok {
		err = bencode.DecodeBytes(token, &r.Token)
		if err != nil {
			return err
		}
		delete(rawData, tokenField) // so it doesnt mess up findValue key finding below
	}

	if protocolVersion, ok := rawData[protocolVersionField]; ok {
		err = bencode.DecodeBytes(protocolVersion, &r.ProtocolVersion)
		if err != nil {
			return err
		}
		delete(rawData, protocolVersionField) // so it doesnt mess up findValue key finding below
	}

	if contacts, ok := rawData[contactsField]; ok {
		err = bencode.DecodeBytes(contacts, &r.Contacts)
		delete(rawData, contactsField) // so it doesnt mess up findValue key finding below
		if err != nil {
			return err
		}
	}
	if page, ok := rawData[pageField]; ok {
		err = bencode.DecodeBytes(page, &r.Page)
		delete(rawData, pageField) // so it doesnt mess up findValue key finding below
		if err != nil {
			return err
		}
	}
	for k, v := range rawData {
		r.FindValueKey = k
		var compactContacts [][]byte
		err = bencode.DecodeBytes(v, &compactContacts)
		if err != nil {
			return err
		}
		for _, compact := range compactContacts {
			var c Contact
			err = c.UnmarshalCompact(compact)
			if err != nil {
				return err
			}
			r.Contacts = append(r.Contacts, c)
		}
		break
	}

	return nil
}

// Error represents a DHT error response
type Error struct {
	ID            messageID
	NodeID        bits.Bitmap
	ExceptionType string
	Response      []string
}

// MarshalBencode returns the serialized byte slice representation of an error message.
func (e Error) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      errorType,
		headerMessageIDField: e.ID,
		headerNodeIDField:    e.NodeID,
		headerPayloadField:   e.ExceptionType,
		headerArgsField:      e.Response,
	})
}

// UnmarshalBencode unmarshals the serialized byte slice into the appropriate fields of the error message.
func (e *Error) UnmarshalBencode(b []byte) error {
	var raw struct {
		ID            messageID   `bencode:"1"`
		NodeID        bits.Bitmap `bencode:"2"`
		ExceptionType string      `bencode:"3"`
		Args          interface{} `bencode:"4"`
	}
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return errors.Wrap(err, "")
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
