package stream

import (
	"encoding/hex"
	"encoding/json"

	"github.com/cockroachdb/errors"
)

// inspired by https://blog.gopheracademy.com/advent-2016/advanced-encoding-decoding/

type SDBlobAlias SDBlob

type JSONSDBlob struct {
	StreamName string `json:"stream_name"`
	SDBlobAlias
	Key               string `json:"key"`
	SuggestedFileName string `json:"suggested_file_name"`
	StreamHash        string `json:"stream_hash"`
}

func (s SDBlob) MarshalJSON() ([]byte, error) {
	var tmp JSONSDBlob

	tmp.StreamName = hex.EncodeToString([]byte(s.StreamName))
	tmp.StreamHash = hex.EncodeToString(s.StreamHash)
	tmp.SuggestedFileName = hex.EncodeToString([]byte(s.SuggestedFileName))
	tmp.Key = hex.EncodeToString(s.Key)

	tmp.SDBlobAlias = SDBlobAlias(s)

	return json.Marshal(tmp)
}

func (s *SDBlob) UnmarshalJSON(b []byte) error {
	var tmp JSONSDBlob
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return errors.WithStack(err)
	}

	*s = SDBlob(tmp.SDBlobAlias)

	str, err := hex.DecodeString(tmp.StreamName)
	if err != nil {
		return errors.WithStack(err)
	}
	s.StreamName = string(str)

	str, err = hex.DecodeString(tmp.SuggestedFileName)
	if err != nil {
		return errors.WithStack(err)
	}
	s.SuggestedFileName = string(str)

	s.StreamHash, err = hex.DecodeString(tmp.StreamHash)
	if err != nil {
		return errors.WithStack(err)
	}

	s.Key, err = hex.DecodeString(tmp.Key)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type BlobInfoAlias BlobInfo

type JSONBlobInfo struct {
	BlobInfoAlias
	BlobHash string `json:"blob_hash,omitempty"`
	IV       string `json:"iv"`
}

func (bi BlobInfo) MarshalJSON() ([]byte, error) {
	var tmp JSONBlobInfo

	tmp.IV = hex.EncodeToString(bi.IV)
	if len(bi.BlobHash) > 0 {
		tmp.BlobHash = hex.EncodeToString(bi.BlobHash)
	}

	tmp.BlobInfoAlias = BlobInfoAlias(bi)

	return json.Marshal(tmp)
}

func (bi *BlobInfo) UnmarshalJSON(b []byte) error {
	var tmp JSONBlobInfo
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return errors.WithStack(err)
	}

	*bi = BlobInfo(tmp.BlobInfoAlias)

	bi.BlobHash, err = hex.DecodeString(tmp.BlobHash)
	if err != nil {
		return errors.WithStack(err)
	}

	bi.IV, err = hex.DecodeString(tmp.IV)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
