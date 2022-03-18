package url

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const regexPartProtocol = "^((?:lbry://|https://)?)"
const regexPartHost = "((?:open.lbry.com/|lbry.tv/|lbry.lat/|lbry.fr/|lbry.in/)?)"
const regexPartStreamOrChannelName = "([^:$#/]*)"
const regexPartModifierSeparator = "([:$#]?)([^/]*)"
const regexQueryStringBreaker = "^([\\S]+)([?][\\S]*)"
const urlComponentsSize = 9

const ChannelNameMinLength = 1
const ClaimIdMaxLength = 40
const ProtoDefault = "lbry://"
const RegexClaimId = "(?i)^[0-9a-f]+$"
const RegexInvalidUri = "(?i)[ =&#:$@%?;/\\\\\\\\\\\"<>%\\\\{\\\\}|^~\\\\[\\\\]`\\u0000-\\u0008\\u000b-\\u000c\\u000e-\\u001F\\uD800-\\uDFFF\\uFFFE-\\uFFFF]"

type LbryUri struct {
	Path                   string
	IsChannel              bool
	StreamName             string
	StreamClaimId          string
	ChannelName            string
	ChannelClaimId         string
	PrimaryClaimSequence   int
	SecondaryClaimSequence int
	PrimaryBidPosition     int
	SecondaryBidPosition   int
	ClaimName              string
	ClaimId                string
	ContentName            string
	QueryString            string
}

type UriModifier struct {
	ClaimId       string
	ClaimSequence int
	BidPosition   int
}

func (uri LbryUri) IsChannelUrl() bool {
	return (!isEmpty(uri.ChannelName) && isEmpty(uri.StreamName)) || (!isEmpty(uri.ClaimName) && strings.HasPrefix(uri.ClaimName, "@"))
}

func (uri LbryUri) IsNameValid(name string) bool {
	return !regexp.MustCompile(RegexInvalidUri).MatchString(name)
}

func (uri LbryUri) String() string {
	return uri.Build(true, ProtoDefault, false)
}

func (uri LbryUri) VanityString() string {
	return uri.Build(true, ProtoDefault, true)
}

func (uri LbryUri) TvString() string {
	return uri.Build(true, "https://lbry.tv/", false)
}

func (uri LbryUri) Build(includeProto bool, protocol string, vanity bool) string {
	formattedChannelName := ""
	if !isEmpty(uri.ChannelName) {
		formattedChannelName = uri.ChannelName
		if !strings.HasPrefix(formattedChannelName, "@") {
			formattedChannelName = fmt.Sprintf("@%s", formattedChannelName)
		}
	}
	primaryClaimName := uri.ClaimName
	if isEmpty(primaryClaimName) {
		primaryClaimName = uri.ContentName
	}
	if isEmpty(primaryClaimName) {
		primaryClaimName = formattedChannelName
	}
	if isEmpty(primaryClaimName) {
		primaryClaimName = uri.StreamName
	}

	primaryClaimId := uri.ClaimId
	if isEmpty(primaryClaimId) {
		if !isEmpty(formattedChannelName) {
			primaryClaimId = uri.ChannelClaimId
		} else {
			primaryClaimId = uri.StreamClaimId
		}
	}

	var sb strings.Builder
	if includeProto {
		sb.WriteString(protocol)
	}
	sb.WriteString(primaryClaimName)
	if vanity {
		return sb.String()
	}

	secondaryClaimName := ""
	if isEmpty(uri.ClaimName) && !isEmpty(uri.ContentName) {
		secondaryClaimName = uri.ContentName
	}
	if isEmpty(secondaryClaimName) {
		if !isEmpty(formattedChannelName) {
			secondaryClaimName = uri.StreamName
		}
	}
	secondaryClaimId := ""
	if !isEmpty(secondaryClaimName) {
		secondaryClaimId = uri.StreamClaimId
	}

	if !isEmpty(primaryClaimId) {
		sb.WriteString("#")
		sb.WriteString(primaryClaimId)
	} else if uri.PrimaryClaimSequence > 0 {
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(uri.PrimaryClaimSequence))
	} else if uri.PrimaryBidPosition > 0 {
		sb.WriteString("$")
		sb.WriteString(strconv.Itoa(uri.PrimaryBidPosition))
	}

	if !isEmpty(secondaryClaimName) {
		sb.WriteString("/")
		sb.WriteString(secondaryClaimName)
	}

	if !isEmpty(secondaryClaimId) {
		sb.WriteString("#")
		sb.WriteString(secondaryClaimId)
	} else if uri.SecondaryClaimSequence > 0 {
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(uri.SecondaryClaimSequence))
	} else if uri.SecondaryBidPosition > 0 {
		sb.WriteString("$")
		sb.WriteString(strconv.Itoa(uri.SecondaryBidPosition))
	}

	return sb.String()
}

func Parse(url string, requireProto bool) (*LbryUri, error) {
	if isEmpty(url) {
		return nil, errors.New("invalid url parameter")
	}

	reComponents := regexp.MustCompile(
		fmt.Sprintf("(?i)%s%s%s%s(/?)%s%s",
			regexPartProtocol,
			regexPartHost,
			regexPartStreamOrChannelName,
			regexPartModifierSeparator,
			regexPartStreamOrChannelName,
			regexPartModifierSeparator))
	reSeparateQueryString := regexp.MustCompile(regexQueryStringBreaker)

	cleanUrl := url
	queryString := ""

	qsMatches := reSeparateQueryString.FindStringSubmatch(url)
	if len(qsMatches) == 3 {
		cleanUrl = qsMatches[1]
		queryString = qsMatches[2][1:]
	}

	var components []string
	componentMatches := reComponents.FindStringSubmatch(cleanUrl)
	for _, component := range componentMatches[1:] {
		components = append(components, component)
	}
	if len(components) != urlComponentsSize {
		return nil, errors.New("regular expression error occurred while trying to Parse the value")
	}

	/*
	 * components[0] = proto
	 * components[1] = host
	 * components[2] = streamName or channelName
	 * components[3] = primaryModSeparator
	 * components[4] = primaryModValue
	 * components[5] = path separator
	 * components[6] = possibleStreamName
	 * components[7] = secondaryModSeparator
	 * components[8] = secondaryModValue
	 */
	if requireProto && isEmpty(components[0]) {
		return nil, errors.New("url must include a protocol prefix (lbry://)")
	}
	if isEmpty(components[2]) {
		return nil, errors.New("url does not include a name")
	}
	for _, component := range components[2:] {
		if strings.Index(component, " ") > -1 {
			return nil, errors.New("url cannot include a space")
		}
	}

	streamOrChannelName := components[2]
	primaryModSeparator := components[3]
	primaryModValue := components[4]
	possibleStreamName := components[6]
	secondaryModSeparator := components[7]
	secondaryModValue := components[8]
	primaryClaimId := ""
	primaryClaimSequence := -1
	primaryBidPosition := -1
	secondaryClaimSequence := -1
	secondaryBidPosition := -1

	includesChannel := strings.HasPrefix(streamOrChannelName, "@")
	isChannel := includesChannel && isEmpty(possibleStreamName)
	channelName := ""
	if includesChannel && len(streamOrChannelName) > 1 {
		channelName = streamOrChannelName[1:]
	}

	// Convert the mod separators when parsing with protocol https://lbry.tv/ or similar
	// [https://] uses ':', [lbry://] expects #
	if !isEmpty(components[1]) {
		if primaryModSeparator == ":" {
			primaryModSeparator = "#"
		}
		if secondaryModSeparator == ":" {
			secondaryModSeparator = "#"
		}
	}

	if includesChannel {
		if isEmpty(channelName) {
			// I wonder if this check is really necessary, considering the subsequent min length check
			return nil, errors.New("no channel name after @")
		}
		if len(channelName) < ChannelNameMinLength {
			return nil, errors.New(fmt.Sprintf("Channel names must be at least %d character long.", ChannelNameMinLength))
		}
	}

	var err error
	var primaryMod *UriModifier
	var secondaryMod *UriModifier
	if !isEmpty(primaryModSeparator) && !isEmpty(primaryModValue) {
		primaryMod, err = parseModifier(primaryModSeparator, primaryModValue)
		if err != nil {
			return nil, err
		}

		primaryClaimId = primaryMod.ClaimId
		primaryClaimSequence = primaryMod.ClaimSequence
		primaryBidPosition = primaryMod.BidPosition
	}
	if !isEmpty(secondaryModSeparator) && !isEmpty(secondaryModValue) {
		secondaryMod, err = parseModifier(secondaryModSeparator, secondaryModValue)
		if err != nil {
			return nil, err
		}

		secondaryClaimSequence = secondaryMod.ClaimSequence
		secondaryBidPosition = secondaryMod.BidPosition
	}

	streamName := streamOrChannelName
	if includesChannel {
		streamName = possibleStreamName
	}

	streamClaimId := ""
	if includesChannel && secondaryMod != nil {
		streamClaimId = secondaryMod.ClaimId
	} else if primaryMod != nil {
		streamClaimId = primaryMod.ClaimId
	}
	channelClaimId := ""
	if includesChannel && primaryMod != nil {
		channelClaimId = primaryMod.ClaimId
	}

	return &LbryUri{
		Path:                   strings.Join(components[2:], ""),
		IsChannel:              isChannel,
		StreamName:             streamName,
		StreamClaimId:          streamClaimId,
		ChannelName:            channelName,
		ChannelClaimId:         channelClaimId,
		PrimaryClaimSequence:   primaryClaimSequence,
		SecondaryClaimSequence: secondaryClaimSequence,
		PrimaryBidPosition:     primaryBidPosition,
		SecondaryBidPosition:   secondaryBidPosition,
		ClaimName:              streamOrChannelName,
		ClaimId:                primaryClaimId,
		ContentName:            streamName,
		QueryString:            queryString,
	}, nil
}

func parseModifier(modSeparator string, modValue string) (*UriModifier, error) {
	claimId := ""
	claimSequence := 0
	bidPosition := 0

	if !isEmpty(modSeparator) {
		if isEmpty(modValue) {
			return nil, errors.New(fmt.Sprintf("No modifier provided after separator %s", modSeparator))
		}

		if modSeparator == "#" {
			claimId = modValue
		} else if modSeparator == ":" {
			claimId = modValue
		} else if modSeparator == "$" {
			bidPosition = parseInt(modValue, -1)
		}
	}

	if !isEmpty(claimId) && (len(claimId) > ClaimIdMaxLength || !regexp.MustCompile(RegexClaimId).MatchString(claimId)) {
		return nil, errors.New(fmt.Sprintf("Invalid claim ID %s", claimId))
	}
	if claimSequence == -1 {
		return nil, errors.New("claim sequence must be a number")
	}
	if bidPosition == -1 {
		return nil, errors.New("bid position must be a number")
	}

	return &UriModifier{
		ClaimId:       claimId,
		ClaimSequence: claimSequence,
		BidPosition:   bidPosition,
	}, nil
}

func parseInt(value string, defaultValue int) int {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int(v)
}

func isEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}
