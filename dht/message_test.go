package dht

import (
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/lyoshenka/bencode"
	log "github.com/sirupsen/logrus"
)

func TestBencodeDecodeStoreArgs(t *testing.T) {
	log.SetLevel(log.DebugLevel)

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

	if hex.EncodeToString([]byte(storeArgs.BlobHash)) != strings.ToLower(blobHash) {
		t.Error("blob hash mismatch")
	}
	if hex.EncodeToString([]byte(storeArgs.Value.LbryID)) != strings.ToLower(lbryID) {
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
		//spew.Dump(reencoded, data)
	}
}
