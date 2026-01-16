package lbrycrd

import (
	"encoding/hex"
	"net/url"
	"os"
	"strconv"

	c "github.com/lbryio/lbry.go/v3/schema/stake"

	"github.com/lbryio/lbcd/btcec"
	"github.com/lbryio/lbcd/btcjson"
	"github.com/lbryio/lbcd/chaincfg"
	"github.com/lbryio/lbcd/chaincfg/chainhash"
	"github.com/lbryio/lbcd/rpcclient"
	"github.com/lbryio/lbcd/wire"
	"github.com/lbryio/lbcutil"

	"github.com/cockroachdb/errors"
	"gopkg.in/ini.v1"
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
	Bech32HRPSegwit:  "lbc",
	//WitnessPubKeyHashAddrID: , // i cant find these in bitcoin codebase either
	//WitnessScriptHashAddrID:,
	GenesisHash:   &GenesisHash,
	Name:          "mainnet",
	Net:           wire.BitcoinNet(0xfae4aaf1),
	DefaultPort:   "9246",
	BIP0034Height: 1,
	BIP0065Height: 200000,
	BIP0066Height: 200000,
}

const (
	lbrycrdMainPubkeyPrefix    = byte(85)
	lbrycrdMainScriptPrefix    = byte(122)
	lbrycrdTestnetPubkeyPrefix = byte(111)
	lbrycrdTestnetScriptPrefix = byte(196)
	lbrycrdRegtestPubkeyPrefix = byte(111)
	lbrycrdRegtestScriptPrefix = byte(196)

	LbrycrdMain    = "lbrycrd_main"
	LbrycrdTestnet = "lbrycrd_testnet"
	LbrycrdRegtest = "lbrycrd_regtest"
)

var mainNetParams = chaincfg.Params{
	PubKeyHashAddrID: lbrycrdMainPubkeyPrefix,
	ScriptHashAddrID: lbrycrdMainScriptPrefix,
	PrivateKeyID:     0x1c,
}

var testNetParams = chaincfg.Params{
	PubKeyHashAddrID: lbrycrdTestnetPubkeyPrefix,
	ScriptHashAddrID: lbrycrdTestnetScriptPrefix,
	PrivateKeyID:     0x1c,
	Bech32HRPSegwit:  "tlbc",
}

var regTestNetParams = chaincfg.Params{
	PubKeyHashAddrID: lbrycrdRegtestPubkeyPrefix,
	ScriptHashAddrID: lbrycrdRegtestScriptPrefix,
	PrivateKeyID:     0x1c,
	Bech32HRPSegwit:  "rlbc",
}

var ChainParamsMap = map[string]chaincfg.Params{LbrycrdMain: mainNetParams, LbrycrdTestnet: testNetParams, LbrycrdRegtest: regTestNetParams}

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
func New(lbrycrdURL string, chainParams string) (*Client, error) {
	// Connect to local bitcoin core RPC server using HTTP POST mode.

	u, err := url.Parse(lbrycrdURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if u.User == nil {
		return nil, errors.WithStack(errors.New("no userinfo"))
	}

	password, _ := u.User.Password()

	connCfg := &rpcclient.ConnConfig{
		Host:         u.Host,
		User:         u.User.Username(),
		Pass:         password,
		Params:       chainParams,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// make sure lbrycrd is running and responsive
	_, err = client.GetBlockChainInfo()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Client{client}, nil
}

func NewWithDefaultURL() (*Client, error) {
	url, err := getLbrycrdURLFromConfFile()
	if err != nil {
		return nil, err
	}
	return New(url, "")
}

var errInsufficientFunds = errors.New("insufficient funds")

// SimpleSend is a convenience function to send credits to an address (0 min confirmations)
func (c *Client) SimpleSend(toAddress string, amount float64) (*chainhash.Hash, error) {
	decodedAddress, err := DecodeAddress(toAddress, &MainNetParams)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	lbcAmount, err := lbcutil.NewAmount(amount)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	hash, err := c.Client.SendToAddress(decodedAddress, lbcAmount, nil)
	if err != nil {
		if err.Error() == "-6: Insufficient funds" {
			err = errors.WithStack(errInsufficientFunds)
		}
		return nil, errors.WithStack(err)
	}
	return hash, nil
}

func getLbrycrdURLFromConfFile() (string, error) {
	if os.Getenv("HOME") == "" {
		return "", errors.WithStack(errors.New("no $HOME var found"))
	}

	defaultConfFile := os.Getenv("HOME") + "/.lbrycrd/lbrycrd.conf"
	if os.Getenv("REGTEST") == "true" {
		defaultConfFile = os.Getenv("HOME") + "/.lbrycrd_regtest/lbrycrd.conf"
	}
	if _, err := os.Stat(defaultConfFile); os.IsNotExist(err) {
		return "", errors.WithStack(errors.New("default lbrycrd conf file not found"))
	}

	cfg, err := ini.Load(defaultConfFile)
	if err != nil {
		return "", errors.WithStack(err)
	}

	section, err := cfg.GetSection("")
	if err != nil {
		return "", errors.WithStack(err)
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
	addresses := make(map[lbcutil.Address]interface{})
	changeAddress, err := c.GetNewAddress("")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	changeAmount, err := lbcutil.NewAmount(change)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	addresses[changeAddress] = changeAmount
	lockTime := int64(0)
	return c.CreateRawTransaction(inputs, addresses, &lockTime)
}

func (c *Client) GetEmptyTx(totalOutputSpend float64) (*wire.MsgTx, error) {
	totalFees := 0.1
	unspentResults, err := c.ListUnspentMin(1)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	finder := newOutputFinder(unspentResults)

	outputs, err := finder.nextBatch(totalOutputSpend + totalFees)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		return nil, errors.WithStack(errors.New("Not enough spendable outputs to create transaction"))
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
		return nil, errors.WithStack(err)
	}
	if !allInputsSigned {
		return nil, errors.WithStack(errors.New("Not all inputs for the tx could be signed!"))
	}

	return c.SendRawTransaction(signedTx, false)
}

type ScriptType int

const (
	ClaimName ScriptType = iota
	ClaimUpdate
	ClaimSupport
)

func (c *Client) AddStakeToTx(rawTx *wire.MsgTx, claim *c.Helper, name string, claimAmount float64, scriptType ScriptType) error {

	address, err := c.GetNewAddress("")
	if err != nil {
		return errors.WithStack(err)
	}
	amount, err := lbcutil.NewAmount(claimAmount)
	if err != nil {
		return errors.WithStack(err)
	}

	value, err := claim.CompileValue()
	if err != nil {
		return errors.WithStack(err)
	}
	var claimID string
	if len(claim.ClaimID) > 0 {
		claimID = hex.EncodeToString(rev(claim.ClaimID))
	}
	var script []byte
	switch scriptType {
	case ClaimName:
		script, err = GetClaimNamePayoutScript(name, value, address)
		if err != nil {
			return errors.WithStack(err)
		}
	case ClaimUpdate:
		script, err = GetUpdateClaimPayoutScript(name, claimID, value, address)
		if err != nil {
			return errors.WithStack(err)
		}
	case ClaimSupport:
		script, err = GetUpdateClaimPayoutScript(name, claimID, value, address)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	rawTx.AddTxOut(wire.NewTxOut(int64(amount), script))

	return nil
}

func (c *Client) CreateChannel(name string, amount float64) (*c.Helper, *btcec.PrivateKey, error) {
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

func (c *Client) SupportClaim(name, claimID, address, blockchainName string, claimAmount float64) (*chainhash.Hash, error) {
	const DefaultFeePerSupport = float64(0.0001)
	unspentResults, err := c.ListUnspentMin(1)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	finder := newOutputFinder(unspentResults)
	outputs, err := finder.nextBatch(claimAmount + DefaultFeePerSupport)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		return nil, errors.WithStack(errors.New("Not enough spendable outputs to create transaction"))
	}
	inputs := make([]btcjson.TransactionInput, len(outputs))

	var totalInputSpend float64
	for i, output := range outputs {
		inputs[i] = btcjson.TransactionInput{Txid: output.TxID, Vout: output.Vout}
		totalInputSpend = totalInputSpend + output.Amount
	}

	change := totalInputSpend - claimAmount - DefaultFeePerSupport
	rawTx, err := c.CreateBaseRawTx(inputs, change)
	if err != nil {
		return nil, err
	}
	chainParams, ok := ChainParamsMap[blockchainName]
	if !ok {
		return nil, errors.WithStack(errors.Newf("invalid blockchain name %s", blockchainName))
	}
	decodedAddress, err := DecodeAddress(address, &chainParams)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	amount, err := lbcutil.NewAmount(claimAmount)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	script, err := GetClaimSupportPayoutScript(name, claimID, decodedAddress)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	rawTx.AddTxOut(wire.NewTxOut(int64(amount), script))

	return c.SignTxAndSend(rawTx)
}
