package lbrycrd

import (
	"encoding/hex"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	c "github.com/lbryio/lbry.go/v2/schema/stake"
	pb "github.com/lbryio/types/v2/go"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
)

func NewImageStreamClaim() (*c.StakeHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	image := new(pb.Stream_Image)
	image.Image = new(pb.Image)
	stream.Type = image

	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.StakeHelper{Claim: pbClaim}

	return &helper, nil
}

func NewVideoStreamClaim() (*c.StakeHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	video := new(pb.Stream_Video)
	video.Video = new(pb.Video)
	stream.Type = video
	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.StakeHelper{Claim: pbClaim}

	return &helper, nil
}

func NewStreamClaim(title, description string) (*c.StakeHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.StakeHelper{Claim: pbClaim}
	helper.Claim.Title = title
	helper.Claim.Description = description

	return &helper, nil
}

func SignClaim(rawTx *wire.MsgTx, privKey btcec.PrivateKey, claim, channel *c.StakeHelper, channelClaimID string) error {
	claimIDHexBytes, err := hex.DecodeString(channelClaimID)
	if err != nil {
		return errors.Err(err)
	}
	claim.Version = c.WithSig
	claim.ClaimID = rev(claimIDHexBytes)
	hash, err := c.GetOutpointHash(rawTx.TxIn[0].PreviousOutPoint.Hash.String(), rawTx.TxIn[0].PreviousOutPoint.Index)
	if err != nil {
		return err
	}
	sig, err := c.Sign(privKey, *channel, *claim, hash)
	if err != nil {
		return err
	}

	lbrySig, err := sig.LBRYSDKEncode()
	if err != nil {
		return err
	}
	claim.Signature = lbrySig

	return nil

}
