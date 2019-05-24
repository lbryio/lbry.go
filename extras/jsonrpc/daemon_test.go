package jsonrpc

import (
	"encoding/json"
	"fmt"
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
	got, err := d.AccountFund(account, account, fmt.Sprintf("%f", balance/2.0), 40)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_AccountList(t *testing.T) {
	d := NewClient("")
	got, err := d.AccountList()
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_SingleAccountList(t *testing.T) {
	d := NewClient("")
	createdAccount, err := d.AccountCreate("test"+fmt.Sprintf("%d", time.Now().Unix())+"@lbry.com", false)
	if err != nil {
		t.Fatal(err)
	}
	account, err := d.SingleAccountList(createdAccount.ID)
	if err != nil {
		t.Fatal(err)
	}
	prettyPrint(*account)
}

func TestClient_AccountBalance(t *testing.T) {
	d := NewClient("")
	got, err := d.AccountBalance(nil)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_AddressUnused(t *testing.T) {
	d := NewClient("")
	got, err := d.AddressUnused(nil)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_ChannelList(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelList(nil, 1, 50)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_StreamCreate(t *testing.T) {
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
		ReleaseTime:        nil,
		Duration:           nil,
		Preview:            nil,
		AllowDuplicateName: nil,
		ChannelName:        nil,
		ChannelID:          util.PtrToString("2e28aa6dbd41f959893907841f4e40d0ecb0ede9"),
		ChannelAccountID:   nil,
	})
	if err != nil {
		t.Error(err)
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
	}
	prettyPrint(*got)
}

func TestClient_ChannelUpdate(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelUpdate("709868122fe3560a3929d6d63bdbc792d8306a6c", ChannelUpdateOptions{
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
	}
	txID := channelResponse.Outputs[0].Txid
	nout := channelResponse.Outputs[0].Nout
	time.Sleep(10 * time.Second)
	got, err := d.ChannelAbandon(txID, nout, nil, false)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_AddressList(t *testing.T) {
	d := NewClient("")
	got, err := d.AddressList(nil)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_ClaimList(t *testing.T) {
	d := NewClient("")
	got, err := d.ClaimList(nil, 1, 10)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_ClaimSearch(t *testing.T) {
	d := NewClient("")
	got, err := d.ClaimSearch(nil, util.PtrToString("1b2b530dfcef9885354f8f41190c8f678da5414e"), nil, nil)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_Status(t *testing.T) {
	d := NewClient("")
	got, err := d.Status()
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_UTXOList(t *testing.T) {
	d := NewClient("")
	got, err := d.UTXOList(nil)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_Version(t *testing.T) {
	d := NewClient("")
	got, err := d.Version()
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_GetFile(t *testing.T) {
	d := NewClient("")
	got, err := d.Get("lbry://test1555965264")
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_FileList(t *testing.T) {
	d := NewClient("")
	got, err := d.FileList()
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
}

func TestClient_Resolve(t *testing.T) {
	d := NewClient("")
	got, err := d.Resolve("test1555965264")
	if err != nil {
		t.Error(err)
	}
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*got)
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
	prettyPrint(*got)
}

func TestClient_AccountCreate(t *testing.T) {
	d := NewClient("")
	name := "test" + fmt.Sprintf("%d", time.Now().Unix()) + "@lbry.com"
	account, err := d.AccountCreate(name, false)
	if err != nil {
		t.Fatal(err)
	}
	if account.Name != name {
		t.Errorf("account name mismatch, expected %q, got %q", name, account.Name)
	}
	prettyPrint(*account)
}

func TestClient_AccountRemove(t *testing.T) {
	d := NewClient("")
	createdAccount, err := d.AccountCreate("test"+fmt.Sprintf("%d", time.Now().Unix())+"@lbry.com", false)
	if err != nil {
		t.Fatal(err)
	}
	removedAccount, err := d.AccountRemove(createdAccount.ID)
	if err != nil {
		t.Error(err)
	}
	if removedAccount.ID != createdAccount.ID {
		t.Error("accounts IDs mismatch")
	}

	account, err := d.SingleAccountList(createdAccount.ID)
	if !strings.HasPrefix(err.Error(), "Error in daemon: Couldn't find account") {
		t.Error("account was not removed")
	}
	fmt.Println(err.Error())
	prettyPrint(*account)
}
