package dht

import "github.com/zeebo/bencode"

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
	headerMessageIDField = "1"
	headerNodeIDField    = "2"
	headerPayloadField   = "3"
	headerArgsField      = "4"
)

type Message interface {
	GetID() string
	Encode() ([]byte, error)
}

type Request struct {
	ID     string
	NodeID string
	Method string
	Args   []string
}

func (r Request) GetID() string { return r.ID }
func (r Request) Encode() ([]byte, error) {
	return bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      requestType,
		headerMessageIDField: r.ID,
		headerNodeIDField:    r.NodeID,
		headerPayloadField:   r.Method,
		headerArgsField:      r.Args,
	})
}

type findNodeDatum struct {
	ID   string
	IP   string
	Port int
}
type Response struct {
	ID           string
	NodeID       string
	Data         string
	FindNodeData []findNodeDatum
}

func (r Response) GetID() string { return r.ID }
func (r Response) Encode() ([]byte, error) {
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

type Error struct {
	ID            string
	NodeID        string
	Response      []string
	ExceptionType string
}

func (e Error) GetID() string { return e.ID }
func (e Error) Encode() ([]byte, error) {
	return bencode.EncodeBytes(map[string]interface{}{
		headerTypeField:      errorType,
		headerMessageIDField: e.ID,
		headerNodeIDField:    e.NodeID,
		headerPayloadField:   e.ExceptionType,
		headerArgsField:      e.Response,
	})
}
