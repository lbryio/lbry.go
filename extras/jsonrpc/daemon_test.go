package jsonrpc

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/lbryio/lbry.go/v2/extras/util"
)

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	code := m.Run()
	os.Exit(code)
}

func TestClient_AccountFund(t *testing.T) {
	d := NewClient("")
	accounts, err := d.AccountList(1, 20)
	if err != nil {
		t.Error(err)
		return
	}
	account := (accounts.Items)[0].ID
	balanceString, err := d.AccountBalance(&account)
	if err != nil {
		t.Error(err)
		return
	}
	balance, err := strconv.ParseFloat(balanceString.Available.String(), 64)
	if err != nil {
		t.Error(err)
		return
	}
	got, err := d.AccountFund(account, account, fmt.Sprintf("%f", balance/2.0), 40, false)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_AccountList(t *testing.T) {
	d := NewClient("")
	got, err := d.AccountList(1, 20)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_SingleAccountList(t *testing.T) {
	d := NewClient("")
	name := "test" + fmt.Sprintf("%d", rand.Int()) + "@lbry.com"
	createdAccount, err := d.AccountCreate(name, false)
	if err != nil {
		t.Fatal(err)
	}
	account, err := d.SingleAccountList(createdAccount.ID)
	if err != nil {
		t.Error(err)
	}
	prettyPrint(*createdAccount)
	prettyPrint(*account)
	if err != nil {
		t.Fatal(err)
	}
	if account.Items[0].Name != name {
		t.Fatalf("account name mismatch: %v != %v", account.Items[0].Name, name)
	}
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
	got, err := d.ChannelList(nil, 1, 50, nil, false)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

var channelID string

func TestClient_ChannelCreate(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelCreate("@Test"+fmt.Sprintf("%d", time.Now().Unix()), 1.337, ChannelCreateOptions{
		ClaimCreateOptions: ClaimCreateOptions{
			Title:       util.PtrToString("Mess with the channels"),
			Description: util.PtrToString("And you'll get what you deserve"),
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
	channelID = got.Outputs[0].ClaimID
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
	f, e := os.OpenFile("/tmp/test.txt", os.O_RDONLY|os.O_CREATE, 0666)
	if e != nil {
		t.Error(e)
		return
	}

	_, _ = f.WriteString("test")
	got, err := d.StreamCreate("test"+fmt.Sprintf("%d", time.Now().Unix()), "/tmp/test.txt", 1.437, StreamCreateOptions{
		ClaimCreateOptions: ClaimCreateOptions{
			Title:       util.PtrToString("This is a Test Title" + fmt.Sprintf("%d", time.Now().Unix())),
			Description: util.PtrToString("My Special Description"),
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
		ChannelID:          util.PtrToString(channelID),
		ChannelAccountID:   nil,
	})
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_ChannelUpdate(t *testing.T) {
	d := NewClient("")
	got, err := d.ChannelUpdate(channelID, ChannelUpdateOptions{
		Bid:            util.PtrToString("0.01"),
		ClearLanguages: util.PtrToBool(true),
		ClearLocations: util.PtrToBool(true),
		ClearTags:      util.PtrToBool(true),
		ChannelCreateOptions: ChannelCreateOptions{
			ClaimCreateOptions: ClaimCreateOptions{
				Title:       util.PtrToString("Mess with the channels"),
				Description: util.PtrToString("And you'll get what you deserve"),
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
			Title:       util.PtrToString("Mess with the channels"),
			Description: util.PtrToString("And you'll get what you deserve"),
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
	got, err := d.AddressList(nil, nil, 1, 20)
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

func TestClient_StreamList(t *testing.T) {
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
	d := NewClient("")
	got, err := d.StreamList(nil, 1, 20, false)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_TransactionList(t *testing.T) {
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
	d := NewClient("")
	got, err := d.TransactionList(nil, nil, 1, 20)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_SupportTest(t *testing.T) {
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
	d := NewClient("")
	got, err := d.ChannelCreate("@Test"+fmt.Sprintf("%d", time.Now().Unix()), 13.37, ChannelCreateOptions{
		ClaimCreateOptions: ClaimCreateOptions{
			Title:       util.PtrToString("Mess with the channels"),
			Description: util.PtrToString("And you'll get what you deserve"),
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
	time.Sleep(10 * time.Second)
	got2, err := d.SupportCreate(got.Outputs[0].ClaimID, "1.0", util.PtrToBool(true), nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got2)

	got3, err := d.SupportList(nil, 1, 10)
	if err != nil {
		t.Error(err)
		return
	}
	found := false
	for _, support := range got3.Items {
		if support.ClaimID == got.Outputs[0].ClaimID {
			found = true
		}
	}
	if !found {
		t.Error(errors.Err("support not found"))
		return
	}
	prettyPrint(*got3)
	got4, err := d.SupportAbandon(util.PtrToString(got.Outputs[0].ClaimID), nil, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got4)
}

func TestClient_TxoSpendTest(t *testing.T) {
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
	d := NewClient("")
	got, err := d.ChannelCreate("@Test"+fmt.Sprintf("%d", time.Now().Unix()), 13.37, ChannelCreateOptions{
		ClaimCreateOptions: ClaimCreateOptions{
			Title:       util.PtrToString("Mess with the channels"),
			Description: util.PtrToString("And you'll get what you deserve"),
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
	time.Sleep(10 * time.Second)
	got2, err := d.SupportCreate(got.Outputs[0].ClaimID, "1.0", util.PtrToBool(true), nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got2)

	got3, err := d.SupportList(nil, 1, 10)
	if err != nil {
		t.Error(err)
		return
	}
	found := false
	for _, support := range got3.Items {
		if support.ClaimID == got.Outputs[0].ClaimID {
			found = true
		}
	}
	if !found {
		t.Error(errors.Err("support not found"))
		return
	}
	prettyPrint(*got3)
	got4, err := d.TxoSpend(util.PtrToString("support"), util.PtrToString(got.Outputs[0].ClaimID), nil, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got4)
	time.Sleep(10 * time.Second)
	got3, err = d.SupportList(nil, 1, 10)
	if err != nil {
		t.Error(err)
		return
	}
	found = false
	for _, support := range got3.Items {
		if support.ClaimID == got.Outputs[0].ClaimID {
			found = true
		}
	}
	if found {
		t.Error(errors.Err("support found even though it should have been abandoned"))
		return
	}
	prettyPrint(*got3)
	got4, err = d.TxoSpend(util.PtrToString("channel"), util.PtrToString(got.Outputs[0].ClaimID), nil, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got4)
	time.Sleep(10 * time.Second)

	got5, err := d.ClaimList(nil, 1, 50)
	if err != nil {
		t.Error(err)
		return
	}
	for _, claim := range got5.Claims {
		if claim.ClaimID == got.Outputs[0].ClaimID {
			t.Error(errors.Err("claim found even though it should have been abandoned"))
			return
		}
	}
	prettyPrint(*got5)
}

func TestClient_ClaimSearch(t *testing.T) {
	d := NewClient("")
	got, err := d.ClaimSearch(nil, util.PtrToString(channelID), nil, nil, 1, 20)
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
	got, err := d.UTXOList(nil, 1, 20)
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
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
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
	got, err := d.FileList(1, 20)
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_Resolve(t *testing.T) {
	_ = os.Setenv("BLOCKCHAIN_NAME", "lbrycrd_regtest")
	d := NewClient("")
	got, err := d.Resolve("test1559058649")
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_AccountSet(t *testing.T) {
	d := NewClient("")
	accounts, err := d.AccountList(1, 20)
	if err != nil {
		t.Error(err)
		return
	}
	account := (accounts.Items)[0].ID

	got, err := d.AccountSet(account, AccountSettings{ChangeMaxUses: util.PtrToInt(10000)})
	if err != nil {
		t.Error(err)
		return
	}
	prettyPrint(*got)
}

func TestClient_AccountCreate(t *testing.T) {
	d := NewClient("")
	name := "lbry#user#id:" + fmt.Sprintf("%d", rand.Int())
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

func TestClient_AccountAdd(t *testing.T) {
	d := NewClient("")
	name := "test" + fmt.Sprintf("%d", time.Now().Unix()) + "@lbry.com"
	pubKey := "tpubDA9GDAntyJu4hD3wU7175p7CuV6DWbYXfyb2HedBA3yuBp9HZ4n3QE4Ex6RHCSiEuVp2nKAL1Lzf2ZLo9ApaFgNaJjG6Xo1wB3iEeVbrDZp"
	account, err := d.AccountAdd(name, nil, nil, &pubKey, util.PtrToBool(true), nil)
	if err != nil {
		t.Fatal(err)
		return
	}
	if account.Name != name {
		t.Errorf("account name mismatch, expected %q, got %q", name, account.Name)
		return
	}
	if account.PublicKey != pubKey {
		t.Errorf("public key mismatch, expected %q, got %q", name, account.Name)
		return
	}
	prettyPrint(*account)
}

func TestClient_AccountRemove(t *testing.T) {
	d := NewClient("")
	name := "lbry#user#id:" + fmt.Sprintf("%d", rand.Int())
	createdAccount, err := d.AccountCreate(name, false)
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
		if strings.Contains(err.Error(), "Couldn't find account:") {
			prettyPrint(*removedAccount)
			return
		}
		t.Fatal(err)
	}
	t.Error("account was not removed")
	prettyPrint(*account)
}

func TestClient_ChannelExport(t *testing.T) {
	d := NewClient("")
	response, err := d.ChannelExport(channelID, nil, nil)
	if err != nil {
		t.Error(err)
	}
	if response == nil || len(*response) == 0 {
		t.Error("nothing returned!")
	}
	t.Log("Export:", *response)
}

func TestClient_ChannelImport(t *testing.T) {
	d := NewClient("")

	// A channel created just for automated testing purposes
	channelName := "@LbryAutomatedTestChannel"
	channelkey := "7943FWPBHZES4dUcMXSpDYwoM5a2tsyJT1R8V54QoUhekGcqmeH3hbzDXoLLQ8" +
		"oKkfb99PgGK5efrZeYqaxg4X5XRJMJ6gKC8hqKcnwhYkmKDXmoBDNgd2ccZ9jhP8z" +
		"HG3NJorAN9Hh4XMyBc5goBLZYYvC9MYvBmT3Fcteb5saqMvmQxFURv74NqXLQZC1t" +
		"p6iRZKfTj77Pd5gsBsCYAbVmCqzbm5m1hHkUmfFEZVGcQNTYCDwZn543xSMYvSPnJ" +
		"zt8tRYCJWaPdj713uENZZMo3gxuAMb1NwSnx8tbwETp7WPkpFLL6HZ9jKpB8BURHM" +
		"F1RFD1PRyqbC6YezPyPQ2oninKKHdBduvXZG5KF2G2Q3ixsuE2ntifBBo1f5PotRk" +
		"UanXKEafWxvXAayJjpsmZ4bFt7n6Xg4438WZXBiZKCPobLJAiHfe72n618kE6PCNU" +
		"77cyU5Rk8J3CuY6QzZPzwuiXz2GLfkUMCYd9jGT6g53XbE6SwCsmGnd9NJkBAaJf5" +
		"1FAYRURrhHnp79PAoHftEWtZEuU8MCPMdSRjzxYMRS4ScUzg5viDMTAkE8frsfCVZ" +
		"hxsFwGUyNNno8eiqrrYmpbJGEwwK3S4437JboAUEFPdMNn8zNQWZcLLVrK9KyQeKM" +
		"XpKkf4zJV6sZJ7gBMpzvPL18ULEgXTy7VsNBKmsfC1rM4WVG9ri1UixEcLDS79foC" +
		"Jb3FnSr1T4MRKESeN3W"
	response, err := d.ChannelImport(channelkey, nil)
	if err != nil {
		t.Error(err)
	}
	channels, err := d.ChannelList(nil, 1, 50, nil, false)
	if err != nil {
		t.Error(err)
	}
	seen := false
	for _, c := range channels.Items {
		if c.Name == channelName {
			seen = true
		}
	}
	if !seen {
		t.Error("couldn't find imported channel")
	}
	t.Log("Response:", *response)
}

func TestClient_ChannelImportWithWalletID(t *testing.T) {
	d := NewClient("")

	id := "lbry#wallet#id:" + fmt.Sprintf("%d", rand.Int())
	wallet, err := d.WalletCreate(id, nil)
	if err != nil {
		t.Error(err)
	}
	// A channel created just for automated testing purposes
	channelName := "@LbryAutomatedTestChannel"
	channelKey := "7943FWPBHZES4dUcMXSpDYwoM5a2tsyJT1R8V54QoUhekGcqmeH3hbzDXoLLQ8" +
		"oKkfb99PgGK5efrZeYqaxg4X5XRJMJ6gKC8hqKcnwhYkmKDXmoBDNgd2ccZ9jhP8z" +
		"HG3NJorAN9Hh4XMyBc5goBLZYYvC9MYvBmT3Fcteb5saqMvmQxFURv74NqXLQZC1t" +
		"p6iRZKfTj77Pd5gsBsCYAbVmCqzbm5m1hHkUmfFEZVGcQNTYCDwZn543xSMYvSPnJ" +
		"zt8tRYCJWaPdj713uENZZMo3gxuAMb1NwSnx8tbwETp7WPkpFLL6HZ9jKpB8BURHM" +
		"F1RFD1PRyqbC6YezPyPQ2oninKKHdBduvXZG5KF2G2Q3ixsuE2ntifBBo1f5PotRk" +
		"UanXKEafWxvXAayJjpsmZ4bFt7n6Xg4438WZXBiZKCPobLJAiHfe72n618kE6PCNU" +
		"77cyU5Rk8J3CuY6QzZPzwuiXz2GLfkUMCYd9jGT6g53XbE6SwCsmGnd9NJkBAaJf5" +
		"1FAYRURrhHnp79PAoHftEWtZEuU8MCPMdSRjzxYMRS4ScUzg5viDMTAkE8frsfCVZ" +
		"hxsFwGUyNNno8eiqrrYmpbJGEwwK3S4437JboAUEFPdMNn8zNQWZcLLVrK9KyQeKM" +
		"XpKkf4zJV6sZJ7gBMpzvPL18ULEgXTy7VsNBKmsfC1rM4WVG9ri1UixEcLDS79foC" +
		"Jb3FnSr1T4MRKESeN3W"
	response, err := d.ChannelImport(channelKey, &wallet.ID)
	if err != nil {
		t.Error(err)
	}
	channels, err := d.ChannelList(nil, 1, 50, &wallet.ID, false)
	if err != nil {
		t.Error(err)
	}
	seen := false
	for _, c := range channels.Items {
		if c.Name == channelName {
			seen = true
		}
	}
	if !seen {
		t.Error("couldn't find imported channel")
	}
	t.Log("Response:", *response)
}

func TestClient_WalletCreate(t *testing.T) {
	d := NewClient("")

	id := "lbry#wallet#id:" + fmt.Sprintf("%d", rand.Int())
	wallet, err := d.WalletCreate(id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if wallet.ID != id {
		prettyPrint(*wallet)
		t.Fatalf("wallet ID mismatch, expected %q, got %q", id, wallet.Name)
	}
}

func TestClient_WalletCreateWithOpts(t *testing.T) {
	d := NewClient("")

	id := "lbry#wallet#id:" + fmt.Sprintf("%d", rand.Int())
	wallet, err := d.WalletCreate(id, &WalletCreateOpts{CreateAccount: true, SingleKey: true})
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := d.AccountListForWallet(id)
	if err != nil {
		t.Fatal(err)
	}
	prettyPrint(wallet)
	prettyPrint(accounts)
	if accounts.Items[0].Name == "" {
		t.Fatalf("account name is empty")
	}
}

func TestClient_WalletList(t *testing.T) {
	d := NewClient("")

	id := "lbry#wallet#id:" + fmt.Sprintf("%d", rand.Int())
	wList, err := d.WalletList(id, 1, 20)
	if err == nil {
		t.Fatalf("wallet %v was unexpectedly found", id)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("Couldn't find wallet: %v.", id)) {
		t.Fatal(err)
	}

	_, err = d.WalletCreate(id, &WalletCreateOpts{CreateAccount: true, SingleKey: true})
	if err != nil {
		t.Fatal(err)
	}

	wList, err = d.WalletList(id, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(wList.Items) < 1 {
		t.Fatal("wallet list is empty")
	}
	if (wList.Items)[0].ID != id {
		t.Fatalf("wallet ID mismatch, expected %q, got %q", id, (wList.Items)[0].ID)
	}
}

func TestClient_WalletRemoveWalletAdd(t *testing.T) {
	d := NewClient("")

	id := "lbry#wallet#id:" + fmt.Sprintf("%d", rand.Int())
	wallet, err := d.WalletCreate(id, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.WalletRemove(id)
	if err != nil {
		t.Fatal(err)
	}

	addedWallet, err := d.WalletAdd(id)
	if err != nil {
		t.Fatal(err)
	}
	if addedWallet.ID != wallet.ID {
		prettyPrint(*addedWallet)
		t.Fatalf("wallet ID mismatch, expected %q, got %q", wallet.ID, addedWallet.Name)
	}
}
