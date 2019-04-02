package jsonrpc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lbryio/lbry.go/extras/util"
	log "github.com/sirupsen/logrus"
)

func TestClient_AccountList(t *testing.T) {
	d := NewClient("")
	got, err := d.AccountList()
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_AccountBalance(t *testing.T) {
	d := NewClient("")
	got, err := d.AccountBalance(nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%s", *got)
}

func TestClient_AddressUnused(t *testing.T) {
	d := NewClient("")
	got, err := d.AddressUnused(nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%s", *got)
}

func TestClient_ChannelList(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelList(nil, 1, 50)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_Publish(t *testing.T) {
	d := NewClient("")
	addressResponse, err := d.AddressUnused(nil)
	if err != nil {
		t.Error(err)
	}
	address := string(*addressResponse)
	got, err := d.StreamCreate("test"+string(time.Now().Unix()), "/home/niko/work/allClaims.txt", 14.37, StreamCreateOptions{
		ClaimCreateOptions: &ClaimCreateOptions{
			Title:       "This is a Test Title" + time.Now().String(),
			Description: "My Special Description",
			Tags:        []string{"nsfw", "test"},
			Languages:   []string{"en-US", "fr-CH"},
			Locations: &Locations{
				Country:    util.PtrToString("CH"),
				State:      util.PtrToString("Ticino"),
				City:       util.PtrToString("Lugano"),
				PostalCode: util.PtrToString("6900"),
				Latitude:   nil,
				Longitude:  nil,
			},
			ThumbnailURL:  util.PtrToString("https://scrn.storni.info/2019-01-18_16-37-39-098537783.png"),
			AccountID:     nil,
			ClaimAddress:  &address,
			ChangeAddress: &address,
			Preview:       nil,
		},
		Fee: &Fee{
			Currency: "LBC",
			Amount:   decimal.NewFromFloat(1.0),
			Address:  &address,
		},
		Author:             util.PtrToString("Niko"),
		License:            util.PtrToString("FREE"),
		LicenseURL:         nil,
		StreamType:         &StreamTypeImage,
		ReleaseTime:        nil,
		Duration:           nil,
		ImageWidth:         nil,
		ImageHeigth:        nil,
		VideoWidth:         nil,
		VideoHeight:        nil,
		Preview:            nil,
		AllowDuplicateName: nil,
		ChannelName:        nil,
		ChannelID:          util.PtrToString("bda0520bff61e4a70c966d7298e6b89107cf8bed"),
		ChannelAccountID:   nil,
	})
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_ChannelCreate(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelCreate("@Test", 13.37, nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_ChannelAbandon(t *testing.T) {
	d := NewClient("")
	channelResponse, err := d.ChannelCreate("@TestToDelete", 13.37, nil)
	if err != nil {
		t.Error(err)
	}
	txID := channelResponse.Output.Txid
	nout := channelResponse.Output.Nout
	time.Sleep(10 * time.Second)
	got, err := d.ChannelAbandon(txID, nout, nil, false)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_AddressList(t *testing.T) {
	d := NewClient("")
	got, err := d.AddressList(nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_ClaimList(t *testing.T) {
	d := NewClient("")
	got, err := d.ClaimList(nil, 1, 10)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_ClaimSearch(t *testing.T) {
	d := NewClient("")
	got, err := d.ClaimSearch(nil, util.PtrToString("4742f25e6d51b4b0483d5b8cd82e3ea121dacde9"), nil, nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_Status(t *testing.T) {
	d := NewClient("")
	got, err := d.Status()
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_UTXOList(t *testing.T) {
	d := NewClient("")
	got, err := d.UTXOList(nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_Version(t *testing.T) {
	d := NewClient("")
	got, err := d.Version()
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_Resolve(t *testing.T) {
	d := NewClient("")
	got, err := d.Resolve("test")
	if err != nil {
		t.Error(err)
	}
	b, err := json.Marshal(*got)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%s", b)
}

func TestClient_AccountFund(t *testing.T) {
	d := NewClient("")
	accounts, err := d.AccountList()
	if err != nil {
		t.Error(err)
	}
	account := (accounts.LBCRegtest)[0].ID
	balanceString, err := d.AccountBalance(&account)
	if err != nil {
		t.Error(err)
	}
	balance, err := strconv.ParseFloat(string(*balanceString), 64)
	if err != nil {
		t.Error(err)
	}
	got, err := d.AccountFund(account, account, fmt.Sprintf("%f", balance-0.1), 40)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}
