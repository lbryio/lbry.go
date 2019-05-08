package jsonrpc

import (
	"encoding/json"
	"reflect"

	"github.com/lbryio/lbry.go/extras/errors"
	lbryschema "github.com/lbryio/types/v1/go"

	"github.com/shopspring/decimal"
)

type Currency string

const (
	CurrencyLBC = Currency("LBC")
	CurrencyUSD = Currency("USD")
	CurrencyBTC = Currency("BTC")
)

type Fee struct {
	Currency Currency        `json:"currency"`
	Amount   decimal.Decimal `json:"amount"`
	Address  *string         `json:"address"`
}

type Support struct {
	Amount decimal.Decimal `json:"amount"`
	Nout   int             `json:"nout"`
	Txid   string          `json:"txid"`
}

type Claim struct {
	Address            string           `json:"address"`
	Amount             decimal.Decimal  `json:"amount"`
	BlocksToExpiration int              `json:"blocks_to_expiration"`
	Category           string           `json:"category"`
	ClaimID            string           `json:"claim_id"`
	ClaimSequence      int              `json:"claim_sequence"`
	Confirmations      int              `json:"confirmations"`
	DecodedClaim       bool             `json:"decoded_claim"`
	Depth              int              `json:"depth"`
	EffectiveAmount    decimal.Decimal  `json:"effective_amount"`
	ExpirationHeight   int              `json:"expiration_height"`
	Expired            bool             `json:"expired"`
	Height             int              `json:"height"`
	Hex                string           `json:"hex"`
	IsSpent            bool             `json:"is_spent"`
	Name               string           `json:"name"`
	Nout               int              `json:"nout"`
	PermanentUrl       string           `json:"permanent_url"`
	Supports           []Support        `json:"supports"`
	Txid               string           `json:"txid"`
	ValidAtHeight      int              `json:"valid_at_height"`
	Value              lbryschema.Claim `json:"value"`
	Error              *string          `json:"error,omitempty"`
	ChannelName        *string          `json:"channel_name,omitempty"`
	HasSignature       *bool            `json:"has_signature,omitempty"`
	SignatureIsValid   *bool            `json:"signature_is_valid,omitempty"`
}

type File struct {
	ClaimID           string            `json:"claim_id"`
	Completed         bool              `json:"completed"`
	DownloadDirectory string            `json:"download_directory"`
	DownloadPath      string            `json:"download_path"`
	FileName          string            `json:"file_name"`
	Key               string            `json:"key"`
	Message           string            `json:"message"`
	Metadata          *lbryschema.Claim `json:"metadata"`
	MimeType          string            `json:"mime_type"`
	Name              string            `json:"name"`
	Outpoint          string            `json:"outpoint"`
	PointsPaid        decimal.Decimal   `json:"points_paid"`
	SdHash            string            `json:"sd_hash"`
	Stopped           bool              `json:"stopped"`
	StreamHash        string            `json:"stream_hash"`
	StreamName        string            `json:"stream_name"`
	SuggestedFileName string            `json:"suggested_file_name"`
	TotalBytes        uint64            `json:"total_bytes"`
	WrittenBytes      uint64            `json:"written_bytes"`
	ChannelName       *string           `json:"channel_name,omitempty"`
	HasSignature      *bool             `json:"has_signature,omitempty"`
	SignatureIsValid  *bool             `json:"signature_is_valid,omitempty"`
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

	case reflect.TypeOf(lbryschema.Metadata_Version(0)):
		val, err := getEnumVal(lbryschema.Metadata_Version_value, data)
		return lbryschema.Metadata_Version(val), err
	case reflect.TypeOf(lbryschema.Metadata_Language(0)):
		val, err := getEnumVal(lbryschema.Metadata_Language_value, data)
		return lbryschema.Metadata_Language(val), err

	case reflect.TypeOf(lbryschema.Stream_Version(0)):
		val, err := getEnumVal(lbryschema.Stream_Version_value, data)
		return lbryschema.Stream_Version(val), err

	case reflect.TypeOf(lbryschema.Claim_Version(0)):
		val, err := getEnumVal(lbryschema.Claim_Version_value, data)
		return lbryschema.Claim_Version(val), err
	case reflect.TypeOf(lbryschema.Claim_ClaimType(0)):
		val, err := getEnumVal(lbryschema.Claim_ClaimType_value, data)
		return lbryschema.Claim_ClaimType(val), err

	case reflect.TypeOf(lbryschema.Fee_Version(0)):
		val, err := getEnumVal(lbryschema.Fee_Version_value, data)
		return lbryschema.Fee_Version(val), err
	case reflect.TypeOf(lbryschema.Fee_Currency(0)):
		val, err := getEnumVal(lbryschema.Fee_Currency_value, data)
		return lbryschema.Fee_Currency(val), err

	case reflect.TypeOf(lbryschema.Source_Version(0)):
		val, err := getEnumVal(lbryschema.Source_Version_value, data)
		return lbryschema.Source_Version(val), err
	case reflect.TypeOf(lbryschema.Source_SourceTypes(0)):
		val, err := getEnumVal(lbryschema.Source_SourceTypes_value, data)
		return lbryschema.Source_SourceTypes(val), err

	case reflect.TypeOf(lbryschema.KeyType(0)):
		val, err := getEnumVal(lbryschema.KeyType_value, data)
		return lbryschema.KeyType(val), err

	case reflect.TypeOf(lbryschema.Signature_Version(0)):
		val, err := getEnumVal(lbryschema.Signature_Version_value, data)
		return lbryschema.Signature_Version(val), err

	case reflect.TypeOf(lbryschema.Certificate_Version(0)):
		val, err := getEnumVal(lbryschema.Certificate_Version_value, data)
		return lbryschema.Certificate_Version(val), err
	}

	return data, nil
}

type CommandsResponse []string

type WalletBalanceResponse decimal.Decimal

type VersionResponse struct {
	Build             string `json:"build"`
	LbrynetVersion    string `json:"lbrynet_version"`
	LbryschemaVersion string `json:"lbryschema_version"`
	LbryumVersion     string `json:"lbryum_version"`
	OsRelease         string `json:"os_release"`
	OsSystem          string `json:"os_system"`
	Platform          string `json:"platform"`
	Processor         string `json:"processor"`
	PythonVersion     string `json:"python_version"`
}
type StatusResponse struct {
	BlobManager struct {
		FinishedBlobs int `json:"finished_blobs"`
	} `json:"blob_manager"`
	ConnectionStatus struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"connection_status"`
	Dht struct {
		NodeID              string `json:"node_id"`
		PeersInRoutingTable int    `json:"peers_in_routing_table"`
	} `json:"dht"`
	FileManager struct {
		ManagedFiles int `json:"managed_files"`
	} `json:"file_manager"`
	HashAnnouncer struct {
		AnnounceQueueSize int `json:"announce_queue_size"`
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
		FileManager         bool `json:"file_manager"`
		HashAnnouncer       bool `json:"hash_announcer"`
		PaymentRateManager  bool `json:"payment_rate_manager"`
		PeerProtocolServer  bool `json:"peer_protocol_server"`
		RateLimiter         bool `json:"rate_limiter"`
		StreamIdentifier    bool `json:"stream_identifier"`
		Upnp                bool `json:"upnp"`
		Wallet              bool `json:"wallet"`
	} `json:"startup_status"`
	Wallet struct {
		BestBlockchain string `json:"best_blockchain"`
		Blocks         int    `json:"blocks"`
		BlocksBehind   int    `json:"blocks_behind"`
		IsEncrypted    bool   `json:"is_encrypted"`
	} `json:"wallet"`
}

type ClaimListResponse struct {
	Claims                []Claim   `json:"claims"`
	LastTakeoverHeight    int       `json:"last_takeover_height"`
	SupportsWithoutClaims []Support `json:"supports_without_claims"`
}
type ClaimListMineResponse []Claim
type ClaimShowResponse Claim
type ClaimAbandonResponse struct {
	Txid string  `json:"txid"`
	Fee  float64 `json:"fee"`
}

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

type ResolveResponse map[string]ResolveResponseItem
type ResolveResponseItem struct {
	Certificate     *Claim  `json:"certificate,omitempty"`
	Claim           *Claim  `json:"claim,omitempty"`
	ClaimsInChannel *uint64 `json:"claims_in_channel,omitempty"`
	Error           *string `json:"error,omitempty"`
}

type ChannelNewResponse struct {
	ClaimID string          `json:"claim_id"`
	Fee     decimal.Decimal `json:"fee"`
	Nout    int             `json:"nout"`
	Success bool            `json:"success"`
	Tx      string          `json:"tx"`
	Txid    string          `json:"txid"`
}

type ChannelListSingleResponse struct {
	Address            string            `json:"address"`
	Amount             decimal.Decimal   `json:"amount"`
	BlocksToExpiration int               `json:"blocks_to_expiration"`
	CanSign            bool              `json:"can_sign"`
	Category           string            `json:"category"`
	ClaimID            string            `json:"claim_id"`
	Confirmations      int               `json:"confirmations"`
	DecodedClaim       bool              `json:"decoded_claim"`
	ExpirationHeight   int               `json:"expiration_height"`
	Expired            bool              `json:"expired"`
	HasSignature       bool              `json:"has_signature"`
	Height             int               `json:"height"`
	Hex                string            `json:"hex"`
	IsPending          bool              `json:"is_pending"`
	IsSpent            bool              `json:"is_spent"`
	Name               string            `json:"name"`
	Nout               int               `json:"nout"`
	Txid               string            `json:"txid"`
	Value              *lbryschema.Claim `json:"value"`
}

type ChannelListResponse []ChannelListSingleResponse

type WalletListResponse []string

type PublishResponse struct {
	ClaimID string          `json:"claim_id"`
	Fee     decimal.Decimal `json:"fee"`
	Nout    int             `json:"nout"`
	Tx      string          `json:"tx"`
	Txid    string          `json:"txid"`
}

type BlobAnnounceResponse bool

type WalletPrefillAddressesResponse struct {
	Broadcast bool   `json:"broadcast"`
	Complete  bool   `json:"complete"`
	Hex       string `json:"hex"`
}

type UTXOListResponse []struct {
	Address    string          `json:"address"`
	Amount     decimal.Decimal `json:"amount"`
	Height     int             `json:"height"`
	IsClaim    bool            `json:"is_claim"`
	IsCoinbase bool            `json:"is_coinbase"`
	IsSupport  bool            `json:"is_support"`
	IsUpdate   bool            `json:"is_update"`
	Nout       int             `json:"nout"`
	Txid       string          `json:"txid"`
}

type WalletNewAddressResponse string

type WalletUnusedAddressResponse string

type NumClaimsInChannelResponse map[string]struct {
	ClaimsInChannel uint64 `json:"claims_in_channel,omitempty"`
	Error           string `json:"error,omitempty"`
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
	Certificates     uint64  `json:"certificates"`
	Coins            float64 `json:"coins"`
	Encrypted        bool    `json:"encrypted"`
	ID               string  `json:"id"`
	IsDefaultAccount bool    `json:"is_default_account"`
	Name             string  `json:"name"`
	PublicKey        string  `json:"public_key"`
	Satoshis         uint64  `json:"satoshis"`
}

type AccountListResponse struct {
	LBCMainnet *[]Account `json:"lbc_mainnet"`
	LBCTestnet *[]Account `json:"lbc_testnet"`
	LBCRegtest *[]Account `json:"lbc_regtest"`
}
type AccountBalanceResponse string

type AccountFundResponse struct {
	Height int64  `json:"height"`
	Hex    string `json:"hex"`
	Inputs []struct {
		Address       string `json:"address"`
		Amount        string `json:"amount"`
		Confirmations int64  `json:"confirmations"`
		Height        int64  `json:"height"`
		IsChange      bool   `json:"is_change"`
		IsMine        bool   `json:"is_mine"`
		Nout          uint64 `json:"nout"`
		Txid          string `json:"txid"`
	} `json:"inputs"`
	Outputs []struct {
		Address       string `json:"address"`
		Amount        string `json:"amount"`
		Confirmations int64  `json:"confirmations"`
		Height        int64  `json:"height"`
		Nout          uint64 `json:"nout"`
		Txid          string `json:"txid"`
	} `json:"outputs"`
	TotalFee    string `json:"total_fee"`
	TotalOutput string `json:"total_output"`
	Txid        string `json:"txid"`
}
