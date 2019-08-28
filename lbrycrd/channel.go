package lbrycrd

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/lbryio/lbry.go/extras/errors"
	c "github.com/lbryio/lbryschema.go/claim"
	pb "github.com/lbryio/types/v2/go"
)

func NewChannel() (*c.ClaimHelper, *btcec.PrivateKey, error) {
	claimChannel := new(pb.Claim_Channel)
	channel := new(pb.Channel)
	claimChannel.Channel = channel

	pbClaim := new(pb.Claim)
	pbClaim.Type = claimChannel

	privateKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, errors.Err(err)
	}
	pubkeyBytes, err := c.PublicKeyToDER(privateKey.PubKey())
	if err != nil {
		return nil, nil, errors.Err(err)
	}

	helper := c.ClaimHelper{Claim: pbClaim}
	helper.Version = c.NoSig
	helper.GetChannel().PublicKey = pubkeyBytes
	helper.Tags = []string{}
	coverSrc := new(pb.Source)
	helper.GetChannel().Cover = coverSrc
	helper.Languages = []*pb.Language{}
	thumbnailSrc := new(pb.Source)
	helper.Thumbnail = thumbnailSrc
	helper.Locations = []*pb.Location{}

	return &helper, privateKey, nil
}
