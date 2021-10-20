package electrum

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/lbryio/lbry.go/v3/lbrycrd"
	"github.com/lbryio/lbry.go/v3/schema/stake"
	types "github.com/lbryio/types/v2/go"

	"github.com/cockroachdb/errors"
	"github.com/golang/protobuf/proto"
	"github.com/lbryio/lbcutil"
	"github.com/spf13/cast"
)

// Raw makes a raw wallet server request
func (n *Node) Raw(method string, params []string, v interface{}) error {
	return n.request(method, params, v)
}

// ServerVersion returns the server's version.
// https://electrumx.readthedocs.io/en/latest/protocol-methods.html#server-version
func (n *Node) ServerVersion() (string, error) {
	resp := &struct {
		Result []string `json:"result"`
	}{}
	err := n.request("server.version", []string{"reflector.go", ProtocolVersion}, resp)

	var v string
	if len(resp.Result) >= 2 {
		v = resp.Result[1]
	}

	return v, err
}

func (n *Node) Resolve(url string) (*types.Output, error) {
	outputs := &types.Outputs{}
	resp := &struct {
		Result string `json:"result"`
	}{}

	err := n.request("blockchain.claimtrie.resolve", []string{url}, resp)
	if err != nil {
		return nil, err
	}

	b, err := base64.StdEncoding.DecodeString(resp.Result)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = proto.Unmarshal(b, outputs)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(outputs.GetTxos()) != 1 {
		return nil, errors.New("expected 1 output, got " + cast.ToString(len(outputs.GetTxos())))
	}

	if e := outputs.GetTxos()[0].GetError(); e != nil {
		return nil, errors.Newf("%s: %s", e.GetCode(), e.GetText())
	}

	return outputs.GetTxos()[0], nil
}

type GetClaimsInTxResp struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  []struct {
		Name            string        `json:"name"`
		ClaimID         string        `json:"claim_id"`
		Txid            string        `json:"txid"`
		Nout            int           `json:"nout"`
		Amount          int           `json:"amount"`
		Depth           int           `json:"depth"`
		Height          int           `json:"height"`
		Value           string        `json:"value"`
		ClaimSequence   int           `json:"claim_sequence"`
		Address         string        `json:"address"`
		Supports        []interface{} `json:"supports"` // TODO: finish me
		EffectiveAmount int           `json:"effective_amount"`
		ValidAtHeight   int           `json:"valid_at_height"`
	} `json:"result"`
}

func (n *Node) GetClaimsInTx(txid string) (*GetClaimsInTxResp, error) {
	var resp GetClaimsInTxResp
	err := n.request("blockchain.claimtrie.getclaimsintx", []string{txid}, &resp)
	return &resp, err
}

func (n *Node) GetTx(txid string) (string, error) {
	resp := &struct {
		Result string `json:"result"`
	}{}

	err := n.request("blockchain.transaction.get", []string{txid}, resp)
	if err != nil {
		return "", err
	}

	return resp.Result, nil
}

func (n *Node) GetClaimInTx(txid string, nout int) (*types.Claim, error) {
	hexTx, err := n.GetTx(txid)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rawTx, err := hex.DecodeString(hexTx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	tx, err := lbcutil.NewTxFromBytes(rawTx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(tx.MsgTx().TxOut) <= nout {
		return nil, errors.WithStack(errors.New("nout not found"))
	}

	script := tx.MsgTx().TxOut[nout].PkScript

	var value []byte
	if lbrycrd.IsClaimNameScript(script) {
		_, value, _, err = lbrycrd.ParseClaimNameScript(script)
	} else if lbrycrd.IsClaimUpdateScript(script) {
		_, _, value, _, err = lbrycrd.ParseClaimUpdateScript(script)
	} else {
		err = errors.New("no claim found in output")
	}
	if err != nil {
		return nil, err
	}

	ch, err := stake.DecodeClaimBytes(value, "")
	if err != nil {
		return nil, err
	}

	return ch.Claim, nil
}
