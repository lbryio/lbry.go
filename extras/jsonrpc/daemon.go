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

	"github.com/fatih/structs"
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
		_, err := d.AccountBalance(nil)
		if err == nil {
			return d
		}
		time.Sleep(5 * time.Second)
	}
}

func Decode(data interface{}, targetStruct interface{}) error {
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
	return Decode(result, response)
}

func (d *Client) SetRPCTimeout(timeout time.Duration) {
	d.conn = jsonrpc.NewClientWithOpts(d.address, &jsonrpc.RPCClientOpts{
		HTTPClient: &http.Client{Timeout: timeout},
	})
}

//============================================
//				NEW SDK
//============================================
func (d *Client) AccountList() (*AccountListResponse, error) {
	response := new(AccountListResponse)
	return response, d.call(response, "account_list", map[string]interface{}{})
}

func (d *Client) SingleAccountList(accountID string) (*Account, error) {
	response := new(Account)
	return response, d.call(response, "account_list", map[string]interface{}{"account_id": accountID})
}

type AccountSettings struct {
	Default          bool   `json:"default"`
	NewName          string `json:"new_name"`
	ReceivingGap     int    `json:"receiving_gap"`
	ReceivingMaxUses int    `json:"receiving_max_uses"`
	ChangeGap        int    `json:"change_gap"`
	ChangeMaxUses    int    `json:"change_max_uses"`
}

func (d *Client) AccountSet(accountID string, settings AccountSettings) (*Account, error) {
	response := new(Account)
	return response, d.call(response, "account_list", map[string]interface{}{})
}

func (d *Client) AccountBalance(account *string) (*AccountBalanceResponse, error) {
	response := new(AccountBalanceResponse)
	return response, d.call(response, "account_balance", map[string]interface{}{
		"account_id": account,
	})
}

// funds an account. If everything is true then amount is ignored
func (d *Client) AccountFund(fromAccount string, toAccount string, amount string, outputs uint64, everything bool) (*AccountFundResponse, error) {
	response := new(AccountFundResponse)
	return response, d.call(response, "account_fund", map[string]interface{}{
		"from_account": fromAccount,
		"to_account":   toAccount,
		"amount":       amount,
		"outputs":      outputs,
		"everything":   everything,
		"broadcast":    true,
	})
}

func (d *Client) AccountCreate(accountName string, singleKey bool) (*AccountCreateResponse, error) {
	response := new(AccountCreateResponse)
	return response, d.call(response, "account_create", map[string]interface{}{
		"account_name": accountName,
		"single_key":   singleKey,
	})
}

func (d *Client) AccountRemove(accountID string) (*AccountRemoveResponse, error) {
	response := new(AccountRemoveResponse)
	return response, d.call(response, "account_remove", map[string]interface{}{
		"account_id": accountID,
	})
}

func (d *Client) AddressUnused(account *string) (*AddressUnusedResponse, error) {
	response := new(AddressUnusedResponse)
	return response, d.call(response, "address_unused", map[string]interface{}{
		"account_id": account,
	})
}

func (d *Client) ChannelList(account *string, page uint64, pageSize uint64) (*ChannelListResponse, error) {
	if page == 0 {
		return nil, errors.Err("pages start from 1")
	}
	response := new(ChannelListResponse)
	return response, d.call(response, "channel_list", map[string]interface{}{
		"account_id":       account,
		"page":             page,
		"page_size":        pageSize,
		"include_protobuf": true,
	})
}

type streamType string

var (
	StreamTypeVideo = streamType("video")
	StreamTypeAudio = streamType("audio")
	StreamTypeImage = streamType("image")
)

type Location struct {
	Country    *string `json:"country,omitempty"`
	State      *string `json:"state,omitempty"`
	City       *string `json:"city,omitempty"`
	PostalCode *string `json:"code,omitempty"`
	Latitude   *string `json:"latitude,omitempty"`
	Longitude  *string `json:"longitude,omitempty"`
}
type ClaimCreateOptions struct {
	Title        *string    `json:"title,omitempty"`
	Description  *string    `json:"description,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
	Languages    []string   `json:"languages,omitempty"`
	Locations    []Location `json:"locations,omitempty"`
	ThumbnailURL *string    `json:"thumbnail_url,omitempty"`
	AccountID    *string    `json:"account_id,omitempty"`
	ClaimAddress *string    `json:"claim_address,omitempty"`
	Preview      *bool      `json:"preview,omitempty"`
}

type ChannelCreateOptions struct {
	ClaimCreateOptions `json:",flatten"`
	Email              *string  `json:"email,omitempty"`
	WebsiteURL         *string  `json:"website_url,omitempty"`
	CoverURL           *string  `json:"cover_url,omitempty"`
	Featured           []string `json:"featured,omitempty"`
}

func (d *Client) ChannelCreate(name string, bid float64, options ChannelCreateOptions) (*TransactionSummary, error) {
	response := new(TransactionSummary)
	args := struct {
		Name                 string `json:"name"`
		Bid                  string `json:"bid"`
		FilePath             string `json:"file_path,omitempty"`
		IncludeProtoBuf      bool   `json:"include_protobuf"`
		ChannelCreateOptions `json:",flatten"`
		Blocking             bool `json:"blocking"`
	}{
		Name:                 name,
		Bid:                  fmt.Sprintf("%.6f", bid),
		IncludeProtoBuf:      true,
		ChannelCreateOptions: options,
		Blocking:             true,
	}
	structs.DefaultTagName = "json"
	return response, d.call(response, "channel_create", structs.Map(args))
}

type ChannelUpdateOptions struct {
	ChannelCreateOptions `json:",flatten"`
	NewSigningKey        *bool `json:"new_signing_key,omitempty"`
	ClearFeatured        *bool `json:"clear_featured,omitempty"`
	ClearTags            *bool `json:"clear_tags,omitempty"`
	ClearLanguages       *bool `json:"clear_languages,omitempty"`
	ClearLocations       *bool `json:"clear_locations,omitempty"`
}

func (d *Client) ChannelUpdate(claimID string, options ChannelUpdateOptions) (*TransactionSummary, error) {
	response := new(TransactionSummary)
	args := struct {
		ClaimID               string `json:"claim_id"`
		IncludeProtoBuf       bool   `json:"include_protobuf"`
		*ChannelUpdateOptions `json:",flatten"`
		Blocking              bool `json:"blocking"`
	}{
		ClaimID:              claimID,
		IncludeProtoBuf:      true,
		ChannelUpdateOptions: &options,
		Blocking:             true,
	}
	structs.DefaultTagName = "json"
	return response, d.call(response, "channel_update", structs.Map(args))
}

type StreamCreateOptions struct {
	ClaimCreateOptions `json:",flatten"`
	Fee                *Fee        `json:",omitempty,flatten"`
	Author             *string     `json:"author,omitempty"`
	License            *string     `json:"license,omitempty"`
	LicenseURL         *string     `json:"license_url,omitempty"`
	StreamType         *streamType `json:"stream_type,omitempty"`
	ReleaseTime        *int64      `json:"release_time,omitempty"`
	Duration           *uint64     `json:"duration,omitempty"`
	Width              *uint       `json:"width,omitempty"`
	Height             *uint       `json:"height,omitempty"`
	Preview            *string     `json:"preview,omitempty"`
	AllowDuplicateName *bool       `json:"allow_duplicate_name,omitempty"`
	ChannelName        *string     `json:"channel_name,omitempty"`
	ChannelID          *string     `json:"channel_id,omitempty"`
	ChannelAccountID   *string     `json:"channel_account_id,omitempty"`
}

func (d *Client) StreamCreate(name, filePath string, bid float64, options StreamCreateOptions) (*TransactionSummary, error) {
	response := new(TransactionSummary)
	args := struct {
		Name                 string  `json:"name"`
		Bid                  string  `json:"bid"`
		FilePath             string  `json:"file_path,omitempty"`
		FileSize             *string `json:"file_size,omitempty"`
		IncludeProtoBuf      bool    `json:"include_protobuf"`
		Blocking             bool    `json:"blocking"`
		*StreamCreateOptions `json:",flatten"`
	}{
		Name:                name,
		FilePath:            filePath,
		Bid:                 fmt.Sprintf("%.6f", bid),
		IncludeProtoBuf:     true,
		Blocking:            true,
		StreamCreateOptions: &options,
	}
	structs.DefaultTagName = "json"
	return response, d.call(response, "stream_create", structs.Map(args))
}

func (d *Client) StreamAbandon(txID string, nOut uint64, accountID *string, blocking bool) (*ClaimAbandonResponse, error) {
	response := new(ClaimAbandonResponse)
	err := d.call(response, "stream_abandon", map[string]interface{}{
		"txid":             txID,
		"nout":             nOut,
		"account_id":       accountID,
		"include_protobuf": true,
		"blocking":         true,
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

type StreamUpdateOptions struct {
	ClearTags            *bool   `json:"clear_tags,omitempty"`
	ClearLanguages       *bool   `json:"clear_languages,omitempty"`
	ClearLocations       *bool   `json:"clear_locations,omitempty"`
	Name                 *string `json:"name,omitempty"`
	FilePath             *string `json:"file_path,omitempty"`
	FileSize             *uint64 `json:"file_size,omitempty"`
	Bid                  *string `json:"bid,omitempty"`
	*StreamCreateOptions `json:",flatten"`
}

func (d *Client) StreamUpdate(claimID string, options StreamUpdateOptions) (*TransactionSummary, error) {
	response := new(TransactionSummary)
	args := struct {
		ClaimID              string `json:"claim_id"`
		IncludeProtoBuf      bool   `json:"include_protobuf"`
		*StreamUpdateOptions `json:",flatten"`
		Blocking             bool `json:"blocking"`
	}{
		ClaimID:             claimID,
		IncludeProtoBuf:     true,
		StreamUpdateOptions: &options,
		Blocking:            true,
	}
	structs.DefaultTagName = "json"
	return response, d.call(response, "stream_update", structs.Map(args))
}

func (d *Client) ChannelAbandon(txID string, nOut uint64, accountID *string, blocking bool) (*TransactionSummary, error) {
	response := new(TransactionSummary)
	err := d.call(response, "channel_abandon", map[string]interface{}{
		"txid":             txID,
		"nout":             nOut,
		"account_id":       accountID,
		"include_protobuf": true,
		"blocking":         true,
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (d *Client) AddressList(account *string) (*AddressListResponse, error) {
	response := new(AddressListResponse)
	return response, d.call(response, "address_list", map[string]interface{}{
		"account_id": account,
	})
}

func (d *Client) ClaimList(account *string, page uint64, pageSize uint64) (*ClaimListResponse, error) {
	if page == 0 {
		return nil, errors.Err("pages start from 1")
	}
	response := new(ClaimListResponse)
	err := d.call(response, "claim_list", map[string]interface{}{
		"account_id":       account,
		"page":             page,
		"page_size":        pageSize,
		"include_protobuf": true,
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (d *Client) Status() (*StatusResponse, error) {
	response := new(StatusResponse)
	return response, d.call(response, "status", map[string]interface{}{})
}

func (d *Client) UTXOList(account *string) (*UTXOListResponse, error) {
	response := new(UTXOListResponse)
	return response, d.call(response, "utxo_list", map[string]interface{}{
		"account_id": account,
	})
}

func (d *Client) Get(uri string) (*GetResponse, error) {
	response := new(GetResponse)
	return response, d.call(response, "get", map[string]interface{}{
		"uri":              uri,
		"include_protobuf": true,
	})
}

func (d *Client) FileList() (*FileListResponse, error) {
	response := new(FileListResponse)
	return response, d.call(response, "file_list", map[string]interface{}{
		"include_protobuf": true,
	})
}

func (d *Client) Version() (*VersionResponse, error) {
	response := new(VersionResponse)
	return response, d.call(response, "version", map[string]interface{}{})
}

func (d *Client) Resolve(urls string) (*ResolveResponse, error) {
	response := new(ResolveResponse)
	return response, d.call(response, "resolve", map[string]interface{}{
		"urls":             urls,
		"include_protobuf": true,
	})
}

/*
// use resolve?
func (d *Client) NumClaimsInChannel(channelClaimID string) (uint64, error) {
	response := new(NumClaimsInChannelResponse)
	err := d.call(response, "claim_search", map[string]interface{}{
		"channel_id": channelClaimID,
	})
	if err != nil {
		return 0, err
	} else if response == nil {
		return 0, errors.Err("no response")
	}

	channel, ok := (*response)[uri]
	if !ok {
		return 0, errors.Err("url not in response")
	}
	if channel.Error != nil {
		if strings.Contains(*channel.Error, "cannot be resolved") {
			return 0, nil
		}
		return 0, errors.Err(*channel.Error)
	}
	return *channel.ClaimsInChannel, nil
}
*/
func (d *Client) ClaimSearch(claimName, claimID, txid *string, nout *uint) (*ClaimSearchResponse, error) {
	response := new(ClaimSearchResponse)
	return response, d.call(response, "claim_search", map[string]interface{}{
		"claim_id": claimID,
		"txid":     txid,
		"nout":     nout,
		"name":     claimName,
	})
}
