package jsonrpc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

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
	got, err := d.Publish("test", "/home/niko/test.txt", 14.37, PublishOptions{
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
		ChannelID:        util.PtrToString("0a32af305113435d1cdf4ec61452b9a6dcb74da8"),
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

func TestClient_ChannelNew(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelNew("@Test", 13.37, nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_ClaimAbandon(t *testing.T) {
	d := NewClient("")
	channelResponse, err := d.ChannelNew("@TestToDelete", 13.37, nil)
	if err != nil {
		t.Error(err)
	}
	txID := channelResponse.Output.Txid
	nout := channelResponse.Output.Nout
	time.Sleep(10 * time.Second)
	got, err := d.ClaimAbandon(txID, nout, nil, false)
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
	got, err := d.ClaimList("test")
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_ClaimListMine(t *testing.T) {
	d := NewClient("")
	got, err := d.ClaimListMine(nil, 0, 50)
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
	got, err := d.UTXOList(nil, 0, 50)
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

func TestClient_Commands(t *testing.T) {
	d := NewClient("")
	got, err := d.Commands()
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

func TestClient_NumClaimsInChannel(t *testing.T) {
	d := NewClient("")
	got, err := d.NumClaimsInChannel("@Test#0a32af305113435d1cdf4ec61452b9a6dcb74da8")
	if err != nil {
		t.Error(err)
	}
	log.Infof("%d", got)
}

func TestClient_ClaimShow(t *testing.T) {
	d := NewClient("")
	got, err := d.ClaimShow(util.PtrToString("4742f25e6d51b4b0483d5b8cd82e3ea121dacde9"), nil, nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}
