package jsonrpc

import (
	"encoding/json"
	"github.com/go-errors/errors"
	"reflect"

	lbryschema "github.com/lbryio/lbryschema.go/pb"
)

func getEnumVal(enum map[string]int32, data interface{}) (int32, error) {
	s, ok := data.(string)
	if !ok {
		return 0, errors.New("expected a string")
	}
	val, ok := enum[s]
	if !ok {
		return 0, errors.New("invalid enum key")
	}
	return val, nil
}

func fixDecodeProto(src, dest reflect.Type, data interface{}) (interface{}, error) {
	switch dest {
	case reflect.TypeOf(uint64(0)):
		if n, ok := data.(json.Number); ok {
			val, err := n.Int64()
			if err != nil {
				return nil, err
			} else if val < 0 {
				return nil, errors.New("must be unsigned int")
			}
			return uint64(val), nil
		}
	case reflect.TypeOf([]byte{}):
		if s, ok := data.(string); ok {
			return []byte(s), nil
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
	}

	return data, nil
}

type CommandsResponse []string

type StatusResponse struct {
	BlockchainStatus struct {
		BestBlockhash string `json:"best_blockhash"`
		Blocks        int    `json:"blocks"`
		BlocksBehind  int    `json:"blocks_behind"`
	} `json:"blockchain_status"`
	BlocksBehind     int `json:"blocks_behind"`
	ConnectionStatus struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"connection_status"`
	InstallationID string `json:"installation_id"`
	IsFirstRun     bool   `json:"is_first_run"`
	IsRunning      bool   `json:"is_running"`
	LbryID         string `json:"lbry_id"`
	StartupStatus  struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"startup_status"`
}

type GetResponse struct {
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
	PointsPaid        float64           `json:"points_paid"`
	SdHash            string            `json:"sd_hash"`
	Stopped           bool              `json:"stopped"`
	StreamHash        string            `json:"stream_hash"`
	StreamName        string            `json:"stream_name"`
	SuggestedFileName string            `json:"suggested_file_name"`
	TotalBytes        uint64            `json:"total_bytes"`
	WrittenBytes      uint64            `json:"written_bytes"`
}
