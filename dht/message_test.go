package dht

import (
	"encoding/hex"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/lyoshenka/bencode"
)

func TestBencodeDecodeStoreArgs(t *testing.T) {
	blobHash := "3214D6C2F77FCB5E8D5FC07EDAFBA614F031CE8B2EAB49F924F8143F6DFBADE048D918710072FB98AB1B52B58F4E1468"
	lbryID := "7CE1B831DEC8689E44F80F547D2DEA171F6A625E1A4FF6C6165E645F953103DABEB068A622203F859C6C64658FD3AA3B"
	port := hex.EncodeToString([]byte("3333"))
	token := "17C2D8E1E48EF21567FE4AD5C8ED944B798D3B65AB58D0C9122AD6587D1B5FED472EA2CB12284CEFA1C21EFF302322BD"
	nodeID := "7CE1B831DEC8689E44F80F547D2DEA171F6A625E1A4FF6C6165E645F953103DABEB068A622203F859C6C64658FD3AA3B"
	selfStore := hex.EncodeToString([]byte("1"))

	raw := "6C" + // start args list
		"3438 3A " + blobHash + // blob hash
		"64" + // start value dict
		"363A6C6272796964 3438 3A " + lbryID + // lbry id
		"343A706F7274 69 " + port + " 65" + // port
		"353A746F6B656E 3438 3A " + token + // token
		"65" + // end value dict
		"3438 3A " + nodeID + // node id
		"69 " + selfStore + " 65" + // self store (integer)
		"65" // end args list

	raw = strings.ToLower(strings.Replace(raw, " ", "", -1))

	data, err := hex.DecodeString(raw)
	if err != nil {
		t.Error(err)
		return
	}

	storeArgs := &storeArgs{}
	err = bencode.DecodeBytes(data, storeArgs)
	if err != nil {
		t.Error(err)
	}

	if storeArgs.BlobHash.Hex() != strings.ToLower(blobHash) {
		t.Error("blob hash mismatch")
	}
	if storeArgs.Value.LbryID.Hex() != strings.ToLower(lbryID) {
		t.Error("lbryid mismatch")
	}
	if hex.EncodeToString([]byte(strconv.Itoa(storeArgs.Value.Port))) != port {
		t.Error("port mismatch")
	}
	if hex.EncodeToString([]byte(storeArgs.Value.Token)) != strings.ToLower(token) {
		t.Error("token mismatch")
	}
	if storeArgs.NodeID.Hex() != strings.ToLower(nodeID) {
		t.Error("node id mismatch")
	}
	if !storeArgs.SelfStore {
		t.Error("selfStore mismatch")
	}

	reencoded, err := bencode.EncodeBytes(storeArgs)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(reencoded, data) {
		t.Error("reencoded data does not match original")
		spew.Dump(reencoded, data)
	}
}

func TestBencodeFindNodesResponse(t *testing.T) {
	res := Response{
		ID:     newMessageID(),
		NodeID: newRandomBitmap(),
		FindNodeData: []Node{
			{id: newRandomBitmap(), ip: net.IPv4(1, 2, 3, 4).To4(), port: 5678},
			{id: newRandomBitmap(), ip: net.IPv4(4, 3, 2, 1).To4(), port: 8765},
		},
	}

	encoded, err := bencode.EncodeBytes(res)
	if err != nil {
		t.Fatal(err)
	}

	var res2 Response
	err = bencode.DecodeBytes(encoded, &res2)
	if err != nil {
		t.Fatal(err)
	}

	compareResponses(t, res, res2)
}

func TestBencodeFindValueResponse(t *testing.T) {
	res := Response{
		ID:           newMessageID(),
		NodeID:       newRandomBitmap(),
		FindValueKey: newRandomBitmap().RawString(),
		FindNodeData: []Node{
			{id: newRandomBitmap(), ip: net.IPv4(1, 2, 3, 4).To4(), port: 5678},
		},
	}

	encoded, err := bencode.EncodeBytes(res)
	if err != nil {
		t.Fatal(err)
	}

	var res2 Response
	err = bencode.DecodeBytes(encoded, &res2)
	if err != nil {
		t.Fatal(err)
	}

	compareResponses(t, res, res2)
}

func compareResponses(t *testing.T, res, res2 Response) {
	if res.ID != res2.ID {
		t.Errorf("expected ID %s, got %s", res.ID, res2.ID)
	}
	if !res.NodeID.Equals(res2.NodeID) {
		t.Errorf("expected NodeID %s, got %s", res.NodeID.Hex(), res2.NodeID.Hex())
	}
	if res.Data != res2.Data {
		t.Errorf("expected Data %s, got %s", res.Data, res2.Data)
	}
	if res.FindValueKey != res2.FindValueKey {
		t.Errorf("expected FindValueKey %s, got %s", res.FindValueKey, res2.FindValueKey)
	}
	if !reflect.DeepEqual(res.FindNodeData, res2.FindNodeData) {
		t.Errorf("expected FindNodeData %s, got %s", spew.Sdump(res.FindNodeData), spew.Sdump(res2.FindNodeData))
	}
}
