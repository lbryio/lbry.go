package lbrycrd

import (
	"github.com/lbryio/lbry.go/v3/schema/keys"
	c "github.com/lbryio/lbry.go/v3/schema/stake"

	"github.com/lbryio/lbcd/btcec"
	pb "github.com/lbryio/types/v2/go"

	"github.com/cockroachdb/errors"
)

func NewChannel() (*c.Helper, *btcec.PrivateKey, error) {
	claimChannel := new(pb.Claim_Channel)
	channel := new(pb.Channel)
	claimChannel.Channel = channel

	pbClaim := new(pb.Claim)
	pbClaim.Type = claimChannel

	privateKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	pubkeyBytes, err := keys.PublicKeyToDER(privateKey.PubKey())
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	helper := c.Helper{Claim: pbClaim}
	helper.Version = c.NoSig
	helper.Claim.GetChannel().PublicKey = pubkeyBytes
	helper.Claim.Tags = []string{}
	coverSrc := new(pb.Source)
	helper.Claim.GetChannel().Cover = coverSrc
	helper.Claim.Languages = []*pb.Language{}
	thumbnailSrc := new(pb.Source)
	helper.Claim.Thumbnail = thumbnailSrc
	helper.Claim.Locations = []*pb.Location{}

	return &helper, privateKey, nil
}
