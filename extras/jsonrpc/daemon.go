package jsonrpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/mitchellh/mapstructure"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/ybbus/jsonrpc"
)

const DefaultPort = 5279

type Client struct {
	conn    jsonrpc.RPCClient
	address string
}

func NewClient(address string) *Client {
	d := Client{}

	if address == "" {
		address = "http://localhost:" + strconv.Itoa(DefaultPort)
	}

	d.conn = jsonrpc.NewClient(address)
	d.address = address

	return &d
}

func NewClientAndWait(address string) *Client {
	d := NewClient(address)
	for {
		_, err := d.WalletBalance()
		if err == nil {
			return d
		}
		time.Sleep(5 * time.Second)
	}
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
		return errors.Wrap(err, 0)
	}

	err = decoder.Decode(data)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	return nil
}

func decodeNumber(data interface{}) (decimal.Decimal, error) {
	var number string

	switch d := data.(type) {
	case json.Number:
		number = d.String()
	case string:
		number = d
	default:
		return decimal.Decimal{}, errors.Err("unexpected number type")
	}

	dec, err := decimal.NewFromString(number)
	if err != nil {
		return decimal.Decimal{}, errors.Wrap(err, 0)
	}

	return dec, nil
}

func debugParams(params map[string]interface{}) string {
	var s []string
	for k, v := range params {
		r := reflect.ValueOf(v)
		if r.Kind() == reflect.Ptr {
			if r.IsNil() {
				continue
			}
			v = r.Elem().Interface()
		}
		s = append(s, fmt.Sprintf("%s=%+v", k, v))
	}
	sort.Strings(s)
	return strings.Join(s, " ")
}

func (d *Client) callNoDecode(command string, params map[string]interface{}) (interface{}, error) {
	log.Debugln("jsonrpc: " + command + " " + debugParams(params))
	r, err := d.conn.Call(command, params)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if r.Error != nil {
		return nil, errors.Err("Error in daemon: " + r.Error.Message)
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

func (d *Client) SetRPCTimeout(timeout time.Duration) {
	d.conn = jsonrpc.NewClientWithOpts(d.address, &jsonrpc.RPCClientOpts{
		HTTPClient: &http.Client{Timeout: timeout},
	})
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
	rawResponse, err := d.callNoDecode("wallet_balance", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	dec, err := decodeNumber(rawResponse)
	if err != nil {
		return nil, err
	}

	response := WalletBalanceResponse(dec)
	return &response, nil
}

func (d *Client) WalletList() (*WalletListResponse, error) {
	response := new(WalletListResponse)
	return response, d.call(response, "wallet_list", map[string]interface{}{})
}

func (d *Client) UTXOList() (*UTXOListResponse, error) {
	response := new(UTXOListResponse)
	return response, d.call(response, "utxo_list", map[string]interface{}{})
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
		return nil, errors.Err("invalid peer_list response")
	}

	peers := []PeerListResponsePeer{}
	for _, peer := range castResponse {
		t, ok := peer.(map[string]interface{})
		if !ok {
			return nil, errors.Err("invalid peer_list response")
		}

		if len(t) != 3 {
			return nil, errors.Err("invalid triplet in peer_list response")
		}

		ip, ok := t["host"].(string)
		if !ok {
			return nil, errors.Err("invalid ip in peer_list response")
		}
		port, ok := t["port"].(json.Number)
		if !ok {
			return nil, errors.Err("invalid port in peer_list response")
		}
		nodeid, ok := t["node_id"].(string)
		if !ok {
			return nil, errors.Err("invalid nodeid in peer_list response")
		}

		portNum, err := port.Int64()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		} else if portNum < 0 {
			return nil, errors.Err("invalid port in peer_list response")
		}

		peers = append(peers, PeerListResponsePeer{
			IP:     ip,
			Port:   uint(portNum),
			NodeId: nodeid,
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

func (d *Client) StreamAvailability(url string, search_timeout *uint64, blob_timeout *uint64) (*StreamAvailabilityResponse, error) {
	response := new(StreamAvailabilityResponse)
	return response, d.call(response, "stream_availability", map[string]interface{}{
		"uri":            url,
		"search_timeout": search_timeout,
		"blob_timeout":   blob_timeout,
	})
}

func (d *Client) StreamCostEstimate(url string, size *uint64) (*StreamCostEstimateResponse, error) {
	rawResponse, err := d.callNoDecode("stream_cost_estimate", map[string]interface{}{
		"uri":  url,
		"size": size,
	})
	if err != nil {
		return nil, err
	}

	dec, err := decodeNumber(rawResponse)
	if err != nil {
		return nil, err
	}

	response := StreamCostEstimateResponse(dec)
	return &response, nil
}

type FileListOptions struct {
	SDHash     *string
	StreamHash *string
	FileName   *string
	ClaimID    *string
	Outpoint   *string
	RowID      *string
	Name       *string
}

func (d *Client) FileList(options FileListOptions) (*FileListResponse, error) {
	response := new(FileListResponse)
	return response, d.call(response, "file_list", map[string]interface{}{
		"sd_hash":     options.SDHash,
		"stream_hash": options.StreamHash,
		"file_name":   options.FileName,
		"claim_id":    options.ClaimID,
		"outpoint":    options.Outpoint,
		"rowid":       options.RowID,
		"name":        options.Name,
	})
}

func (d *Client) Resolve(url string) (*ResolveResponse, error) {
	response := new(ResolveResponse)
	return response, d.call(response, "resolve", map[string]interface{}{
		"uri": url,
	})
}

func (d *Client) ChannelNew(name string, amount float64) (*ChannelNewResponse, error) {
	response := new(ChannelNewResponse)
	return response, d.call(response, "channel_new", map[string]interface{}{
		"channel_name": name,
		"amount":       amount,
	})
}

func (d *Client) ChannelList() (*ChannelListResponse, error) {
	response := new(ChannelListResponse)
	return response, d.call(response, "channel_list", map[string]interface{}{})
}

type PublishOptions struct {
	Fee           *Fee
	Title         *string
	Description   *string
	Author        *string
	Language      *string
	License       *string
	LicenseURL    *string
	Thumbnail     *string
	Preview       *string
	NSFW          *bool
	ChannelName   *string
	ChannelID     *string
	ClaimAddress  *string
	ChangeAddress *string
}

func (d *Client) Publish(name, filePath string, bid float64, options PublishOptions) (*PublishResponse, error) {
	response := new(PublishResponse)
	return response, d.call(response, "publish", map[string]interface{}{
		"name":           name,
		"file_path":      filePath,
		"bid":            bid,
		"fee":            options.Fee,
		"title":          options.Title,
		"description":    options.Description,
		"author":         options.Author,
		"language":       options.Language,
		"license":        options.License,
		"license_url":    options.LicenseURL,
		"thumbnail":      options.Thumbnail,
		"preview":        options.Preview,
		"nsfw":           options.NSFW,
		"channel_name":   options.ChannelName,
		"channel_id":     options.ChannelID,
		"claim_address":  options.ClaimAddress,
		"change_address": options.ChangeAddress,
	})
}

func (d *Client) BlobAnnounce(blobHash, sdHash, streamHash *string) (*BlobAnnounceResponse, error) {
	response := new(BlobAnnounceResponse)
	return response, d.call(response, "blob_announce", map[string]interface{}{
		"blob_hash":   blobHash,
		"stream_hash": streamHash,
		"sd_hash":     sdHash,
	})
}

func (d *Client) WalletPrefillAddresses(numAddresses int, amount decimal.Decimal, broadcast bool) (*WalletPrefillAddressesResponse, error) {
	if numAddresses < 1 {
		return nil, errors.Err("must create at least 1 address")
	}
	response := new(WalletPrefillAddressesResponse)
	return response, d.call(response, "wallet_prefill_addresses", map[string]interface{}{
		"num_addresses": numAddresses,
		"amount":        amount,
		"no_broadcast":  !broadcast,
	})
}

func (d *Client) WalletNewAddress() (*WalletNewAddressResponse, error) {
	rawResponse, err := d.callNoDecode("wallet_new_address", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	address, ok := rawResponse.(string)
	if !ok {
		return nil, errors.Err("unexpected response")
	}

	response := WalletNewAddressResponse(address)
	return &response, nil
}

func (d *Client) WalletUnusedAddress() (*WalletUnusedAddressResponse, error) {
	rawResponse, err := d.callNoDecode("wallet_unused_address", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	address, ok := rawResponse.(string)
	if !ok {
		return nil, errors.Err("unexpected response")
	}

	response := WalletUnusedAddressResponse(address)
	return &response, nil
}

func (d *Client) NumClaimsInChannel(url string) (uint64, error) {
	response := new(NumClaimsInChannelResponse)
	err := d.call(response, "claim_list_by_channel", map[string]interface{}{
		"uri": url,
	})
	if err != nil {
		return 0, err
	} else if response == nil {
		return 0, errors.Err("no response")
	}

	channel, ok := (*response)[url]
	if !ok {
		return 0, errors.Err("url not in response")
	}
	if channel.Error != "" {
		if strings.Contains(channel.Error, "cannot be resolved") {
			return 0, nil
		}
		return 0, errors.Err(channel.Error)
	}
	return channel.ClaimsInChannel, nil
}

func (d *Client) ClaimListMine() (*ClaimListMineResponse, error) {
	response := new(ClaimListMineResponse)
	err := d.call(response, "claim_list_mine", map[string]interface{}{})
	if err != nil {
		return nil, err
	} else if response == nil {
		return nil, errors.Err("no response")
	}

	return response, nil
}

func (d *Client) ClaimAbandon(txID string, nOut int) (*ClaimAbandonResponse, error) {
	response := new(ClaimAbandonResponse)
	err := d.call(response, "claim_abandon", map[string]interface{}{
		"txid": txID,
		"nout": nOut,
	})
	if err != nil {
		return nil, err
	} else if response == nil {
		return nil, errors.Err("no response")
	}

	return response, nil
}
