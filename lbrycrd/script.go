package lbrycrd

import (
	"encoding/hex"

	btcutil "github.com/lbryio/lbcutil"
	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/lbryio/lbcd/txscript"
)

func getClaimSupportPayoutScript(name, claimid string, address btcutil.Address) ([]byte, error) {
	//OP_SUPPORT_CLAIM <name> <claimid> OP_2DROP OP_DROP OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG

	pkscript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, errors.Err(err)
	}

	bytes, err := hex.DecodeString(claimid)
	if err != nil {
		return nil, errors.Err(err)
	}

	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_SUPPORTCLAIM). //OP_SUPPORT_CLAIM
		AddData([]byte(name)).           //<name>
		AddData(rev(bytes)).             //<claimid>
		AddOp(txscript.OP_2DROP).        //OP_2DROP
		AddOp(txscript.OP_DROP).         //OP_DROP
		AddOps(pkscript).                //OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG
		Script()

}

func getClaimNamePayoutScript(name string, value []byte, address btcutil.Address) ([]byte, error) {
	//OP_CLAIM_NAME <name> <value> OP_2DROP OP_DROP OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG

	pkscript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, errors.Err(err)
	}

	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_CLAIMNAME). //OP_CLAIMNAME
		AddData([]byte(name)).        //<name>
		AddData(value).               //<value>
		AddOp(txscript.OP_2DROP).     //OP_2DROP
		AddOp(txscript.OP_DROP).      //OP_DROP
		AddOps(pkscript).             //OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG
		Script()
}

func getUpdateClaimPayoutScript(name, claimid string, value []byte, address btcutil.Address) ([]byte, error) {
	//OP_UPDATE_CLAIM <name> <claimid> <value> OP_2DROP OP_DROP OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG

	pkscript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, errors.Err(err)
	}

	bytes, err := hex.DecodeString(claimid)
	if err != nil {
		return nil, errors.Err(err)
	}

	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_UPDATECLAIM). //OP_UPDATE_CLAIM
		AddData([]byte(name)).          //<name>
		AddData(rev(bytes)).            //<claimid>
		AddData(value).                 //<value>
		AddOp(txscript.OP_2DROP).       //OP_2DROP
		AddOp(txscript.OP_DROP).        //OP_DROP
		AddOps(pkscript).               //OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG
		Script()
}
