package jsonrpc

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/stream"

	schema "github.com/lbryio/lbryschema.go/claim"
	lbryschema "github.com/lbryio/types/v2/go"

	"github.com/shopspring/decimal"
)

type Currency string

const (
	CurrencyLBC = Currency("LBC")
	CurrencyUSD = Currency("USD")
	CurrencyBTC = Currency("BTC")
)

type Fee struct {
	FeeCurrency Currency        `json:"fee_currency"`
	FeeAmount   decimal.Decimal `json:"fee_amount"`
	FeeAddress  *string         `json:"fee_address"`
}

type File struct {
	BlobsCompleted       uint64            `json:"blobs_completed"`
	BlobsInStream        uint64            `json:"blobs_in_stream"`
	BlobsRemaining       uint64            `json:"blobs_remaining"`
	ChannelClaimID       string            `json:"channel_claim_id"`
	ChannelName          string            `json:"channel_name"`
	ClaimID              string            `json:"claim_id"`
	ClaimName            string            `json:"claim_name"`
	Completed            bool              `json:"completed"`
	Confirmations        int64             `json:"confirmations"`
	ContentFee           *Fee              `json:"content_fee"`
	DownloadDirectory    string            `json:"download_directory"`
	DownloadPath         string            `json:"download_path"`
	FileName             string            `json:"file_name"`
	Height               int               `json:"height"`
	Key                  string            `json:"key"`
	Metadata             *lbryschema.Claim `json:"protobuf"`
	MimeType             string            `json:"mime_type"`
	Nout                 int               `json:"nout"`
	Outpoint             string            `json:"outpoint"`
	PointsPaid           decimal.Decimal   `json:"points_paid"`
	SdHash               string            `json:"sd_hash"`
	Status               string            `json:"status"`
	Stopped              bool              `json:"stopped"`
	StreamHash           string            `json:"stream_hash"`
	StreamName           string            `json:"stream_name"`
	StreamingURL         string            `json:"streaming_url"`
	SuggestedFileName    string            `json:"suggested_file_name"`
	Timestamp            int64             `json:"timestamp"`
	TotalBytes           uint64            `json:"total_bytes"`
	TotalBytesLowerBound uint64            `json:"total_bytes_lower_bound"`
	Txid                 string            `json:"txid"`
	WrittenBytes         uint64            `json:"written_bytes"`
}

func getEnumVal(enum map[string]int32, data interface{}) (int32, error) {
	s, ok := data.(string)
	if !ok {
		return 0, errors.Err("expected a string")
	}
	val, ok := enum[s]
	if !ok {
		return 0, errors.Err("invalid enum key")
	}
	return val, nil
}

func fixDecodeProto(src, dest reflect.Type, data interface{}) (interface{}, error) {
	switch dest {
	case reflect.TypeOf(uint64(0)):
		if n, ok := data.(json.Number); ok {
			val, err := n.Int64()
			if err != nil {
				return nil, errors.Wrap(err, 0)
			} else if val < 0 {
				return nil, errors.Err("must be unsigned int")
			}
			return uint64(val), nil
		}
	case reflect.TypeOf([]byte{}):
		if s, ok := data.(string); ok {
			return []byte(s), nil
		}

	case reflect.TypeOf(decimal.Decimal{}):
		if n, ok := data.(json.Number); ok {
			val, err := n.Float64()
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
			return decimal.NewFromFloat(val), nil
		} else if s, ok := data.(string); ok {
			d, err := decimal.NewFromString(s)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
			return d, nil
		}

	case reflect.TypeOf(lbryschema.Fee_Currency(0)):
		val, err := getEnumVal(lbryschema.Fee_Currency_value, data)
		return lbryschema.Fee_Currency(val), err
	case reflect.TypeOf(lbryschema.Claim{}):
		blockChainName := os.Getenv("BLOCKCHAIN_NAME")
		if blockChainName == "" {
			blockChainName = "lbrycrd_main"
		}

		claim, err := schema.DecodeClaimHex(data.(string), blockChainName)
		if err != nil {
			return nil, err
		}
		return claim.Claim, nil
	}

	return data, nil
}

type WalletBalanceResponse decimal.Decimal

type PeerListResponsePeer struct {
	IP     string `json:"host"`
	Port   uint   `json:"port"`
	NodeId string `json:"node_id"`
}
type PeerListResponse []PeerListResponsePeer

type BlobGetResponse struct {
	Blobs []struct {
		BlobHash string `json:"blob_hash,omitempty"`
		BlobNum  int    `json:"blob_num"`
		IV       string `json:"iv"`
		Length   int    `json:"length"`
	} `json:"blobs"`
	Key               string `json:"key"`
	StreamHash        string `json:"stream_hash"`
	StreamName        string `json:"stream_name"`
	StreamType        string `json:"stream_type"`
	SuggestedFileName string `json:"suggested_file_name"`
}

type StreamCostEstimateResponse decimal.Decimal

type BlobAvailability struct {
	IsAvailable      bool     `json:"is_available"`
	ReachablePeers   []string `json:"reachable_peers"`
	UnReachablePeers []string `json:"unreachable_peers"`
}

type StreamAvailabilityResponse struct {
	IsAvailable          bool             `json:"is_available"`
	DidDecode            bool             `json:"did_decode"`
	DidResolve           bool             `json:"did_resolve"`
	IsStream             bool             `json:"is_stream"`
	NumBlobsInStream     uint64           `json:"num_blobs_in_stream"`
	SDHash               string           `json:"sd_hash"`
	SDBlobAvailability   BlobAvailability `json:"sd_blob_availability"`
	HeadBlobHash         string           `json:"head_blob_hash"`
	HeadBlobAvailability BlobAvailability `json:"head_blob_availability"`
	UseUPNP              bool             `json:"use_upnp"`
	UPNPRedirectIsSet    bool             `json:"upnp_redirect_is_set"`
	Error                string           `json:"error,omitempty"`
}

type GetResponse File
type FileListResponse []File

type WalletListResponse []string

type BlobAnnounceResponse bool

type WalletPrefillAddressesResponse struct {
	Broadcast bool   `json:"broadcast"`
	Complete  bool   `json:"complete"`
	Hex       string `json:"hex"`
}

type WalletNewAddressResponse string

type WalletUnusedAddressResponse string

type Account struct {
	AddressGenerator struct {
		Change struct {
			Gap                   uint64 `json:"gap"`
			MaximumUsesPerAddress uint64 `json:"maximum_uses_per_address"`
		} `json:"change"`
		Name      string `json:"name"`
		Receiving struct {
			Gap                   uint64 `json:"gap"`
			MaximumUsesPerAddress uint64 `json:"maximum_uses_per_address"`
		} `json:"receiving"`
	} `json:"address_generator"`
	Certificates uint64  `json:"certificates"`
	Coins        float64 `json:"coins"`
	Encrypted    bool    `json:"encrypted"`
	ID           string  `json:"id"`
	IsDefault    bool    `json:"is_default"`
	Name         string  `json:"name"`
	PublicKey    string  `json:"public_key"`
	Satoshis     uint64  `json:"satoshis"`
}

type AccountListResponse struct {
	LBCMainnet []Account `json:"lbc_mainnet"`
	LBCTestnet []Account `json:"lbc_testnet"`
	LBCRegtest []Account `json:"lbc_regtest"`
}

type AccountBalanceResponse struct {
	Available         decimal.Decimal  `json:"available"`
	Reserved          decimal.Decimal  `json:"reserved"`
	ReservedSubtotals *decimal.Decimal `json:"reserved_subtotals"`
	Total             decimal.Decimal  `json:"total"`
}

type AccountCreateResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	PublicKey  string  `json:"public_key"`
	PrivateKey string  `json:"private_key"`
	Seed       string  `json:"seed"`
	Ledger     string  `json:"ledger"`
	ModifiedOn float64 `json:"modified_on"`
}

type AccountRemoveResponse AccountCreateResponse

type Transaction struct {
	Address       string            `json:"address"`
	Amount        string            `json:"amount"`
	ClaimID       string            `json:"claim_id"`
	Confirmations int               `json:"confirmations"`
	Height        int               `json:"height"`
	IsChange      bool              `json:"is_change"`
	IsMine        bool              `json:"is_mine"`
	Name          string            `json:"name"`
	Nout          uint64            `json:"nout"`
	PermanentUrl  string            `json:"permanent_url"`
	Protobuf      string            `json:"protobuf,omitempty"`
	Txid          string            `json:"txid"`
	Type          string            `json:"type"`
	Value         *lbryschema.Claim `json:"protobuf"`
}

type TransactionSummary struct {
	Height      int           `json:"height"`
	Hex         string        `json:"hex"`
	Inputs      []Transaction `json:"inputs"`
	Outputs     []Transaction `json:"outputs"`
	TotalFee    string        `json:"total_fee"`
	TotalOutput string        `json:"total_output"`
	Txid        string        `json:"txid"`
}

type AccountFundResponse TransactionSummary

type Address string
type AddressUnusedResponse Address
type AddressListResponse []Address
type ChannelExportResponse string

type ChannelListResponse struct {
	Items      []Transaction `json:"items"`
	Page       uint64        `json:"page"`
	PageSize   uint64        `json:"page_size"`
	TotalPages uint64        `json:"total_pages"`
}

type ClaimAbandonResponse struct {
	Success bool               `json:"success"`
	Tx      TransactionSummary `json:"tx"`
}
type Support struct {
	Amount string `json:"amount"`
	Nout   uint64 `json:"nout"`
	Txid   string `json:"txid"`
}

type Claim struct {
	Address                 string           `json:"address"`
	Amount                  string           `json:"amount"`
	CanonicalURL            string           `json:"canonical_url"`
	ClaimID                 string           `json:"claim_id"`
	ClaimOp                 string           `json:"claim_op,omitempty"`
	Confirmations           int              `json:"confirmations"`
	Height                  int              `json:"height"`
	IsChange                bool             `json:"is_change,omitempty"`
	IsChannelSignatureValid bool             `json:"is_channel_signature_valid,omitempty"`
	Meta                    Meta             `json:"meta,omitempty"`
	Name                    string           `json:"name"`
	NormalizedName          string           `json:"normalized_name"`
	Nout                    uint64           `json:"nout"`
	PermanentURL            string           `json:"permanent_url"`
	ShortURL                string           `json:"short_url"`
	SigningChannel          *Claim           `json:"signing_channel,omitempty"`
	Timestamp               int              `json:"timestamp"`
	Txid                    string           `json:"txid"`
	Type                    string           `json:"type,omitempty"`
	Value                   lbryschema.Claim `json:"protobuf,omitempty"`
	ValueType               string           `json:"value_type,omitempty"`
	AbsoluteChannelPosition int              `json:"absolute_channel_position,omitempty"`
	ChannelName             string           `json:"channel_name,omitempty"`
	ClaimSequence           int64            `json:"claim_sequence,omitempty"`
	DecodedClaim            bool             `json:"decoded_claim,omitempty"`
	EffectiveAmount         string           `json:"effective_amount,omitempty"`
	HasSignature            *bool            `json:"has_signature,omitempty"`
	SignatureIsValid        *bool            `json:"signature_is_valid,omitempty"`
	Supports                []Support        `json:"supports,omitempty"`
	ValidAtHeight           int              `json:"valid_at_height,omitempty"`
}

type Meta struct {
	ActivationHeight  int64   `json:"activation_height,omitempty"`
	CreationHeight    int64   `json:"creation_height,omitempty"`
	CreationTimestamp int     `json:"creation_timestamp,omitempty"`
	EffectiveAmount   string  `json:"effective_amount,omitempty"`
	ExpirationHeight  int64   `json:"expiration_height,omitempty"`
	IsControlling     bool    `json:"is_controlling,omitempty"`
	SupportAmount     string  `json:"support_amount,omitempty"`
	TrendingGlobal    float64 `json:"trending_global,omitempty"`
	TrendingGroup     float64 `json:"trending_group,omitempty"`
	TrendingLocal     float64 `json:"trending_local,omitempty"`
	TrendingMixed     float64 `json:"trending_mixed,omitempty"`
}

const reflectorURL = "http://blobs.lbry.io/"

// GetStreamSizeByMagic uses "magic" to not just estimate, but actually return the exact size of a stream
// It does so by fetching the sd blob and the last blob from our S3 bucket, decrypting and unpadding the last blob
// adding up all full blobs that have a known size and finally adding the real last blob size too.
// This will only work if we host at least the sd blob and the last blob on S3, if not, this will error.
func (c *Claim) GetStreamSizeByMagic() (streamSize uint64, e error) {
	if c.Value.GetStream() == nil {
		return 0, errors.Err("this claim is not a stream")
	}
	resp, err := http.Get(reflectorURL + hex.EncodeToString(c.Value.GetStream().Source.SdHash))
	if err != nil {
		return 0, errors.Err(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.Err(err)
	}
	sdb := &stream.SDBlob{}
	err = sdb.UnmarshalJSON(body)

	if err != nil {
		return 0, err
	}
	lastBlobIndex := len(sdb.BlobInfos) - 2
	lastBlobHash := sdb.BlobInfos[lastBlobIndex].BlobHash

	if len(sdb.BlobInfos) > 2 {
		streamSize = uint64(stream.MaxBlobSize-1) * uint64(len(sdb.BlobInfos)-2)
	}

	resp2, err := http.Get(reflectorURL + hex.EncodeToString(lastBlobHash))
	if err != nil {
		return 0, errors.Err(err)
	}
	defer resp2.Body.Close()

	body2, err := ioutil.ReadAll(resp2.Body)
	if err != nil {
		return 0, errors.Err(err)
	}
	defer func() {
		if r := recover(); r != nil {
			e = errors.Err("recovered from DecryptBlob panic for blob %s", lastBlobHash)
		}
	}()
	lastBlob, err := stream.DecryptBlob(body2, sdb.Key, sdb.BlobInfos[lastBlobIndex].IV)
	if err != nil {
		return 0, errors.Err(err)
	}

	streamSize += uint64(len(lastBlob))
	return streamSize, nil
}

type ClaimListResponse struct {
	Claims     []Claim `json:"items"`
	Page       uint64  `json:"page"`
	PageSize   uint64  `json:"page_size"`
	TotalPages uint64  `json:"total_pages"`
}
type ClaimSearchResponse ClaimListResponse

type SupportListResponse struct {
	Items      []Claim
	Page       uint64 `json:"page"`
	PageSize   uint64 `json:"page_size"`
	TotalPages uint64 `json:"total_pages"`
}
type StatusResponse struct {
	BlobManager struct {
		FinishedBlobs uint64 `json:"finished_blobs"`
	} `json:"blob_manager"`
	ConnectionStatus struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"connection_status"`
	Dht struct {
		NodeID              string `json:"node_id"`
		PeersInRoutingTable uint64 `json:"peers_in_routing_table"`
	} `json:"dht"`
	HashAnnouncer struct {
		AnnounceQueueSize uint64 `json:"announce_queue_size"`
	} `json:"hash_announcer"`
	InstallationID    string   `json:"installation_id"`
	IsFirstRun        bool     `json:"is_first_run"`
	IsRunning         bool     `json:"is_running"`
	SkippedComponents []string `json:"skipped_components"`
	StartupStatus     struct {
		BlobManager         bool `json:"blob_manager"`
		BlockchainHeaders   bool `json:"blockchain_headers"`
		Database            bool `json:"database"`
		Dht                 bool `json:"dht"`
		ExchangeRateManager bool `json:"exchange_rate_manager"`
		HashAnnouncer       bool `json:"hash_announcer"`
		PeerProtocolServer  bool `json:"peer_protocol_server"`
		StreamManager       bool `json:"stream_manager"`
		Upnp                bool `json:"upnp"`
		Wallet              bool `json:"wallet"`
	} `json:"startup_status"`
	StreamManager struct {
		ManagedFiles int64 `json:"managed_files"`
	} `json:"stream_manager"`
	Upnp struct {
		AioupnpVersion  string   `json:"aioupnp_version"`
		DhtRedirectSet  bool     `json:"dht_redirect_set"`
		ExternalIp      string   `json:"external_ip"`
		Gateway         string   `json:"gateway"`
		PeerRedirectSet bool     `json:"peer_redirect_set"`
		Redirects       struct{} `json:"redirects"`
	}
	Wallet struct {
		BestBlochash string `json:"best_blockhash"`
		Blocks       int    `json:"blocks"`
		BlocksBehind int    `json:"blocks_behind"`
		IsEncrypted  bool   `json:"is_encrypted"`
		IsLocked     bool   `json:"is_locked"`
	} `json:"wallet"`
}

type UTXOListResponse []struct {
	Address       string `json:"address"`
	Amount        string `json:"amount"`
	Confirmations int    `json:"confirmations"`
	Height        int    `json:"height"`
	IsChange      bool   `json:"is_change"`
	IsMine        bool   `json:"is_mine"`
	Nout          int    `json:"nout"`
	Txid          string `json:"txid"`
	Type          string `json:"type"`
}

type VersionResponse struct {
	Build   string `json:"build"`
	Desktop string `json:"desktop"`
	Distro  struct {
		Codename     string `json:"codename"`
		ID           string `json:"id"`
		Like         string `json:"like"`
		Version      string `json:"version"`
		VersionParts struct {
			BuildNumber string `json:"build_number"`
			Major       string `json:"major"`
			Minor       string `json:"minor"`
		} `json:"version_parts"`
	} `json:"distro"`
	LbrynetVersion    string `json:"lbrynet_version"`
	LbryschemaVersion string `json:"lbryschema_version"`
	OsRelease         string `json:"os_release"`
	OsSystem          string `json:"os_system"`
	Platform          string `json:"platform"`
	Processor         string `json:"processor"`
	PythonVersion     string `json:"python_version"`
}

type ResolveResponse map[string]Claim

type NumClaimsInChannelResponse map[string]struct {
	ClaimsInChannel *uint64 `json:"claims_in_channel,omitempty"`
	Error           *string `json:"error,omitempty"`
}

type ClaimShowResponse *Claim
