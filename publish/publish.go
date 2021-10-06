package publish

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/lbryio/lbry.go/v3/lbrycrd"
	"github.com/lbryio/lbry.go/v3/stream"
	pb "github.com/lbryio/types/v2/go"

	"github.com/lbryio/lbcd/btcjson"
	"github.com/lbryio/lbcd/chaincfg/chainhash"
	"github.com/lbryio/lbcd/wire"
	"github.com/lbryio/lbcutil"

	"github.com/cockroachdb/errors"
	"github.com/golang/protobuf/proto"
)

/* TODO:
import cert from wallet
get all utxos from chainquery
create transaction
sign it with the channel
track state of utxos across publishes from this channel so that we can just do one query to get utxos
prioritize only confirmed utxos

Handling all the issues we handle currently with lbrynet:
	"Couldn't find private key for id",
	"You already have a stream claim published under the name",
	"Cannot publish using channel",
	"txn-mempool-conflict",
	"too-long-mempool-chain",
	"Missing inputs",
	"Not enough funds to cover this transaction",
*/

type Details struct {
	Title       string
	Description string
	Author      string
	Tags        []string
	ReleaseTime int64
}

func Publish(client *lbrycrd.Client, path, name, address string, details Details) (stream.Stream, *wire.MsgTx, *chainhash.Hash, error) {
	if name == "" {
		return nil, nil, nil, errors.WithStack(errors.New("name required"))
	}

	//TODO: sign claim if publishing into channel

	addr, err := lbcutil.DecodeAddress(address, &lbrycrd.MainNetParams)
	if errors.Is(err, lbcutil.ErrUnknownAddressType) {
		return nil, nil, nil, errors.WithStack(errors.New(`unknown address type. here's what you need to make this work:
- deprecatedrpc=validateaddress" and "deprecatedrpc=signrawtransaction" in your lbrycrd.conf
- github.com/lbryio/lbcd pinned to hash 306aecffea32
- github.com/btcsuite/lbcutil pinned to 4c204d697803
- github.com/lbryio/lbry.go/v2 (make sure you have v2 at the end)`))
	}
	if err != nil {
		return nil, nil, nil, err
	}

	amount := 0.01
	changeAddr := addr // TODO: fix this? or maybe its fine?
	tx, err := baseTx(client, amount, changeAddr)
	if err != nil {
		return nil, nil, nil, err
	}

	st, stPB, err := makeStream(path)
	if err != nil {
		return nil, nil, nil, err
	}

	stPB.Author = details.Author
	stPB.ReleaseTime = details.ReleaseTime

	claim := &pb.Claim{
		Title:       details.Title,
		Description: details.Description,
		Type:        &pb.Claim_Stream{Stream: stPB},
	}

	err = addClaimToTx(tx, claim, name, amount, addr)
	if err != nil {
		return nil, nil, nil, err
	}

	// sign and send
	signedTx, allInputsSigned, err := client.SignRawTransaction(tx)
	if err != nil {
		return nil, nil, nil, err
	}
	if !allInputsSigned {
		return nil, nil, nil, errors.WithStack(errors.New("not all inputs for the tx could be signed"))
	}

	txid, err := client.SendRawTransaction(signedTx, false)
	if err != nil {
		return nil, nil, nil, err
	}

	return st, signedTx, txid, nil
}

//TODO: lots of assumptions. hardcoded values need to be passed in or calculated
func baseTx(client *lbrycrd.Client, amount float64, changeAddress lbcutil.Address) (*wire.MsgTx, error) {
	txFee := 0.0002 // TODO: estimate this better?

	inputs, total, err := coinChooser(client, amount+txFee)
	if err != nil {
		return nil, err
	}

	change := total - amount - txFee

	// create base raw tx
	addresses := make(map[lbcutil.Address]lbcutil.Amount)
	//changeAddr, err := client.GetNewAddress("")
	changeAmount, err := lbcutil.NewAmount(change)
	if err != nil {
		return nil, err
	}
	addresses[changeAddress] = changeAmount
	lockTime := int64(0)
	return client.CreateRawTransaction(inputs, addresses, &lockTime)
}

func coinChooser(client *lbrycrd.Client, amount float64) ([]btcjson.TransactionInput, float64, error) {
	utxos, err := client.ListUnspentMin(1)
	if err != nil {
		return nil, 0, err
	}

	sort.Slice(utxos, func(i, j int) bool { return utxos[i].Amount < utxos[j].Amount })

	var utxo btcjson.ListUnspentResult
	for _, u := range utxos {
		if u.Spendable && u.Amount >= amount {
			utxo = u
			break
		}
	}
	if utxo.TxID == "" {
		return nil, 0, errors.WithStack(errors.New("not enough utxos to create tx"))
	}

	return []btcjson.TransactionInput{{Txid: utxo.TxID, Vout: utxo.Vout}}, utxo.Amount, nil
}

func addClaimToTx(tx *wire.MsgTx, claim *pb.Claim, name string, amount float64, claimAddress lbcutil.Address) error {
	claimBytes, err := proto.Marshal(claim)
	if err != nil {
		return err
	}
	claimBytes = append([]byte{0}, claimBytes...) // version 0 = no channel sig

	amt, err := lbcutil.NewAmount(amount)
	if err != nil {
		return err
	}

	script, err := lbrycrd.GetClaimNamePayoutScript(name, claimBytes, claimAddress)
	if err != nil {
		return err
	}

	tx.AddTxOut(wire.NewTxOut(int64(amt), script))
	return nil
}

func decodeTx(client *lbrycrd.Client, tx *wire.MsgTx) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return "", errors.WithStack(err)
	}
	//txHex := hex.EncodeToString(buf.Bytes())
	//spew.Dump(txHex)
	decoded, err := client.DecodeRawTransaction(buf.Bytes())
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(decoded, "", "  ")
	return string(data), err
}

func makeStream(path string) (stream.Stream, *pb.Stream, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	defer file.Close()

	enc := stream.NewEncoder(file)

	s, err := enc.Stream()
	if err != nil {
		return nil, nil, err
	}

	streamProto := &pb.Stream{
		Source: &pb.Source{
			SdHash: enc.SDBlob().Hash(),
			Name:   filepath.Base(file.Name()),
			Size:   uint64(enc.SourceLen()),
			Hash:   enc.SourceHash(),
		},
	}

	mimeType, category := guessMimeType(filepath.Ext(file.Name()))
	streamProto.Source.MediaType = mimeType

	switch category {
	case "video":
		//t, err := streamVideoMetadata(path)
		//if err != nil {
		//	return nil, nil, err
		//}
		streamProto.Type = &pb.Stream_Video{}
	case "audio":
		streamProto.Type = &pb.Stream_Audio{}
	case "image":
		streamProto.Type = &pb.Stream_Image{}
	}

	return s, streamProto, nil
}

//func streamVideoMetadata(path string) (*pb.Stream_Video, error) {
//	mi, err := mediainfo.GetMediaInfo(path)
//	if err != nil {
//		return nil, err
//	}
//	return &pb.Stream_Video{
//		Video: &pb.Video{
//			Duration: uint32(mi.General.Duration / 1000),
//			Height:   uint32(mi.Video.Height),
//			Width:    uint32(mi.Video.Width),
//		},
//	}, nil
//}
