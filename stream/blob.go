package stream

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/lbryio/lbry.go/errors"
)

const MaxBlobSize = 2 * 1024 * 1024

type Blob []byte

var ErrBlobTooBig = errors.Base("blob must be at most " + strconv.Itoa(MaxBlobSize) + " bytes")
var ErrBlobEmpty = errors.Base("blob is empty")

func (b Blob) Size() int {
	return len(b)
}

// Hash returns a hash of the blob data
func (b Blob) Hash() []byte {
	if b.Size() == 0 {
		return nil
	}
	hashBytes := sha512.Sum384(b)
	return hashBytes[:]
}

// HexHash returns th blob hash as a hex string
func (b Blob) HashHex() string {
	return hex.EncodeToString(b.Hash())
}

// ValidForSend returns true if the blob size is within the limits
func (b Blob) ValidForSend() error {
	if b.Size() > MaxBlobSize {
		return ErrBlobTooBig
	}
	if b.Size() == 0 {
		return ErrBlobEmpty
	}
	return nil
}

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
	ivFunc            func() []byte
}

// ToBlob converts the SDBlob to a normal data Blob
func (s SDBlob) ToBlob() (Blob, error) {
	b, err := json.Marshal(s)
	return Blob(b), err
}

// FromBlob unmarshals a data Blob that should contain SDBlob data
func (s *SDBlob) FromBlob(b Blob) error {
	return json.Unmarshal(b, s)
}

func NewSdBlob(blobs []Blob) *SDBlob {
	return newSdBlob(blobs, nil, nil)
}

func newSdBlob(blobs []Blob, key []byte, ivs [][]byte) *SDBlob {
	sd := &SDBlob{}

	if key == nil {
		key = randIV()
	}
	sd.Key = key

	if ivs == nil {
		ivs = make([][]byte, len(blobs))
		for i := range ivs {
			ivs[i] = randIV()
		}
	}

	for i, b := range blobs {
		sd.addBlob(b, ivs[i])
	}

	sd.updateStreamHash()

	return sd
}

// addBlob adds the blob's info to stream
func (s *SDBlob) addBlob(b Blob, iv []byte) {
	if iv == nil {
		iv = s.nextIV()
	}
	s.BlobInfos = append(s.BlobInfos, BlobInfo{
		BlobNum:  len(s.BlobInfos),
		Length:   b.Size(),
		BlobHash: b.Hash(),
		IV:       iv,
	})
}

// nextIV returns the next IV using ivFunc, or a random IV if no ivFunc is set
func (s SDBlob) nextIV() []byte {
	if s.ivFunc != nil {
		return s.ivFunc()
	}
	return randIV()
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
	blob := make([]byte, aes.BlockSize)
	_, err := rand.Read(blob)
	if err != nil {
		panic("failed to make random blob")
	}
	return blob
}

// NullIV returns an IV of 0s
func NullIV() []byte {
	return make([]byte, aes.BlockSize)
}
