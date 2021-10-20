package stake

import (
	"encoding/hex"
	"strconv"

	"github.com/lbryio/lbry.go/v3/schema/address"
	"github.com/lbryio/lbry.go/v3/schema/keys"
	v1PB "github.com/lbryio/types/v1/go"
	v2PB "github.com/lbryio/types/v2/go"

	"github.com/cockroachdb/errors"
	"github.com/golang/protobuf/proto"
	"github.com/lbryio/lbcd/btcec"
)

type version byte

func (v version) byte() byte {
	return byte(v)
}

const (
	NoSig = version(byte(0))
	//Signature using ECDSA SECP256k1 key and SHA-256 hash.
	WithSig = version(byte(1))
	UNKNOWN = version(byte(2))
)

type Helper struct {
	Claim       *v2PB.Claim
	Support     *v2PB.Support
	LegacyClaim *v1PB.Claim
	ClaimID     []byte
	Version     version
	Signature   []byte
	Payload     []byte
}

func (c *Helper) ValidateAddresses(blockchainName string) error {
	if c.Claim != nil { // V2
		// check the validity of a fee address
		if c.Claim.GetStream() != nil {
			fee := c.GetStream().GetFee()
			if fee != nil {
				return validateAddress(fee.GetAddress(), blockchainName)
			}
			return nil
		}

		if c.Claim.GetChannel() != nil {
			return nil
		}
	}

	return errors.WithStack(errors.New("claim helper created with migrated v2 protobuf claim 'invalid state'"))
}

func validateAddress(tmpAddr []byte, blockchainName string) error {
	if len(tmpAddr) != 25 {
		return errors.WithStack(errors.New("invalid address length: " + strconv.FormatInt(int64(len(tmpAddr)), 10)))
	}
	addr := [25]byte{}
	for i := range addr {
		addr[i] = tmpAddr[i]
	}
	_, err := address.EncodeAddress(addr, blockchainName)
	if err != nil {
		return err
	}

	return nil
}

func getVersionFromByte(versionByte byte) version {
	if versionByte == byte(0) {
		return NoSig
	} else if versionByte == byte(1) {
		return WithSig
	}

	return UNKNOWN
}

func (c *Helper) ValidateCertificate() error {
	if !c.IsClaim() || c.Claim.GetChannel() == nil {
		return nil
	}
	_, err := c.GetPublicKey()
	if err != nil {
		return err
	}
	return nil
}

func (c *Helper) IsClaim() bool {
	return c.Claim != nil && c.Claim.String() != ""
}

func (c *Helper) IsSupport() bool {
	return c.Support != nil
}

func (c *Helper) LoadFromBytes(rawClaim []byte, blockchainName string) error {
	return c.loadFromBytes(rawClaim, false, blockchainName)
}

func (c *Helper) LoadSupportFromBytes(rawClaim []byte, blockchainName string) error {
	return c.loadFromBytes(rawClaim, true, blockchainName)
}

func (c *Helper) loadFromBytes(rawClaim []byte, isSupport bool, blockchainName string) error {
	if c.Claim.String() != "" && !isSupport {
		return errors.WithStack(errors.New("already initialized"))
	}
	if len(rawClaim) < 1 {
		return errors.WithStack(errors.New("there is nothing to decode"))
	}

	var claimPb *v2PB.Claim
	var legacyClaimPb *v1PB.Claim
	var supportPb *v2PB.Support

	version := getVersionFromByte(rawClaim[0]) //First byte = version
	pbPayload := rawClaim[1:]
	var claimID []byte
	var signature []byte
	if version == WithSig {
		if len(rawClaim) < 85 {
			return errors.WithStack(errors.New("signature version indicated by 1st byte but not enough bytes for valid format"))
		}
		claimID = rawClaim[1:21]    // channel claimid = next 20 bytes
		signature = rawClaim[21:85] // signature = next 64 bytes
		pbPayload = rawClaim[85:]   // protobuf payload = remaining bytes
	}

	var err error
	if !isSupport {
		claimPb = &v2PB.Claim{}
		err = proto.Unmarshal(pbPayload, claimPb)
	} else {
		support := &v2PB.Support{}
		err = proto.Unmarshal(pbPayload, support)
		if err == nil {
			supportPb = support
		}
	}
	if err != nil {
		legacyClaimPb = &v1PB.Claim{}
		legacyErr := proto.Unmarshal(rawClaim, legacyClaimPb)
		if legacyErr == nil {
			claimPb, err = migrateV1PBClaim(*legacyClaimPb)
			if err != nil {
				return errors.WithMessage(err, "migration from v1 to v2 protobuf failed")
			}
			if legacyClaimPb.GetPublisherSignature() != nil {
				version = WithSig
				claimID = legacyClaimPb.GetPublisherSignature().GetCertificateId()
				signature = legacyClaimPb.GetPublisherSignature().GetSignature()
			}
			if legacyClaimPb.GetCertificate() != nil {
				version = NoSig
			}
		} else {
			return err
		}
	}

	*c = Helper{
		Claim:       claimPb,
		Support:     supportPb,
		LegacyClaim: legacyClaimPb,
		ClaimID:     claimID,
		Version:     version,
		Signature:   signature,
		Payload:     pbPayload,
	}

	// Commenting out because of a bug in SDK release allowing empty addresses.
	//err = c.ValidateAddresses(blockchainName)
	//if err != nil {
	//	return err
	//}

	err = c.ValidateCertificate()
	if err != nil {
		return err
	}

	return nil
}

func (c *Helper) LoadFromHexString(claim_hex string, blockchainName string) error {
	buf, err := hex.DecodeString(claim_hex)
	if err != nil {
		return err
	}
	return c.LoadFromBytes(buf, blockchainName)
}

func (c *Helper) LoadSupportFromHexString(claim_hex string, blockchainName string) error {
	buf, err := hex.DecodeString(claim_hex)
	if err != nil {
		return err
	}
	return c.LoadSupportFromBytes(buf, blockchainName)
}

func DecodeClaimProtoBytes(serialized []byte, blockchainName string) (*Helper, error) {
	claim := &Helper{&v2PB.Claim{}, &v2PB.Support{}, nil, nil, NoSig, nil, nil}
	err := claim.LoadFromBytes(serialized, blockchainName)
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func DecodeSupportProtoBytes(serialized []byte, blockchainName string) (*Helper, error) {
	claim := &Helper{nil, &v2PB.Support{}, nil, nil, NoSig, nil, nil}
	err := claim.LoadSupportFromBytes(serialized, blockchainName)
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func DecodeClaimHex(serialized string, blockchainName string) (*Helper, error) {
	claimBytes, err := hex.DecodeString(serialized)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return DecodeClaimBytes(claimBytes, blockchainName)
}

// DecodeClaimBytes take a byte array and tries to decode it to a protobuf claim or migrate it from either json v1,2,3 or pb v1
func DecodeClaimBytes(serialized []byte, blockchainName string) (*Helper, error) {
	helper, err := DecodeClaimProtoBytes(serialized, blockchainName)
	if err == nil {
		return helper, nil
	}
	helper = &Helper{}
	//If protobuf fails, try json versions before returning an error.
	v1Claim := new(V1Claim)
	err = v1Claim.Unmarshal(serialized)
	if err != nil {
		v2Claim := new(V2Claim)
		err := v2Claim.Unmarshal(serialized)
		if err != nil {
			v3Claim := new(V3Claim)
			err := v3Claim.Unmarshal(serialized)
			if err != nil {
				return nil, errors.WithMessage(err, "vlaim value has no matching version")
			}
			helper.Claim, err = migrateV3Claim(*v3Claim)
			if err != nil {
				return nil, errors.WithMessage(err, "v3 metadata migration")
			}
			return helper, nil
		}
		helper.Claim, err = migrateV2Claim(*v2Claim)
		if err != nil {
			return nil, errors.WithMessage(err, "v2 metadata migration")
		}
		return helper, nil
	}

	helper.Claim, err = migrateV1Claim(*v1Claim)
	if err != nil {
		return nil, errors.WithMessage(err, "v1 metadata migration")
	}
	return helper, nil
}

// DecodeSupportBytes take a byte array and tries to decode it to a protobuf support
func DecodeSupportBytes(serialized []byte, blockchainName string) (*Helper, error) {
	return DecodeSupportProtoBytes(serialized, blockchainName)
}

func (c *Helper) GetStream() *v2PB.Stream {
	if c != nil {
		return c.Claim.GetStream()
	}
	return nil
}

func (c *Helper) CompileValue() ([]byte, error) {
	payload, err := c.serialized()
	if err != nil {
		return nil, err
	}
	var value []byte
	value = append(value, c.Version.byte())
	if c.Version == WithSig {
		value = append(value, c.ClaimID...)
		value = append(value, c.Signature...)
	}
	value = append(value, payload...)

	return value, nil
}

func (c *Helper) GetPublicKey() (*btcec.PublicKey, error) {
	if c.IsClaim() {
		if c.Claim.GetChannel() == nil {
			return nil, errors.WithStack(errors.New("claim is not of type channel, so there is no public key to get"))
		}

	} else if c.IsSupport() {
		return nil, errors.WithStack(errors.New("stake is a support and does not come with a public key to get"))
	}
	return keys.GetPublicKeyFromBytes(c.Claim.GetChannel().PublicKey)
}
