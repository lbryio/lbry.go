package jsonrpc

import (
	"fmt"
	"strconv"
	"testing"

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
