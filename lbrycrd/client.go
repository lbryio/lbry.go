package lbrycrd

import (
	e "errors"
	"net/url"
	"os"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcrpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/go-errors/errors"
	"github.com/go-ini/ini"
)

const DefaultPort = 9245

// MainNetParams define the lbrycrd network. See https://github.com/lbryio/lbrycrd/blob/master/src/chainparams.cpp
var MainNetParams = chaincfg.Params{
	PubKeyHashAddrID: 0x55,
	ScriptHashAddrID: 0x7a,
	PrivateKeyID:     0x1c,
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
	*btcrpcclient.Client
}

// New initializes a new Client
func New(lbrycrdURL string) (*Client, error) {
	// Connect to local bitcoin core RPC server using HTTP POST mode.

	u, err := url.Parse(lbrycrdURL)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if u.User == nil {
		return nil, errors.New("no userinfo")
	}

	password, _ := u.User.Password()

	connCfg := &btcrpcclient.ConnConfig{
		Host:         u.Host,
		User:         u.User.Username(),
		Pass:         password,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are not supported in HTTP POST mode.
	client, err := btcrpcclient.New(connCfg, nil)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// make sure lbrycrd is running and responsive
	_, err = client.GetInfo()
	if err != nil {
		return nil, errors.Wrap(err, 0)
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

var errInsufficientFunds = e.New("insufficient funds")

// SimpleSend is a convenience function to send credits to an address (0 min confirmations)
func (c *Client) SimpleSend(toAddress string, amount float64) (*chainhash.Hash, error) {
	decodedAddress, err := btcutil.DecodeAddress(toAddress, &MainNetParams)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	lbcAmount, err := btcutil.NewAmount(amount)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	hash, err := c.Client.SendFromMinConf("", decodedAddress, lbcAmount, 0)
	if err != nil && err.Error() == "-6: Insufficient funds" {
		err = errors.Wrap(errInsufficientFunds, 0)
	}
	return hash, errors.Wrap(err, 0)
}

//func (c *Client) SendWithSplit(toAddress string, amount float64, numUTXOs int) (*chainhash.Hash, error) {
//	decodedAddress, err := btcutil.DecodeAddress(toAddress, &MainNetParams)
//	if err != nil {
//		return nil, errors.Wrap(err, 0)
//	}
//
//	amountPerAddress, err := btcutil.NewAmount(amount / float64(numUTXOs))
//	if err != nil {
//		return nil, errors.Wrap(err, 0)
//	}
//
//	amounts := map[btcutil.Address]btcutil.Amount{}
//	for i := 0; i < numUTXOs; i++ {
//		addr := decodedAddress // to give it a new address, so
//		amounts[addr] = amountPerAddress
//	}
//
//	hash, err := c.Client.SendManyMinConf("", amounts, 0)
//	if err != nil && err.Error() == "-6: Insufficient funds" {
//		err = errors.Wrap(errInsufficientFunds, 0)
//	}
//	return hash, errors.Wrap(err, 0)
//}

func getLbrycrdURLFromConfFile() (string, error) {
	if os.Getenv("HOME") == "" {
		return "", errors.New("no $HOME var found")
	}

	defaultConfFile := os.Getenv("HOME") + "/.lbrycrd/lbrycrd.conf"
	if _, err := os.Stat(defaultConfFile); os.IsNotExist(err) {
		return "", errors.New("default lbrycrd conf file not found")
	}

	cfg, err := ini.Load(defaultConfFile)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}

	section, err := cfg.GetSection("")
	if err != nil {
		return "", errors.Wrap(err, 0)
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
