package lbrycrd

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/lbryio/lbcd/txscript"
	"github.com/lbryio/lbcutil"

	"github.com/cockroachdb/errors"
)

func GetClaimSupportPayoutScript(name, claimid string, address lbcutil.Address) ([]byte, error) {
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

func GetClaimNamePayoutScript(name string, value []byte, address lbcutil.Address) ([]byte, error) {
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

func GetUpdateClaimPayoutScript(name, claimid string, value []byte, address lbcutil.Address) ([]byte, error) {
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

// IsClaimNameScript returns true if the script for the vout contains the OP_CLAIM_NAME code.
func IsClaimNameScript(script []byte) bool {
	return len(script) > 0 && script[0] == txscript.OP_CLAIMNAME
}

// IsClaimUpdateScript returns true if the script for the vout contains the OP_CLAIM_UPDATE code.
func IsClaimUpdateScript(script []byte) bool {
	return len(script) > 0 && script[0] == txscript.OP_UPDATECLAIM
}

// ParseClaimNameScript parses a script for the claim of a name.
func ParseClaimNameScript(script []byte) (name string, value []byte, pubkeyscript []byte, err error) {
	// Already validated by blockchain so can be assumed
	// opClaimName Name Value OP_2DROP OP_DROP pubkeyscript
	nameBytesToRead := int(script[1])
	nameStart := 2
	if nameBytesToRead == txscript.OP_PUSHDATA1 {
		nameBytesToRead = int(script[2])
		nameStart = 3
	} else if nameBytesToRead > txscript.OP_PUSHDATA1 {
		return "", nil, nil, errors.WithStack(errors.New("bytes to read is more than next byte"))
	}
	nameEnd := nameStart + nameBytesToRead
	name = string(script[nameStart:nameEnd])
	dataPushType := int(script[nameEnd])
	valueBytesToRead := int(script[nameEnd])
	valueStart := nameEnd + 1
	if dataPushType == txscript.OP_PUSHDATA1 {
		valueBytesToRead = int(script[nameEnd+1])
		valueStart = nameEnd + 2
	} else if dataPushType == txscript.OP_PUSHDATA2 {
		valueStart = nameEnd + 3
		valueBytesToRead = int(binary.LittleEndian.Uint16(script[nameEnd+1 : valueStart]))
	} else if dataPushType == txscript.OP_PUSHDATA4 {
		valueStart = nameEnd + 5
		valueBytesToRead = int(binary.LittleEndian.Uint32(script[nameEnd+2 : valueStart]))
	}
	valueEnd := valueStart + valueBytesToRead
	value = script[valueStart:valueEnd]
	pksStart := valueEnd + 2         // +2 to ignore OP_2DROP and OP_DROP
	pubkeyscript = script[pksStart:] //Remainder is always pubkeyscript

	return name, value, pubkeyscript, err
}

// ParseClaimUpdateScript parses a script for an update of a claim.
func ParseClaimUpdateScript(script []byte) (name string, claimid string, value []byte, pubkeyscript []byte, err error) {
	// opUpdateClaim Name ClaimID Value OP_2DROP OP_2DROP pubkeyscript

	//Name
	nameBytesToRead := int(script[1])
	nameStart := 2
	if nameBytesToRead == txscript.OP_PUSHDATA1 {
		nameBytesToRead = int(script[2])
		nameStart = 3
	} else if nameBytesToRead > txscript.OP_PUSHDATA1 {
		err = errors.WithStack(errors.New("ParseClaimUpdateScript: Bytes to read is more than next byte! "))
		return
	}
	nameEnd := nameStart + nameBytesToRead
	name = string(script[nameStart:nameEnd])

	//ClaimID
	claimidBytesToRead := int(script[nameEnd])
	claimidStart := nameEnd + 1
	claimidEnd := claimidStart + claimidBytesToRead
	bytes := rev(script[claimidStart:claimidEnd])
	claimid = hex.EncodeToString(bytes)

	//Value
	dataPushType := int(script[claimidEnd])
	valueBytesToRead := int(script[claimidEnd])
	valueStart := claimidEnd + 1
	if dataPushType == txscript.OP_PUSHDATA1 {
		valueBytesToRead = int(script[claimidEnd+1])
		valueStart = claimidEnd + 2
	} else if dataPushType == txscript.OP_PUSHDATA2 {
		valueStart = claimidEnd + 3
		valueBytesToRead = int(binary.LittleEndian.Uint16(script[claimidEnd+1 : valueStart]))
	} else if dataPushType == txscript.OP_PUSHDATA4 {
		valueStart = claimidEnd + 5
		valueBytesToRead = int(binary.LittleEndian.Uint32(script[claimidEnd+2 : valueStart]))
	}
	valueEnd := valueStart + valueBytesToRead
	value = script[valueStart:valueEnd]

	//PublicKeyScript
	pksStart := valueEnd + 2         // +2 to ignore OP_2DROP and OP_DROP
	pubkeyscript = script[pksStart:] //Remainder is always pubkeyscript

	return name, claimid, value, pubkeyscript, err
}
