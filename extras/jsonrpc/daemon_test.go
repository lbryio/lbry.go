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
	got, err := d.StreamCreate("test"+fmt.Sprintf("%d", time.Now().Unix()), "/home/niko/work2/2019-04-11_17-36-25-925698088.png", 14.37, StreamCreateOptions{
		ClaimCreateOptions: ClaimCreateOptions{
			Title:       "This is a Test Title" + fmt.Sprintf("%d", time.Now().Unix()),
			Description: "My Special Description",
			Tags:        []string{"nsfw", "test"},
			Languages:   []string{"en-US", "fr-CH"},
			Locations: []Location{{
				Country:    util.PtrToString("CH"),
				State:      util.PtrToString("Ticino"),
				City:       util.PtrToString("Lugano"),
				PostalCode: util.PtrToString("6900"),
				Latitude:   nil,
				Longitude:  nil,
			}},
			ThumbnailURL: util.PtrToString("https://scrn.storni.info/2019-01-18_16-37-39-098537783.png"),
			AccountID:    nil,
			ClaimAddress: &address,
			Preview:      nil,
		},

		Fee: &Fee{
			FeeCurrency: "LBC",
			FeeAmount:   decimal.NewFromFloat(1.0),
			FeeAddress:  &address,
		},
		Author:             util.PtrToString("Niko"),
		License:            util.PtrToString("FREE"),
		LicenseURL:         nil,
		StreamType:         &StreamTypeImage,
		ReleaseTime:        nil,
		Duration:           nil,
		ImageWidth:         nil,
		ImageHeight:        nil,
		VideoWidth:         nil,
		VideoHeight:        nil,
		Preview:            nil,
		AllowDuplicateName: nil,
		ChannelName:        nil,
		ChannelID:          util.PtrToString("5205b93465014f9f8ae3e7b1e5a7ad46f925163d"),
		ChannelAccountID:   nil,
	})
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_ChannelCreate(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelCreate("@Test", 13.37, ChannelCreateOptions{
		ClaimCreateOptions: ClaimCreateOptions{
			Title:       "Mess with the channels",
			Description: "And you'll get what you deserve",
			Tags:        []string{"we", "got", "tags"},
			Languages:   []string{"en-US"},
			Locations: []Location{{
				Country: util.PtrToString("CH"),
				State:   util.PtrToString("Ticino"),
				City:    util.PtrToString("Lugano"),
			}},
			ThumbnailURL: util.PtrToString("https://scrn.storni.info/2019-04-12_15-43-25-001592625.png"),
		},
		ContactEmail: util.PtrToString("niko@lbry.com"),
		HomepageURL:  util.PtrToString("https://lbry.com"),
		CoverURL:     util.PtrToString("https://scrn.storni.info/2019-04-12_15-43-25-001592625.png"),
	})
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}

func TestClient_ChannelAbandon(t *testing.T) {
	d := NewClient("")
	channelResponse, err := d.ChannelCreate("@TestToDelete", 13.37, ChannelCreateOptions{
		ClaimCreateOptions: ClaimCreateOptions{
			Title:       "Mess with the channels",
			Description: "And you'll get what you deserve",
			Tags:        []string{"we", "got", "tags"},
			Languages:   []string{"en-US"},
			Locations: []Location{{
				Country: util.PtrToString("CH"),
				State:   util.PtrToString("Ticino"),
				City:    util.PtrToString("Lugano"),
			}},
			ThumbnailURL: util.PtrToString("https://scrn.storni.info/2019-04-12_15-43-25-001592625.png"),
		},
		ContactEmail: util.PtrToString("niko@lbry.com"),
		HomepageURL:  util.PtrToString("https://lbry.com"),
		CoverURL:     util.PtrToString("https://scrn.storni.info/2019-04-12_15-43-25-001592625.png"),
	})
	if err != nil {
		t.Error(err)
	}
	txID := channelResponse.Outputs[0].Txid
	nout := channelResponse.Outputs[0].Nout
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
	got, err := d.ClaimSearch(nil, util.PtrToString("d3d84b191b05b1915db3f78150c5d42d172f4c5f"), nil, nil)
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
	got, err := d.Resolve("crashtest")
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

func TestClient_AccountSet(t *testing.T) {
	d := NewClient("")
	accounts, err := d.AccountList()
	if err != nil {
		t.Error(err)
	}
	account := (accounts.LBCRegtest)[0].ID

	got, err := d.AccountSet(account, AccountSettings{ChangeMaxUses: 10000})
	if err != nil {
		t.Error(err)
	}
	log.Infof("%+v", *got)
}
func TestClient_AccountCreate(t *testing.T) {
	d := NewClient("")
	account, err := d.AccountCreate("test@lbry.com", false)
	if err != nil {
		t.Error(err)
	}
	if account.Status != "created" {
		t.Fail()
	}
}
