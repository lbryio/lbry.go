package stake

import (
	"encoding/hex"

	"github.com/lbryio/lbcutil/base58"
	v1pb "github.com/lbryio/types/v1/go"
	pb "github.com/lbryio/types/v2/go"

	"github.com/cockroachdb/errors"
)

const lbrySDHash = "lbry_sd_hash"

func newStreamClaim() *pb.Claim {
	claimStream := new(pb.Claim_Stream)
	stream := new(pb.Stream)

	pbClaim := new(pb.Claim)
	pbClaim.Type = claimStream
	claimStream.Stream = stream

	return pbClaim
}

func newChannelClaim() *pb.Claim {
	claimChannel := new(pb.Claim_Channel)
	channel := new(pb.Channel)

	pbClaim := new(pb.Claim)
	pbClaim.Type = claimChannel
	claimChannel.Channel = channel

	return pbClaim
}

func setMetaData(claim *pb.Claim, author string, description string, language pb.Language_Language, license string,
	licenseURL *string, title string, thumbnail *string, nsfw bool) {
	claim.Title = title
	claim.Description = description

	claim.GetStream().Author = author
	claim.Languages = []*pb.Language{{Language: language}}

	if thumbnail != nil {
		source := new(pb.Source)
		source.Url = *thumbnail
		claim.Thumbnail = source
	}
	if nsfw {
		claim.Tags = []string{"mature"}
	}
	claim.GetStream().License = license
	if licenseURL != nil {
		claim.GetStream().LicenseUrl = *licenseURL
	}
}

func migrateV1PBClaim(vClaim v1pb.Claim) (*pb.Claim, error) {
	if *vClaim.ClaimType == v1pb.Claim_streamType {
		return migrateV1PBStream(vClaim)
	}
	if *vClaim.ClaimType == v1pb.Claim_certificateType {
		return migrateV1PBChannel(vClaim)
	}
	return nil, errors.WithStack(errors.Newf("Could not migrate v1 protobuf claim due to unknown type '%s'.", vClaim.ClaimType.String()))
}

func migrateV1PBStream(vClaim v1pb.Claim) (*pb.Claim, error) {
	claim := newStreamClaim()
	source := new(pb.Source)
	source.MediaType = vClaim.GetStream().GetSource().GetContentType()
	source.SdHash = vClaim.GetStream().GetSource().GetSource()
	claim.GetStream().Source = source
	md := vClaim.GetStream().GetMetadata()
	if md.GetFee() != nil {
		claim.GetStream().Fee = new(pb.Fee)
		claim.GetStream().GetFee().Amount = uint64(*md.GetFee().Amount * 100000000)
		claim.GetStream().GetFee().Address = md.GetFee().GetAddress()
		claim.GetStream().GetFee().Currency = pb.Fee_Currency(pb.Fee_Currency_value[md.GetFee().GetCurrency().String()])
	}
	if vClaim.GetStream().GetMetadata().GetNsfw() {
		claim.Tags = []string{"mature"}
	}
	thumbnailSource := new(pb.Source)
	thumbnailSource.Url = md.GetThumbnail()
	claim.Thumbnail = thumbnailSource
	language := pb.Language_Language(pb.Language_Language_value[md.GetLanguage().String()])
	claim.Languages = []*pb.Language{{Language: language}}
	claim.GetStream().LicenseUrl = md.GetLicenseUrl()
	claim.GetStream().License = md.GetLicense()
	claim.Title = md.GetTitle()
	claim.Description = md.GetDescription()
	claim.GetStream().Author = md.GetAuthor()

	return claim, nil
}

func migrateV1PBChannel(vClaim v1pb.Claim) (*pb.Claim, error) {
	claim := newChannelClaim()
	claim.GetChannel().PublicKey = vClaim.GetCertificate().PublicKey

	return claim, nil
}

func migrateV1Claim(vClaim V1Claim) (*pb.Claim, error) {
	pbClaim := newStreamClaim()
	//Stream
	// -->Universal
	setFee(vClaim.Fee, pbClaim)
	// -->MetaData
	language := pb.Language_Language(pb.Language_Language_value[vClaim.Language])
	setMetaData(pbClaim, vClaim.Author, vClaim.Description, language,
		vClaim.License, nil, vClaim.Title, vClaim.Thumbnail, false)
	// -->Source
	source := new(pb.Source)
	source.MediaType = vClaim.ContentType

	src, err := hex.DecodeString(vClaim.Sources.LbrySDHash)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	source.SdHash = src
	pbClaim.GetStream().Source = source

	return pbClaim, nil
}

func migrateV2Claim(vClaim V2Claim) (*pb.Claim, error) {
	pbClaim := newStreamClaim()
	//Stream
	// -->Fee
	setFee(vClaim.Fee, pbClaim)
	// -->MetaData
	language := pb.Language_Language(pb.Language_Language_value[vClaim.Language])
	setMetaData(pbClaim, vClaim.Author, vClaim.Description, language,
		vClaim.License, vClaim.LicenseURL, vClaim.Title, vClaim.Thumbnail, vClaim.NSFW)
	// -->Source
	source := new(pb.Source)
	source.MediaType = vClaim.ContentType
	src, err := hex.DecodeString(vClaim.Sources.LbrySDHash)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	source.SdHash = src
	pbClaim.GetStream().Source = source

	return pbClaim, nil
}

func migrateV3Claim(vClaim V3Claim) (*pb.Claim, error) {
	pbClaim := newStreamClaim()
	//Stream
	// -->Fee
	setFee(vClaim.Fee, pbClaim)
	// -->MetaData
	language := pb.Language_Language(pb.Language_Language_value[vClaim.Language])
	setMetaData(pbClaim, vClaim.Author, vClaim.Description, language,
		vClaim.License, vClaim.LicenseURL, vClaim.Title, vClaim.Thumbnail, vClaim.NSFW)
	// -->Source
	source := new(pb.Source)
	source.MediaType = vClaim.ContentType
	src, err := hex.DecodeString(vClaim.Sources.LbrySDHash)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	source.SdHash = src
	pbClaim.GetStream().Source = source

	return pbClaim, nil
}

func setFee(fee *Fee, pbClaim *pb.Claim) {
	if fee != nil {
		amount := float32(0.0)
		currency := pb.Fee_LBC
		address := ""
		if fee.BTC != nil {
			amount = fee.BTC.Amount
			currency = pb.Fee_BTC
			address = fee.BTC.Address
		} else if fee.LBC != nil {
			amount = fee.LBC.Amount
			currency = pb.Fee_LBC
			address = fee.LBC.Address
		} else if fee.USD != nil {
			amount = fee.USD.Amount
			currency = pb.Fee_USD
			address = fee.USD.Address
		}
		pbClaim.GetStream().Fee = new(pb.Fee)
		//Fee Settings
		pbClaim.GetStream().GetFee().Amount = uint64(amount * 100000000)
		pbClaim.GetStream().GetFee().Currency = currency
		pbClaim.GetStream().GetFee().Address = base58.Decode(address)
	}
}
