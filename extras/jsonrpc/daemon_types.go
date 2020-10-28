package jsonrpc

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/stream"

	schema "github.com/lbryio/lbry.go/v2/schema/stake"
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
	AddedOn              int64             `json:"added_on"`
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
	IsFullyReflected     bool              `json:"is_fully_reflected"`
	Key                  string            `json:"key"`
	Metadata             *lbryschema.Claim `json:"protobuf"`
	MimeType             string            `json:"mime_type"`
	Nout                 int               `json:"nout"`
	Outpoint             string            `json:"outpoint"`
	Protobuf             string            `json:"protobuf"`
	PurchaseReceipt      interface{}       `json:"purchase_receipt"`
	ReflectorProgress    int               `json:"reflector_progress"`
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
	UploadingToReflector bool              `json:"uploading_to_reflector"`
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
type FileListResponse struct {
	Items      []File `json:"items"`
	Page       uint64 `json:"page"`
	PageSize   uint64 `json:"page_size"`
	TotalPages uint64 `json:"total_pages"`
}

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
	Certificates uint64   `json:"certificates"`
	Coins        float64  `json:"coins"`
	Encrypted    bool     `json:"encrypted"`
	ID           string   `json:"id"`
	IsDefault    bool     `json:"is_default"`
	Ledger       *string  `json:"ledger,omitempty"`
	ModifiedOn   *float64 `json:"modified_on,omitempty"`
	Name         string   `json:"name"`
	Preferences  *struct {
		Theme string `json:"theme"`
	} `json:"preferences,omitempty"`
	PrivateKey *string `json:"private_key,omitempty"`
	PublicKey  string  `json:"public_key"`
	Seed       *string `json:"seed,omitempty"`
	Satoshis   uint64  `json:"satoshis"`
}

type AccountListResponse struct {
	Items      []Account `json:"items"`
	Page       uint64    `json:"page"`
	PageSize   uint64    `json:"page_size"`
	TotalPages uint64    `json:"total_pages"`
}

type AccountBalanceResponse struct {
	Available         decimal.Decimal `json:"available"`
	Reserved          decimal.Decimal `json:"reserved"`
	ReservedSubtotals struct {
		Claims   decimal.Decimal `json:"claims"`
		Supports decimal.Decimal `json:"supports"`
		Tips     decimal.Decimal `json:"tips"`
	} `json:"reserved_subtotals"`
	Total decimal.Decimal `json:"total"`
}

type Transaction struct {
	Address            string            `json:"address"`
	Amount             string            `json:"amount"`
	ClaimID            string            `json:"claim_id"`
	ClaimOp            string            `json:"claim_op"`
	Confirmations      int               `json:"confirmations"`
	HasSigningKey      bool              `json:"has_signing_key"`
	Height             int               `json:"height"`
	IsInternalTransfer bool              `json:"is_internal_transfer"`
	IsMyInput          bool              `json:"is_my_input"`
	IsMyOutput         bool              `json:"is_my_output"`
	IsSpent            bool              `json:"is_spent"`
	Name               string            `json:"name"`
	NormalizedName     string            `json:"normalized_name"`
	Nout               uint64            `json:"nout"`
	PermanentUrl       string            `json:"permanent_url"`
	TimeStamp          uint64            `json:"time_stamp"`
	Protobuf           string            `json:"protobuf,omitempty"`
	Txid               string            `json:"txid"`
	Type               string            `json:"type"`
	Value              *lbryschema.Claim `json:"protobuf"`
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
type AddressListResponse struct {
	Items []struct {
		Account   string  `json:"account"`
		Address   Address `json:"address"`
		Pubkey    string  `json:"pubkey"`
		UsedTimes uint64  `json:"used_times"`
	} `json:"items"`
	Page       uint64 `json:"page"`
	PageSize   uint64 `json:"page_size"`
	TotalPages uint64 `json:"total_pages"`
}

type ChannelExportResponse string
type ChannelImportResponse string

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

type PurchaseReceipt struct {
	Address       string `json:"file_name"`
	Amount        string `json:"amount"`
	ClaimID       string `json:"claim_id"`
	Confirmations int    `json:"confirmations"`
	Height        int    `json:"height"`
	Nout          uint64 `json:"nout"`
	Timestamp     uint64 `json:"timestamp"`
	Txid          string `json:"txid"`
	Type          string `json:"purchase"`
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
	IsInternalTransfer      bool             `json:"is_internal_transfer"`
	IsMyInput               bool             `json:"is_my_input"`
	IsMyOutput              bool             `json:"is_my_output"`
	IsSpent                 bool             `json:"is_spent"`
	Meta                    Meta             `json:"meta,omitempty"`
	Name                    string           `json:"name"`
	NormalizedName          string           `json:"normalized_name"`
	Nout                    uint64           `json:"nout"`
	PermanentURL            string           `json:"permanent_url"`
	PurchaseReceipt         *PurchaseReceipt `json:"purchase_receipt,omitempty"`
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

type StreamListResponse struct {
	Items      []Claim `json:"items"`
	Page       uint64  `json:"page"`
	PageSize   uint64  `json:"page_size"`
	TotalPages uint64  `json:"total_pages"`
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
		Connections struct {
			MaxIncomingMbs   float64 `json:"max_incoming_mbs"`
			MaxOutgoingMbs   float64 `json:"max_outgoing_mbs"`
			TotalIncomingMbs float64 `json:"total_incoming_mbs"`
			TotalOutgoingMbs float64 `json:"total_outgoing_mbs"`
			TotalReceived    int64   `json:"total_received"`
			TotalSent        int64   `json:"total_sent"`
		} `json:"connections"`
		FinishedBlobs int64 `json:"finished_blobs"`
	} `json:"blob_manager"`
	ConnectionStatus struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"connection_status"`
	Dht struct {
		NodeID              string `json:"node_id"`
		PeersInRoutingTable uint64 `json:"peers_in_routing_table"`
	} `json:"dht"`
	FfmpegStatus struct {
		AnalyzeAudioVolume bool   `json:"analyze_audio_volume"`
		Available          bool   `json:"available"`
		Which              string `json:"which"`
	} `json:"ffmpeg_status"`
	FileManager struct {
		ManagedFiles int64 `json:"managed_files"`
	} `json:"file_manager"`
	HashAnnouncer struct {
		AnnounceQueueSize uint64 `json:"announce_queue_size"`
	} `json:"hash_announcer"`
	InstallationID    string   `json:"installation_id"`
	IsRunning         bool     `json:"is_running"`
	SkippedComponents []string `json:"skipped_components"`
	StartupStatus     struct {
		BlobManager          bool `json:"blob_manager"`
		Database             bool `json:"database"`
		Dht                  bool `json:"dht"`
		ExchangeRateManager  bool `json:"exchange_rate_manager"`
		FileManager          bool `json:"file_manager"`
		HashAnnouncer        bool `json:"hash_announcer"`
		LibtorrentComponent  bool `json:"libtorrent_component"`
		PeerProtocolServer   bool `json:"peer_protocol_server"`
		Upnp                 bool `json:"upnp"`
		Wallet               bool `json:"wallet"`
		WalletServerPayments bool `json:"wallet_server_payments"`
	} `json:"startup_status"`
	Upnp struct {
		AioupnpVersion  string   `json:"aioupnp_version"`
		DhtRedirectSet  bool     `json:"dht_redirect_set"`
		ExternalIp      string   `json:"external_ip"`
		Gateway         string   `json:"gateway"`
		PeerRedirectSet bool     `json:"peer_redirect_set"`
		Redirects       struct{} `json:"redirects"`
	} `json:"upnp"`
	Wallet struct {
		AvailableServers  int    `json:"available_servers"`
		BestBlockhash     string `json:"best_blockhash"`
		Blocks            int    `json:"blocks"`
		BlocksBehind      int    `json:"blocks_behind"`
		Connected         string `json:"connected"`
		ConnectedFeatures struct {
			DailyFee        string `json:"daily_fee"`
			Description     string `json:"description"`
			DonationAddress string `json:"donation_address"`
			GenesisHash     string `json:"genesis_hash"`
			HashFunction    string `json:"hash_function"`
			Hosts           struct {
			} `json:"hosts"`
			PaymentAddress    string      `json:"payment_address"`
			ProtocolMax       string      `json:"protocol_max"`
			ProtocolMin       string      `json:"protocol_min"`
			Pruning           interface{} `json:"pruning"`
			ServerVersion     string      `json:"server_version"`
			TrendingAlgorithm string      `json:"trending_algorithm"`
		} `json:"connected_features"`
		HeadersSynchronizationProgress int `json:"headers_synchronization_progress"`
		KnownServers                   int `json:"known_servers"`
		Servers                        []struct {
			Availability bool    `json:"availability"`
			Host         string  `json:"host"`
			Latency      float64 `json:"latency"`
			Port         int     `json:"port"`
		} `json:"servers"`
	} `json:"wallet"`
	WalletServerPayments struct {
		MaxFee  string `json:"max_fee"`
		Running bool   `json:"running"`
	} `json:"wallet_server_payments"`
}

type UTXOListResponse struct {
	Items []struct {
		Address            string `json:"address"`
		Amount             string `json:"amount"`
		Confirmations      int    `json:"confirmations"`
		Height             int    `json:"height"`
		IsInternalTransfer bool   `json:"is_internal_transfer"`
		IsMyInput          bool   `json:"is_my_input"`
		IsMyOutput         bool   `json:"is_my_output"`
		IsSpent            bool   `json:"is_spent"`
		Nout               int    `json:"nout"`
		Timestamp          int64  `json:"timestamp"`
		Txid               string `json:"txid"`
		Type               string `json:"type"`
	} `json:"items"`
	Page       uint64 `json:"page"`
	PageSize   uint64 `json:"page_size"`
	TotalPages uint64 `json:"total_pages"`
}

type UTXOReleaseResponse *string

type transactionListBlob struct {
	Address      string `json:"address"`
	Amount       string `json:"amount"`
	BalanceDelta string `json:"balance_delta"`
	ClaimId      string `json:"claim_id"`
	ClaimName    string `json:"claim_name"`
	IsSpent      bool   `json:"is_spent"`
	Nout         int    `json:"nout"`
}

//TODO: this repeats all the fields from transactionListBlob which doesn't make sense
// but if i extend the type with transactionListBlob it doesn't fill the fields. does our unmarshaller crap out on these?
type supportBlob struct {
	Address      string `json:"address"`
	Amount       string `json:"amount"`
	BalanceDelta string `json:"balance_delta"`
	ClaimId      string `json:"claim_id"`
	ClaimName    string `json:"claim_name"`
	IsSpent      bool   `json:"is_spent"`
	IsTip        bool   `json:"is_tip"`
	Nout         int    `json:"nout"`
}

type TransactionListResponse struct {
	Items []struct {
		AbandonInfo   []transactionListBlob `json:"abandon_info"`
		ClaimInfo     []transactionListBlob `json:"claim_info"`
		Confirmations int64                 `json:"confirmations"`
		Date          string                `json:"date"`
		Fee           string                `json:"fee"`
		SupportInfo   []supportBlob         `json:"support_info"`
		Timestamp     int64                 `json:"timestamp"`
		Txid          string                `json:"txid"`
		UpdateInfo    []transactionListBlob `json:"update_info"`
		Value         string                `json:"value"`
	} `json:"items"`
	Page       uint64 `json:"page"`
	PageSize   uint64 `json:"page_size"`
	TotalPages uint64 `json:"total_pages"`
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
	LbrynetVersion string `json:"lbrynet_version"`
	OsRelease      string `json:"os_release"`
	OsSystem       string `json:"os_system"`
	Platform       string `json:"platform"`
	Processor      string `json:"processor"`
	PythonVersion  string `json:"python_version"`
	Version        string `json:"version"`
}

type ResolveResponse map[string]Claim

type ClaimShowResponse *Claim

type Wallet struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WalletList struct {
	Items      []Wallet `json:"items"`
	Page       uint64   `json:"page"`
	PageSize   uint64   `json:"page_size"`
	TotalPages uint64   `json:"total_pages"`
}
