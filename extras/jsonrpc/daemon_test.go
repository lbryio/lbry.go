package jsonrpc

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/lbryio/lbry.go/extras/util"

	"github.com/shopspring/decimal"
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

func TestClient_AccountFund(t *testing.T) {
	d := NewClient("")
	accounts, err := d.AccountList()
	if err != nil {
		t.Error(err)
	}
	account := (*accounts.LBCRegtest)[0].ID
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
	got, err := d.ChannelList(nil, 0, 50)
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
	got, err := d.Publish("test", "/home/niko/test.txt", 13.37, PublishOptions{
		Metadata: &Metadata{
			Fee: &Fee{
				Currency: "LBC",
				Amount:   decimal.NewFromFloat(1.0),
				Address:  &address,
			},
			Title:       "This is a Test Title",
			Description: "My Special Description",
			Author:      "Niko",
			Language:    "en",
			License:     "FREEEEE",
			LicenseURL:  nil,
			Thumbnail:   util.PtrToString("https://scrn.storni.info/2019-01-18_16-37-39-098537783.png"),
			Preview:     nil,
			NSFW:        false,
			Sources:     nil,
		},
		ChannelName:      nil,
		ChannelID:        nil,
		ChannelAccountID: nil,
		AccountID:        nil,
		ClaimAddress:     &address,
		ChangeAddress:    &address,
	})
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}
