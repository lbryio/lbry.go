package jsonrpc

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lbryio/lbry.go/extras/util"
)

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}

func TestClient_AccountFund(t *testing.T) {
	d := NewClient("")
	accounts, err := d.AccountList()
	if err != nil {
		t.Error(err)
		return
	}
	account := (accounts.LBCRegtest)[0].ID
	balanceString, err := d.AccountBalance(&account)
	if err != nil {
		t.Error(err)
		return
	}
	balance, err := strconv.ParseFloat(string(*balanceString), 64)
	if err != nil {
		t.Error(err)
		return
	}
	got, err := d.AccountFund(account, account, fmt.Sprintf("%f", balance/2.0), 40)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_AccountList(t *testing.T) {
	d := NewClient("")
	got, err := d.AccountList()
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_SingleAccountList(t *testing.T) {
	d := NewClient("")
	createdAccount, err := d.AccountCreate("test"+fmt.Sprintf("%d", time.Now().Unix())+"@lbry.com", false)
	if err != nil {
		t.Fatal(err)
		return
	}
	account, err := d.SingleAccountList(createdAccount.ID)
	if err != nil {
		t.Fatal(err)
		return
	}
	prettyPrint(*account)
}

func TestClient_AccountBalance(t *testing.T) {
	d := NewClient("")
	got, err := d.AccountBalance(nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_AddressUnused(t *testing.T) {
	d := NewClient("")
	got, err := d.AddressUnused(nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_ChannelList(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelList(nil, 1, 50)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_StreamCreate(t *testing.T) {
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
	d := NewClient("")
	addressResponse, err := d.AddressUnused(nil)
	if err != nil {
		t.Error(err)
		return
	}
	address := string(*addressResponse)
	got, err := d.StreamCreate("test"+fmt.Sprintf("%d", time.Now().Unix()), "/home/niko/Downloads/IMG_20171012_205120.jpg", 14.37, StreamCreateOptions{
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
		ReleaseTime:        nil,
		Duration:           nil,
		Preview:            nil,
		AllowDuplicateName: nil,
		ChannelName:        nil,
		ChannelID:          util.PtrToString("253ca42da47ad8a430e18d52860cb499c50eff25"),
		ChannelAccountID:   nil,
	})
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_ChannelCreate(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelCreate("@Test"+fmt.Sprintf("%d", time.Now().Unix()), 13.37, ChannelCreateOptions{
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
		Email:      util.PtrToString("niko@lbry.com"),
		WebsiteURL: util.PtrToString("https://lbry.com"),
		CoverURL:   util.PtrToString("https://scrn.storni.info/2019-04-12_15-43-25-001592625.png"),
	})
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_ChannelUpdate(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelUpdate("2a8b6d061c5ecb2515f1dd7e04729e9fafac660d", ChannelUpdateOptions{
		ClearLanguages: util.PtrToBool(true),
		ClearLocations: util.PtrToBool(true),
		ClearTags:      util.PtrToBool(true),
		ChannelCreateOptions: ChannelCreateOptions{
			ClaimCreateOptions: ClaimCreateOptions{
				Title:       "Mess with the channels",
				Description: "And you'll get what you deserve",
				Tags:        []string{"we", "got", "more", "tags"},
				Languages:   []string{"en-US"},
				Locations: []Location{{
					Country: util.PtrToString("CH"),
					State:   util.PtrToString("Ticino"),
					City:    util.PtrToString("Lugano"),
				}},
				ThumbnailURL: util.PtrToString("https://scrn.storni.info/2019-04-12_15-43-25-001592625.png"),
			},
			Email:      util.PtrToString("niko@lbry.com"),
			WebsiteURL: util.PtrToString("https://lbry.com"),
			CoverURL:   util.PtrToString("https://scrn.storni.info/2019-04-12_15-43-25-001592625.png"),
		}})
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_ChannelAbandon(t *testing.T) {
	d := NewClient("")
	channelName := "@TestToDelete" + fmt.Sprintf("%d", time.Now().Unix())
	channelResponse, err := d.ChannelCreate(channelName, 13.37, ChannelCreateOptions{
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
		Email:      util.PtrToString("niko@lbry.com"),
		WebsiteURL: util.PtrToString("https://lbry.com"),
		CoverURL:   util.PtrToString("https://scrn.storni.info/2019-04-12_15-43-25-001592625.png"),
	})
	if err != nil {
		t.Error(err)
		return
	}
	txID := channelResponse.Outputs[0].Txid
	nout := channelResponse.Outputs[0].Nout
	time.Sleep(10 * time.Second)
	got, err := d.ChannelAbandon(txID, nout, nil, false)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_AddressList(t *testing.T) {
	d := NewClient("")
	got, err := d.AddressList(nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_ClaimList(t *testing.T) {
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
	d := NewClient("")
	got, err := d.ClaimList(nil, 1, 10)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_ClaimSearch(t *testing.T) {
	d := NewClient("")
	got, err := d.ClaimSearch(nil, util.PtrToString("2a8b6d061c5ecb2515f1dd7e04729e9fafac660d"), nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_Status(t *testing.T) {
	d := NewClient("")
	got, err := d.Status()
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_UTXOList(t *testing.T) {
	d := NewClient("")
	got, err := d.UTXOList(nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_Version(t *testing.T) {
	d := NewClient("")
	got, err := d.Version()
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_GetFile(t *testing.T) {
	d := NewClient("")
	got, err := d.Get("lbry://test1559058649")
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_FileList(t *testing.T) {
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
	d := NewClient("")
	got, err := d.FileList()
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_Resolve(t *testing.T) {
	d := NewClient("")
	got, err := d.Resolve("test1559058649")
	if err != nil {
		t.Error(err)
		return
	}
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_AccountSet(t *testing.T) {
	d := NewClient("")
	accounts, err := d.AccountList()
	if err != nil {
		t.Error(err)
		return
	}
	account := (accounts.LBCRegtest)[0].ID

	got, err := d.AccountSet(account, AccountSettings{ChangeMaxUses: 10000})
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_AccountCreate(t *testing.T) {
	d := NewClient("")
	name := "test" + fmt.Sprintf("%d", time.Now().Unix()) + "@lbry.com"
	account, err := d.AccountCreate(name, false)
	if err != nil {
		t.Fatal(err)
		return
	}
	if account.Name != name {
		t.Errorf("account name mismatch, expected %q, got %q", name, account.Name)
		return
	}
	prettyPrint(*account)
}

func TestClient_AccountRemove(t *testing.T) {
	d := NewClient("")
	createdAccount, err := d.AccountCreate("test"+fmt.Sprintf("%d", time.Now().Unix())+"@lbry.com", false)
	if err != nil {
		t.Fatal(err)
		return
	}
	removedAccount, err := d.AccountRemove(createdAccount.ID)
	if err != nil {
		t.Error(err)
		return
	}
	if removedAccount.ID != createdAccount.ID {
		t.Error("accounts IDs mismatch")
	}

	account, err := d.SingleAccountList(createdAccount.ID)
	if err != nil {
		if strings.HasPrefix(err.Error(), "Error in daemon: Couldn't find account") {
			prettyPrint(*removedAccount)
			return
		}

		t.Error(err)
		return
	}
	t.Error("account was not removed")
	prettyPrint(*account)
}
