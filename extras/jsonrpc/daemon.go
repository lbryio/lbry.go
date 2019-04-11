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

	"github.com/fatih/structs"

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

func (d *Client) AccountBalance(account *string) (*AccountBalanceResponse, error) {
	response := new(AccountBalanceResponse)
	return response, d.call(response, "account_balance", map[string]interface{}{
		"account_id": account,
	})
}

func (d *Client) AccountFund(fromAccount string, toAccount string, amount string, outputs uint64) (*AccountFundResponse, error) {
	response := new(AccountFundResponse)
	return response, d.call(response, "account_fund", map[string]interface{}{
		"from_account": fromAccount,
		"to_account":   toAccount,
		"amount":       amount,
		"outputs":      outputs,
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
		"account_id": account,
		"page":       page,
		"page_size":  pageSize,
	})
}

type streamType string

var (
	StreamTypeVideo = streamType("video")
	StreamTypeAudio = streamType("audio")
	StreamTypeImage = streamType("image")
)

type Locations struct {
	Country    *string `json:"country,omitempty"`
	State      *string `json:"state,omitempty"`
	City       *string `json:"city,omitempty"`
	PostalCode *string `json:"code,omitempty"`
	Latitude   *string `json:"latitude,omitempty"`
	Longitude  *string `json:"longitude,omitempty"`
}
type ClaimCreateOptions struct {
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Tags          []string    `json:"tags,omitempty"`
	Languages     []string    `json:"languages"`
	Locations     []Locations `json:"locations,omitempty"`
	ThumbnailURL  *string     `json:"thumbnail_url,omitempty"`
	AccountID     *string     `json:"account_id,omitempty"`
	ClaimAddress  *string     `json:"claim_address,omitempty"`
	ChangeAddress *string     `json:"change_address,omitempty"`
	Preview       *bool       `json:"preview,omitempty"`
}

type ChannelCreateOptions struct {
	ClaimCreateOptions `json:",flatten"`
	ContactEmail       *string `json:"contact_email,omitempty"`
	HomepageURL        *string `json:"homepage_url,omitempty"`
	CoverURL           *string `json:"cover_url,omitempty"`
}

func (d *Client) ChannelCreate(name string, bid float64, options *ChannelCreateOptions) (*PublishResponse, error) {
	response := new(PublishResponse)
	args := struct {
		Name                  string `json:"name"`
		Bid                   string `json:"bid"`
		FilePath              string `json:"file_path,omitempty"`
		*ChannelCreateOptions `json:",flatten"`
	}{
		Name:                 name,
		Bid:                  fmt.Sprintf("%.6f", bid),
		ChannelCreateOptions: options,
	}
	structs.DefaultTagName = "json"
	return response, d.call(response, "channel_create", structs.Map(args))
}

type StreamCreateOptions struct {
	ClaimCreateOptions `json:",flatten"`
	Fee                *Fee        `json:",omitempty,flatten"`
	Author             *string     `json:"author,omitempty"`
	License            *string     `json:"license,omitempty"`
	LicenseURL         *string     `json:"license_url,omitempty"`
	StreamType         *streamType `json:"stream_type,omitempty"`
	ReleaseTime        *int        `json:"release_time,omitempty"`
	Duration           *int        `json:"duration,omitempty"`
	ImageWidth         *int        `json:"image_width,omitempty"`
	ImageHeight        *int        `json:"image_height,omitempty"`
	VideoWidth         *int        `json:"video_width,omitempty"`
	VideoHeight        *int        `json:"video_height,omitempty"`
	Preview            *string     `json:"preview,omitempty"`
	AllowDuplicateName *bool       `json:"allow_duplicate_name,omitempty"`
	ChannelName        *string     `json:"channel_name,omitempty"`
	ChannelID          *string     `json:"channel_id,omitempty"`
	ChannelAccountID   *string     `json:"channel_account_id,omitempty"`
}

func (d *Client) StreamCreate(name, filePath string, bid float64, options StreamCreateOptions) (*TransactionSummary, error) {
	response := new(TransactionSummary)
	args := struct {
		Name                 string `json:"name"`
		Bid                  string `json:"bid"`
		FilePath             string `json:"file_path,omitempty"`
		*StreamCreateOptions `json:",flatten"`
	}{
		Name:                name,
		FilePath:            filePath,
		Bid:                 fmt.Sprintf("%.6f", bid),
		StreamCreateOptions: &options,
	}
	structs.DefaultTagName = "json"
	return response, d.call(response, "stream_create", structs.Map(args))
}

func (d *Client) StreamAbandon(txID string, nOut uint64, accountID *string, blocking bool) (*ClaimAbandonResponse, error) {
	response := new(ClaimAbandonResponse)
	err := d.call(response, "claim_abandon", map[string]interface{}{
		"txid":       txID,
		"nout":       nOut,
		"account_id": accountID,
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
	Name                 *string `json:"name"`
	FilePath             *string `json:"file_path,omitempty"`
	Bid                  *string `json:"bid"`
	*StreamCreateOptions `json:",flatten"`
}

func (d *Client) StreamUpdate(claimID string, options StreamUpdateOptions) (*PublishResponse, error) {
	response := new(PublishResponse)
	args := struct {
		ClaimID              string `json:"claim_id"`
		FilePath             string `json:"file_path,omitempty"`
		Bid                  string `json:"bid"`
		*StreamUpdateOptions `json:",flatten"`
	}{
		ClaimID:             claimID,
		StreamUpdateOptions: &options,
	}
	structs.DefaultTagName = "json"
	return response, d.call(response, "stream_create", structs.Map(args))
}

func (d *Client) ChannelAbandon(txID string, nOut uint64, accountID *string, blocking bool) (*ClaimAbandonResponse, error) {
	response := new(ClaimAbandonResponse)
	err := d.call(response, "claim_abandon", map[string]interface{}{
		"txid":       txID,
		"nout":       nOut,
		"account_id": accountID,
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

func (d *Client) ClaimList(account *string, page uint64, pageSize uint64) (*ClaimListMineResponse, error) {
	if page == 0 {
		return nil, errors.Err("pages start from 1")
	}
	response := new(ClaimListMineResponse)
	err := d.call(response, "claim_list", map[string]interface{}{
		"account_id": account,
		"page":       page,
		"page_size":  pageSize,
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

func (d *Client) Version() (*VersionResponse, error) {
	response := new(VersionResponse)
	return response, d.call(response, "version", map[string]interface{}{})
}

func (d *Client) Resolve(urls string) (*ResolveResponse, error) {
	response := new(ResolveResponse)
	return response, d.call(response, "resolve", map[string]interface{}{
		"urls": urls,
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
