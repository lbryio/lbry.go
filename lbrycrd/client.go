package lbrycrd

import (
	"encoding/hex"
	"net/url"
	"os"
	"strconv"

	"github.com/lbryio/lbry.go/extras/errors"
	c "github.com/lbryio/lbryschema.go/claim"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/go-ini/ini"
)

const DefaultPort = 9245

var GenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x9c, 0x89, 0x28, 0x3b, 0xa0, 0xf3, 0x22, 0x7f,
	0x6c, 0x03, 0xb7, 0x02, 0x16, 0xb9, 0xf6, 0x65,
	0xf0, 0x11, 0x8d, 0x5e, 0x0f, 0xa7, 0x29, 0xce,
	0xdf, 0x4f, 0xb3, 0x4d, 0x6a, 0x34, 0xf4, 0x63,
})

// MainNetParams define the lbrycrd network. See https://github.com/lbryio/lbrycrd/blob/master/src/chainparams.cpp
var MainNetParams = chaincfg.Params{
	PubKeyHashAddrID: 0x55,
	ScriptHashAddrID: 0x7a,
	PrivateKeyID:     0x1c,
	Bech32HRPSegwit:  "not-used", // we don't have this (yet)
	GenesisHash:      &GenesisHash,
}

func init() {
	// Register lbrycrd network
	err := chaincfg.Register(&MainNetParams)
	if err != nil {
		panic("failed to register lbrycrd network: " + err.Error())
	}
}

// Client connects to a lbrycrd instance
type Client struct {
	*rpcclient.Client
}

// New initializes a new Client
func New(lbrycrdURL string) (*Client, error) {
	// Connect to local bitcoin core RPC server using HTTP POST mode.

	u, err := url.Parse(lbrycrdURL)
	if err != nil {
		return nil, errors.Err(err)
	}

	if u.User == nil {
		return nil, errors.Err("no userinfo")
	}

	password, _ := u.User.Password()

	connCfg := &rpcclient.ConnConfig{
		Host:         u.Host,
		User:         u.User.Username(),
		Pass:         password,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, errors.Err(err)
	}

	// make sure lbrycrd is running and responsive
	_, err = client.GetBlockChainInfo()
	if err != nil {
		return nil, errors.Err(err)
	}

	return &Client{client}, nil
}

func NewWithDefaultURL() (*Client, error) {
	url, err := getLbrycrdURLFromConfFile()
	if err != nil {
		return nil, err
	}
	return New(url)
}

var errInsufficientFunds = errors.Base("insufficient funds")

// SimpleSend is a convenience function to send credits to an address (0 min confirmations)
func (c *Client) SimpleSend(toAddress string, amount float64) (*chainhash.Hash, error) {
	decodedAddress, err := DecodeAddress(toAddress, &MainNetParams)
	if err != nil {
		return nil, errors.Err(err)
	}

	lbcAmount, err := btcutil.NewAmount(amount)
	if err != nil {
		return nil, errors.Err(err)
	}

	hash, err := c.Client.SendFromMinConf("", decodedAddress, lbcAmount, 0)
	if err != nil {
		if err.Error() == "-6: Insufficient funds" {
			err = errors.Err(errInsufficientFunds)
		}
		return nil, errors.Err(err)
	}
	return hash, nil
}

func getLbrycrdURLFromConfFile() (string, error) {
	if os.Getenv("HOME") == "" {
		return "", errors.Err("no $HOME var found")
	}

	defaultConfFile := os.Getenv("HOME") + "/.lbrycrd/lbrycrd.conf"
	if os.Getenv("REGTEST") == "true" {
		defaultConfFile = os.Getenv("HOME") + "/.lbrycrd_regtest/lbrycrd.conf"
	}
	if _, err := os.Stat(defaultConfFile); os.IsNotExist(err) {
		return "", errors.Err("default lbrycrd conf file not found")
	}

	cfg, err := ini.Load(defaultConfFile)
	if err != nil {
		return "", errors.Err(err)
	}

	section, err := cfg.GetSection("")
	if err != nil {
		return "", errors.Err(err)
	}

	username := section.Key("rpcuser").String()
	password := section.Key("rpcpassword").String()
	host := section.Key("rpchost").String()
	if host == "" {
		host = "localhost"
	}
	port := section.Key("rpcport").String()
	if port == "" {
		port = strconv.Itoa(DefaultPort)
	}

	userpass := ""
	if username != "" || password != "" {
		userpass = username + ":" + password + "@"
	}

	return "rpc://" + userpass + host + ":" + port, nil
}

func (c *Client) CreateBaseRawTx(inputs []btcjson.TransactionInput, change float64) (*wire.MsgTx, error) {
	addresses := make(map[btcutil.Address]btcutil.Amount)
	changeAddress, err := c.GetNewAddress("")
	if err != nil {
		return nil, errors.Err(err)
	}
	changeAmount, err := btcutil.NewAmount(change)
	if err != nil {
		return nil, errors.Err(err)
	}
	addresses[changeAddress] = changeAmount
	lockTime := int64(0)
	return c.CreateRawTransaction(inputs, addresses, &lockTime)
}

func (c *Client) GetEmptyTx(totalOutputSpend float64) (*wire.MsgTx, error) {
	totalFees := 0.1
	unspentResults, err := c.ListUnspentMin(1)
	if err != nil {
		return nil, errors.Err(err)
	}
	finder := newOutputFinder(unspentResults)

	outputs, err := finder.nextBatch(totalOutputSpend + totalFees)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		return nil, errors.Err("Not enough spendable outputs to create transaction")
	}
	inputs := make([]btcjson.TransactionInput, len(outputs))
	var totalInputSpend float64
	for i, output := range outputs {
		inputs[i] = btcjson.TransactionInput{Txid: output.TxID, Vout: output.Vout}
		totalInputSpend = totalInputSpend + output.Amount
	}

	change := totalInputSpend - totalOutputSpend - totalFees
	return c.CreateBaseRawTx(inputs, change)
}

func (c *Client) SignTxAndSend(rawTx *wire.MsgTx) (*chainhash.Hash, error) {
	signedTx, allInputsSigned, err := c.SignRawTransaction(rawTx)
	if err != nil {
		return nil, errors.Err(err)
	}
	if !allInputsSigned {
		return nil, errors.Err("Not all inputs for the tx could be signed!")
	}

	return c.SendRawTransaction(signedTx, false)
}

type ScriptType int

const (
	ClaimName ScriptType = iota
	ClaimUpdate
	ClaimSupport
)

func (c *Client) AddStakeToTx(rawTx *wire.MsgTx, claim *c.ClaimHelper, name string, claimAmount float64, scriptType ScriptType) error {

	address, err := c.GetNewAddress("")
	if err != nil {
		return errors.Err(err)
	}
	amount, err := btcutil.NewAmount(claimAmount)
	if err != nil {
		return errors.Err(err)
	}

	value, err := claim.CompileValue()
	if err != nil {
		return errors.Err(err)
	}
	var claimID string
	if len(claim.ClaimID) > 0 {
		claimID = hex.EncodeToString(rev(claim.ClaimID))
	}
	var script []byte
	switch scriptType {
	case ClaimName:
		script, err = getClaimNamePayoutScript(name, value, address)
		if err != nil {
			return errors.Err(err)
		}
	case ClaimUpdate:
		script, err = getUpdateClaimPayoutScript(name, claimID, value, address)
		if err != nil {
			return errors.Err(err)
		}
	case ClaimSupport:
		script, err = getUpdateClaimPayoutScript(name, claimID, value, address)
		if err != nil {
			return errors.Err(err)
		}
	}

	rawTx.AddTxOut(wire.NewTxOut(int64(amount), script))

	return nil
}

func (c *Client) CreateChannel(name string, amount float64) (*c.ClaimHelper, *btcec.PrivateKey, error) {
	channel, key, err := NewChannel()
	if err != nil {
		return nil, nil, err
	}

	rawTx, err := c.GetEmptyTx(amount)
	if err != nil {
		return nil, nil, err
	}
	err = c.AddStakeToTx(rawTx, channel, name, amount, ClaimName)
	if err != nil {
		return nil, nil, err
	}

	_, err = c.SignTxAndSend(rawTx)
	if err != nil {
		return nil, nil, err
	}

	return channel, key, nil
}
