package jsonrpc

import (
	"errors"
	"strconv"

	"github.com/mitchellh/mapstructure"
	"github.com/ybbus/jsonrpc"
)

const DefaultPort = 5279

type Client struct {
	conn *jsonrpc.RPCClient
}

func NewClient(address string) *Client {
	d := Client{}

	if address == "" {
		address = "http://localhost:" + strconv.Itoa(DefaultPort)
	}

	d.conn = jsonrpc.NewRPCClient(address)

	return &d
}

func decode(data interface{}, targetStruct interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   targetStruct,
		TagName:  "json",
		//WeaklyTypedInput: true,
		DecodeHook: fixDecodeProto,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(data)
}

func (d *Client) call(response interface{}, command string, params ...interface{}) error {
	r, err := d.conn.Call(command, params...)
	if err != nil {
		return err
	}

	if r.Error != nil {
		return errors.New("Error in daemon: " + r.Error.Message)
	}

	return decode(r.Result, response)
}

func (d *Client) Commands() (*CommandsResponse, error) {
	response := &CommandsResponse{}
	return response, d.call(response, "commands")
}

func (d *Client) Status() (*StatusResponse, error) {
	response := &StatusResponse{}
	return response, d.call(response, "status")
}

func (d *Client) Get(url string, filename *string, timeout *uint) (*GetResponse, error) {
	response := &GetResponse{}
	return response, d.call(response, "get", map[string]interface{}{
		"uri":       url,
		"file_name": filename,
		"timeout":   timeout,
	})
}
