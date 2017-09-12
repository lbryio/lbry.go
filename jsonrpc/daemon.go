package jsonrpc

import (
	"encoding/json"
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

func (d *Client) callNoDecode(command string, params map[string]interface{}) (interface{}, error) {
	r, err := d.conn.CallNamed(command, params)
	if err != nil {
		return nil, err
	}

	if r.Error != nil {
		return nil, errors.New("Error in daemon: " + r.Error.Message)
	}

	return r.Result, nil
}

func (d *Client) call(response interface{}, command string, params map[string]interface{}) error {
	result, err := d.callNoDecode(command, params)
	if err != nil {
		return err
	}
	return decode(result, response)
}

func (d *Client) Commands() (*CommandsResponse, error) {
	response := new(CommandsResponse)
	return response, d.call(response, "commands", map[string]interface{}{})
}

func (d *Client) Status() (*StatusResponse, error) {
	response := new(StatusResponse)
	return response, d.call(response, "status", map[string]interface{}{})
}

func (d *Client) WalletBalance() (*WalletBalanceResponse, error) {
	response := new(WalletBalanceResponse)
	return response, d.call(response, "wallet_balance", map[string]interface{}{})
}

func (d *Client) Version() (*VersionResponse, error) {
	response := new(VersionResponse)
	return response, d.call(response, "version", map[string]interface{}{})
}

func (d *Client) Get(url string, filename *string, timeout *uint) (*GetResponse, error) {
	response := new(GetResponse)
	return response, d.call(response, "get", map[string]interface{}{
		"uri":       url,
		"file_name": filename,
		"timeout":   timeout,
	})
}

func (d *Client) ClaimList(name string) (*ClaimListResponse, error) {
	response := new(ClaimListResponse)
	return response, d.call(response, "claim_list", map[string]interface{}{
		"name": name,
	})
}

func (d *Client) ClaimShow(claimID *string, txid *string, nout *uint) (*ClaimShowResponse, error) {
	response := new(ClaimShowResponse)
	return response, d.call(response, "claim_show", map[string]interface{}{
		"claim_id": claimID,
		"txid":     txid,
		"nout":     nout,
	})
}

func (d *Client) PeerList(blobHash string, timeout *uint) (*PeerListResponse, error) {
	rawResponse, err := d.callNoDecode("peer_list", map[string]interface{}{
		"blob_hash": blobHash,
		"timeout":   timeout,
	})
	if err != nil {
		return nil, err
	}

	castResponse, ok := rawResponse.([]interface{})
	if !ok {
		return nil, errors.New("invalid peer_list response")
	}

	peers := []PeerListResponsePeer{}
	for _, peer := range castResponse {
		t, ok := peer.([]interface{})
		if !ok {
			return nil, errors.New("invalid peer_list response")
		}

		if len(t) != 3 {
			return nil, errors.New("invalid triplet in peer_list response")
		}

		ip, ok := t[0].(string)
		if !ok {
			return nil, errors.New("invalid ip in peer_list response")
		}
		port, ok := t[1].(json.Number)
		if !ok {
			return nil, errors.New("invalid port in peer_list response")
		}
		available, ok := t[2].(bool)
		if !ok {
			return nil, errors.New("invalid is_available in peer_list response")
		}

		portNum, err := port.Int64()
		if err != nil {
			return nil, err
		} else if portNum < 0 {
			return nil, errors.New("invalid port in peer_list response")
		}

		peers = append(peers, PeerListResponsePeer{
			IP:          ip,
			Port:        uint(portNum),
			IsAvailable: available,
		})
	}

	response := PeerListResponse(peers)
	return &response, nil
}

func (d *Client) BlobGet(blobHash string, encoding *string, timeout *uint) (*BlobGetResponse, error) {
	rawResponse, err := d.callNoDecode("blob_get", map[string]interface{}{
		"blob_hash": blobHash,
		"timeout":   timeout,
		"encoding":  encoding,
	})
	if err != nil {
		return nil, err
	}

	if _, ok := rawResponse.(string); ok {
		return nil, nil // blob was downloaded, nothing to return
	}

	response := new(BlobGetResponse)
	return response, decode(rawResponse, response)
}

func (d *Client) StreamCostEstimate(url string, size *uint64) (*StreamCostEstimateResponse, error) {
	response := new(StreamCostEstimateResponse)
	return response, d.call(response, "stream_cost_estimate", map[string]interface{}{
		"uri":  url,
		"size": size,
	})
}

func (d *Client) FileList() (*FileListResponse, error) {
	response := new(FileListResponse)
	return response, d.call(response, "file_list", map[string]interface{}{})
}

func (d *Client) Resolve(url string) (*ResolveResponse, error) {
	response := new(ResolveResponse)
	return response, d.call(response, "resolve", map[string]interface{}{
		"uri": url,
	})
}

//func (d *Client) Publish() (*PublishResponse, error) {
//	response := new(PublishResponse)
//	return response, d.call(response, "publish")
//}
