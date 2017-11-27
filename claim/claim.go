package claim

import (
	"../address"
	"../pb"
	"encoding/hex"
	"errors"
	"github.com/golang/protobuf/proto"
)

type ClaimHelper struct {
	*pb.Claim
}

func (c *ClaimHelper) ValidateAddresses(blockchainName string) error {
	// check the validity of a fee address
	if c.GetClaimType() == pb.Claim_streamType {
		fee := c.GetStream().GetMetadata().GetFee()
		if fee.String() != "" {
			tmp_addr := fee.GetAddress()
			if len(tmp_addr) != 25 {
				return errors.New("invalid address length")
			}
			addr := [25]byte{}
			for i := range addr {
				addr[i] = tmp_addr[i]
			}
			_, err := address.EncodeAddress(addr, blockchainName)
			if err != nil {
				return err
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
		return err
	}
	if keyType.String() != SECP256k1 {
		return errors.New("wrong curve: " + keyType.String())
	}
	return nil
}

func (c *ClaimHelper) LoadFromBytes(raw_claim []byte, blockchainName string) error {
	if c.String() != "" {
		return errors.New("already initialized")
	}
	if len(raw_claim) < 1 {
		return errors.New("there is nothing to decode")
	}

	claim_pb := &pb.Claim{}
	err := proto.Unmarshal(raw_claim, claim_pb)
	if err != nil {
		return err
	}
	*c = ClaimHelper{claim_pb}
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

func DecodeClaimBytes(serialized []byte, blockchainName string) (*ClaimHelper, error) {
	claim := &ClaimHelper{&pb.Claim{}}
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
