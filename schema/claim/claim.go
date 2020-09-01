package claim

import (
	"encoding/hex"
	"strconv"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/schema/address"
	legacy_pb "github.com/lbryio/types/v1/go"
	pb "github.com/lbryio/types/v2/go"

	"github.com/golang/protobuf/proto"
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

type ClaimHelper struct {
	*pb.Claim
	LegacyClaim *legacy_pb.Claim
	ClaimID     []byte
	Version     version
	Signature   []byte
	Payload     []byte
}

const migrationErrorMessage = "migration from v1 to v2 protobuf failed with: "

func (c *ClaimHelper) ValidateAddresses(blockchainName string) error {
	if c.Claim != nil { // V2
		// check the validity of a fee address
		if c.Claim.GetStream() != nil {
			fee := c.GetStream().GetFee()
			if fee != nil {
				return validateAddress(fee.GetAddress(), blockchainName)
			} else {
				return nil
			}
		} else if c.GetChannel() != nil {
			return nil
		}
	}

	return errors.Err("claim helper created with migrated v2 protobuf claim 'invalid state'")
}

func validateAddress(tmp_addr []byte, blockchainName string) error {
	if len(tmp_addr) != 25 {
		return errors.Err("invalid address length: " + strconv.FormatInt(int64(len(tmp_addr)), 10) + "!")
	}
	addr := [25]byte{}
	for i := range addr {
		addr[i] = tmp_addr[i]
	}
	_, err := address.EncodeAddress(addr, blockchainName)
	if err != nil {
		return errors.Err(err)
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

func (c *ClaimHelper) ValidateCertificate() error {
	if c.GetChannel() == nil {
		return nil
	}
	_, err := c.GetPublicKey()
	if err != nil {
		return errors.Err(err)
	}
	return nil
}

func (c *ClaimHelper) LoadFromBytes(raw_claim []byte, blockchainName string) error {
	if c.String() != "" {
		return errors.Err("already initialized")
	}
	if len(raw_claim) < 1 {
		return errors.Err("there is nothing to decode")
	}

	var claim_pb *pb.Claim
	var legacy_claim_pb *legacy_pb.Claim

	version := getVersionFromByte(raw_claim[0]) //First byte = version
	pbPayload := raw_claim[1:]
	var claimID []byte
	var signature []byte
	if version == WithSig {
		if len(raw_claim) < 86 {
			return errors.Err("signature version indicated by 1st byte but not enough bytes for valid format")
		}
		claimID = raw_claim[1:21]    // channel claimid = next 20 bytes
		signature = raw_claim[21:85] // signature = next 64 bytes
		pbPayload = raw_claim[85:]   // protobuf payload = remaining bytes
	}

	claim_pb = &pb.Claim{}
	err := proto.Unmarshal(pbPayload, claim_pb)
	if err != nil {
		legacy_claim_pb = &legacy_pb.Claim{}
		legacyErr := proto.Unmarshal(raw_claim, legacy_claim_pb)
		if legacyErr == nil {
			claim_pb, err = migrateV1PBClaim(*legacy_claim_pb)
			if err != nil {
				return errors.Prefix(migrationErrorMessage, err)
			}
			if legacy_claim_pb.GetPublisherSignature() != nil {
				version = WithSig
				claimID = legacy_claim_pb.GetPublisherSignature().GetCertificateId()
				signature = legacy_claim_pb.GetPublisherSignature().GetSignature()
			}
			if legacy_claim_pb.GetCertificate() != nil {
				version = NoSig
			}
		} else {
			return err
		}
	}

	*c = ClaimHelper{
		Claim:       claim_pb,
		LegacyClaim: legacy_claim_pb,
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

func (c *ClaimHelper) LoadFromHexString(claim_hex string, blockchainName string) error {
	buf, err := hex.DecodeString(claim_hex)
	if err != nil {
		return err
	}
	return c.LoadFromBytes(buf, blockchainName)
}

func DecodeClaimProtoBytes(serialized []byte, blockchainName string) (*ClaimHelper, error) {
	claim := &ClaimHelper{&pb.Claim{}, nil, nil, NoSig, nil, nil}
	err := claim.LoadFromBytes(serialized, blockchainName)
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func DecodeClaimHex(serialized string, blockchainName string) (*ClaimHelper, error) {
	claim_bytes, err := hex.DecodeString(serialized)
	if err != nil {
		return nil, errors.Err(err)
	}
	return DecodeClaimBytes(claim_bytes, blockchainName)
}

// DecodeClaimBytes take a byte array and tries to decode it to a protobuf claim or migrate it from either json v1,2,3 or pb v1
func DecodeClaimBytes(serialized []byte, blockchainName string) (*ClaimHelper, error) {
	helper, err := DecodeClaimProtoBytes(serialized, blockchainName)
	if err == nil {
		return helper, nil
	}
	helper = &ClaimHelper{}
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
				return nil, errors.Prefix("Claim value has no matching version", err)
			}
			helper.Claim, err = migrateV3Claim(*v3Claim)
			if err != nil {
				return nil, errors.Prefix("V3 Metadata Migration Error", err)
			}
			return helper, nil
		}
		helper.Claim, err = migrateV2Claim(*v2Claim)
		if err != nil {
			return nil, errors.Prefix("V2 Metadata Migration Error ", err)
		}
		return helper, nil
	}

	helper.Claim, err = migrateV1Claim(*v1Claim)
	if err != nil {
		return nil, errors.Prefix("V1 Metadata Migration Error ", err)
	}
	return helper, nil
}

func (c *ClaimHelper) GetStream() *pb.Stream {
	if c != nil {
		return c.Claim.GetStream()
	}
	return nil
}

func (c *ClaimHelper) CompileValue() ([]byte, error) {
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
