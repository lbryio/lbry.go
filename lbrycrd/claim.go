package lbrycrd

import (
	"encoding/hex"

	"github.com/lbryio/lbry.go/extras/errors"
	c "github.com/lbryio/lbryschema.go/claim"
	pb "github.com/lbryio/types/v2/go"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
)

func NewImageStreamClaim() (*c.ClaimHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	image := new(pb.Stream_Image)
	image.Image = new(pb.Image)
	stream.Type = image

	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.ClaimHelper{Claim: pbClaim}

	return &helper, nil
}

func NewVideoStreamClaim() (*c.ClaimHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	video := new(pb.Stream_Video)
	video.Video = new(pb.Video)
	stream.Type = video
	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.ClaimHelper{Claim: pbClaim}

	return &helper, nil
}

func NewStreamClaim(title, description string) (*c.ClaimHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.ClaimHelper{Claim: pbClaim}
	helper.Title = title
	helper.Description = description

	return &helper, nil
}

func SignClaim(rawTx *wire.MsgTx, privKey btcec.PrivateKey, claim, channel *c.ClaimHelper, channelClaimID string) error {
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
