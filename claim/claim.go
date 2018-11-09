package claim

import (
	"encoding/hex"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbryschema.go/address"
	"github.com/lbryio/types/go"
)

type ClaimHelper struct {
	*pb.Claim
	migratedFrom []byte
}

func (c *ClaimHelper) ValidateAddresses(blockchainName string) error {
	// check the validity of a fee address
	if c.GetClaimType() == pb.Claim_streamType {
		fee := c.GetStream().GetMetadata().GetFee()
		if fee != nil {
			tmp_addr := fee.GetAddress()
			if len(tmp_addr) != 25 {
				return errors.Err("invalid address length: " + string(len(tmp_addr)) + "!")
			}
			addr := [25]byte{}
			for i := range addr {
				addr[i] = tmp_addr[i]
			}
			_, err := address.EncodeAddress(addr, blockchainName)
			if err != nil {
				return errors.Err(err)
			}
		}
	}
	return nil
}

func (c *ClaimHelper) ValidateCertificate() error {
	certificate := c.GetCertificate()
	if certificate == nil {
		return nil
	}
	keyType := certificate.GetKeyType()
	_, err := c.GetCertificatePublicKey()
	if err != nil {
		return errors.Err(err)
	}
	if keyType.String() != SECP256k1 {
		return errors.Err("wrong curve: " + keyType.String())
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

	claim_pb := &pb.Claim{}
	err := proto.Unmarshal(raw_claim, claim_pb)
	if err != nil {
		return err
	}
	*c = ClaimHelper{claim_pb, raw_claim}
	err = c.ValidateAddresses(blockchainName)
	if err != nil {
		return err
	}
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
	claim := &ClaimHelper{&pb.Claim{}, serialized}
	err := claim.LoadFromBytes(serialized, blockchainName)
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func DecodeClaimHex(serialized string, blockchainName string) (*ClaimHelper, error) {
	claim_bytes, err := hex.DecodeString(serialized)
	if err != nil {
		return nil, err
	}
	return DecodeClaimBytes(claim_bytes, blockchainName)
}

func DecodeClaimJSON(claimJSON string, blockchainName string) (*ClaimHelper, error) {
	c := &pb.Claim{}
	err := jsonpb.UnmarshalString(claimJSON, c)
	if err != nil {
		return nil, err
	}
	return &ClaimHelper{c, []byte(claimJSON)}, nil
}

// DecodeClaimBytes take a byte array and tries to decode it to a protobuf claim or migrate it from either json v1,2,3
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
				return nil, errors.Prefix("Claim value has no matching verion", err)
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
		return c.Stream
	}
	return nil
}

func (c *ClaimHelper) GetCertificate() *pb.Certificate {
	if c != nil {
		return c.Certificate
	}
	return nil
}

func (c *ClaimHelper) GetPublisherSignature() *pb.Signature {
	if c != nil {
		return c.PublisherSignature
	}
	return nil
}
