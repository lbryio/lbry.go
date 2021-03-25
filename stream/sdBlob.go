package stream

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

const streamTypeLBRYFile = "lbryfile"

// BlobInfo is the stream descriptor info for a single blob in a stream
// Encoding to and from JSON is customized to match existing behavior (see json.go in package)
type BlobInfo struct {
	Length   int    `json:"length"`
	BlobNum  int    `json:"blob_num"`
	BlobHash []byte `json:"-"`
	IV       []byte `json:"-"`
}

// Hash returns the hash of the blob info for calculating the stream hash
func (bi BlobInfo) Hash() []byte {
	sum := sha512.New384()
	if bi.Length > 0 {
		sum.Write([]byte(hex.EncodeToString(bi.BlobHash)))
	}
	sum.Write([]byte(strconv.Itoa(bi.BlobNum)))
	sum.Write([]byte(hex.EncodeToString(bi.IV)))
	sum.Write([]byte(strconv.Itoa(bi.Length)))
	return sum.Sum(nil)
}

// SDBlob contains information about the rest of the blobs in the stream
// Encoding to and from JSON is customized to match existing behavior (see json.go in package)
type SDBlob struct {
	StreamName        string     `json:"-"`
	BlobInfos         []BlobInfo `json:"blobs"`
	StreamType        string     `json:"stream_type"`
	Key               []byte     `json:"-"`
	SuggestedFileName string     `json:"-"`
	StreamHash        []byte     `json:"-"`
}

// Hash returns a hash of the SD blob data
func (s SDBlob) Hash() []byte {
	hashBytes := sha512.Sum384(s.ToBlob())
	return hashBytes[:]
}

// HashHex returns the SD blob hash as a hex string
func (s SDBlob) HashHex() string {
	return hex.EncodeToString(s.Hash())
}

// ToBlob converts the SDBlob to a normal data Blob
func (s SDBlob) ToBlob() Blob {
	jsonSD, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}

	// COMPATIBILITY HACK to make json output match python's json. this can be
	// removed when we implement canonical JSON encoding
	jsonSD = []byte(strings.Replace(string(jsonSD), ",", ", ", -1))
	jsonSD = []byte(strings.Replace(string(jsonSD), ":", ": ", -1))

	return jsonSD
}

// FromBlob unmarshals a data Blob that should contain SDBlob data
func (s *SDBlob) FromBlob(b Blob) error {
	return json.Unmarshal(b, s)
}

// addBlob adds the blob's info to stream
func (s *SDBlob) addBlob(b Blob, iv []byte) {
	if len(iv) == 0 {
		panic("empty IV")
	}
	s.BlobInfos = append(s.BlobInfos, BlobInfo{
		BlobNum:  len(s.BlobInfos),
		Length:   b.Size(),
		BlobHash: b.Hash(),
		IV:       iv,
	})
}

// IsValid returns true if the set StreamHash matches the current hash of the stream data
func (s SDBlob) IsValid() bool {
	return bytes.Equal(s.StreamHash, s.computeStreamHash())
}

// updateStreamHash sets the stream hash to the current hash of the stream data
func (s *SDBlob) updateStreamHash() {
	s.StreamHash = s.computeStreamHash()
}

// computeStreamHash calculates the stream hash for the stream
func (s *SDBlob) computeStreamHash() []byte {
	return streamHash(
		hex.EncodeToString([]byte(s.StreamName)),
		hex.EncodeToString(s.Key),
		hex.EncodeToString([]byte(s.SuggestedFileName)),
		s.BlobInfos,
	)
}

func (s SDBlob) fileSize() int {
	size := 0
	for _, bi := range s.BlobInfos {
		size += bi.Length
	}
	return size
}

// streamHash calculates the stream hash, given the stream's fields and blobs
func streamHash(hexStreamName, hexKey, hexSuggestedFileName string, blobInfos []BlobInfo) []byte {
	blobSum := sha512.New384()
	for _, b := range blobInfos {
		blobSum.Write(b.Hash())
	}

	sum := sha512.New384()
	sum.Write([]byte(hexStreamName))
	sum.Write([]byte(hexKey))
	sum.Write([]byte(hexSuggestedFileName))
	sum.Write(blobSum.Sum(nil))
	return sum.Sum(nil)
}

// randIV returns a random AES IV
func randIV() []byte {
	iv := make([]byte, aes.BlockSize)
	_, err := rand.Read(iv)
	if err != nil {
		panic("failed to make random iv")
	}
	return iv
}

// NullIV returns an IV of 0s
func NullIV() []byte {
	return make([]byte, aes.BlockSize)
}
