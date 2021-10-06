package claim

import (
	"encoding/hex"

	"github.com/cockroachdb/errors"
	"github.com/lbryio/lbcd/txscript"
	"github.com/lbryio/lbcutil"
)

func ClaimSupportPayoutScript(name, claimid string, address lbcutil.Address) ([]byte, error) {
	//OP_SUPPORT_CLAIM <name> <claimid> OP_2DROP OP_DROP OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG

	pkscript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	bytes, err := hex.DecodeString(claimid)
	if err != nil {
		return nil, errors.WithStack(err)
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

func ClaimNamePayoutScript(name string, value []byte, address lbcutil.Address) ([]byte, error) {
	//OP_CLAIM_NAME <name> <value> OP_2DROP OP_DROP OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG

	pkscript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, errors.WithStack(err)
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

func UpdateClaimPayoutScript(name, claimid string, value []byte, address lbcutil.Address) ([]byte, error) {
	//OP_UPDATE_CLAIM <name> <claimid> <value> OP_2DROP OP_DROP OP_DUP OP_HASH160 <address> OP_EQUALVERIFY OP_CHECKSIG

	pkscript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	bytes, err := hex.DecodeString(claimid)
	if err != nil {
		return nil, errors.WithStack(err)
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

// rev reverses a byte slice. useful for switching endian-ness
func rev(b []byte) []byte {
	r := make([]byte, len(b))
	for left, right := 0, len(b)-1; left < right; left, right = left+1, right-1 {
		r[left], r[right] = b[right], b[left]
	}
	return r
}
