package stream

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"math"

	"github.com/lbryio/lbry.go/v2/extras/errors"
)

type Stream []Blob

// -1 to leave room for padding, since there must be at least one byte of pkcs7 padding
const maxBlobDataSize = MaxBlobSize - 1

// New creates a new Stream from a stream of bytes.
func New(src io.Reader) (Stream, error) {
	return NewEncoder(src).Stream()
}

// Data returns the file data that a stream encapsulates.
//
// Deprecated: use Decode() instead. It's a more accurate name. Data() will be removed in the future.
func (s Stream) Data() ([]byte, error) {
	return s.Decode()
}

// Decode returns the file data that a stream encapsulates
//
// TODO: this should use io.Writer instead of returning bytes
func (s Stream) Decode() ([]byte, error) {
	if len(s) < 2 {
		return nil, errors.Err("stream must be at least 2 blobs long") // sd blob and content blob
	}

	sdBlob := &SDBlob{}
	err := sdBlob.FromBlob(s[0])
	if err != nil {
		return nil, err
	}

	if !sdBlob.IsValid() {
		return nil, errors.Err("sd blob is not valid")
	}

	if sdBlob.BlobInfos[len(sdBlob.BlobInfos)-1].Length != 0 {
		return nil, errors.Err("sd blob is missing the terminating 0-length blob")
	}

	if len(s[1:]) != len(sdBlob.BlobInfos)-1 { // -1 for terminating 0-length blob
		return nil, errors.Err("number of blobs in stream does not match number of blobs in sd info")
	}

	var file []byte
	for i, blobInfo := range sdBlob.BlobInfos {
		if blobInfo.Length == 0 {
			if i != len(sdBlob.BlobInfos)-1 {
				return nil, errors.Err("got 0-length blob before end of stream")
			}
			break
		}

		if blobInfo.BlobNum != i {
			return nil, errors.Err("blobs are out of order in sd blob")
		}

		blob := s[i+1]

		if !bytes.Equal(blob.Hash(), blobInfo.BlobHash) {
			return nil, errors.Err("blob hash doesn't match hash in blobInfo")
		}

		data, err := blob.Plaintext(sdBlob.Key, blobInfo.IV)
		if err != nil {
			return nil, err
		}
		file = append(file, data...)
	}

	return file, nil
}

// Encoder reads bytes from a source and returns blobs of the stream
type Encoder struct {
	// source data to be encoded into a stream
	src io.Reader
	// preset IVs to use for encrypting blobs
	ivs [][]byte
	// an optionals hint about the total size of the source data
	// encoder will use this to preallocate space for blobs
	srcSizeHint int

	// buffer for reading bytes from reader
	buf []byte
	// sd blob that gets built as stream is encoded
	sd *SDBlob
	// number of bytes read from src
	srcLen int
	// running hash bytes read from src
	srcHash hash.Hash
}

// NewEncoder creates a new stream encoder
func NewEncoder(src io.Reader) *Encoder {
	return &Encoder{
		src: src,

		buf: make([]byte, maxBlobDataSize),
		sd: &SDBlob{
			StreamType: streamTypeLBRYFile,
			Key:        randIV(),
		},
		srcHash: sha512.New384(),
	}
}

// NewEncoderWithIVs creates a new encoder that uses preset cryptographic material
func NewEncoderWithIVs(src io.Reader, key []byte, ivs [][]byte) *Encoder {
	e := NewEncoder(src)
	e.sd.Key = key
	e.ivs = ivs
	return e
}

// NewEncoderFromSD creates a new encoder that reuses cryptographic material from an sd blob
// This can be used to reconstruct a stream exactly from a file
// NOTE: this will assume that all blobs except the last one are at max length. in theory this is not
// required, but in practice this is always true. if this is false, streams may not match exactly
func NewEncoderFromSD(src io.Reader, sdBlob *SDBlob) *Encoder {
	ivs := make([][]byte, len(sdBlob.BlobInfos))
	for i := range ivs {
		ivs[i] = sdBlob.BlobInfos[i].IV
	}

	e := NewEncoderWithIVs(src, sdBlob.Key, ivs)
	e.sd.StreamName = sdBlob.StreamName
	e.sd.SuggestedFileName = sdBlob.SuggestedFileName
	return e
}

// TODO: consider making a NewPartialEncoder that also copies blobinfos from sdBlobs and seeks forward in the data
// this would avoid re-creating blobs that were created in the past

// Next reads the next chunk of data, encodes it into a blob, and adds it to the stream
// When the source is fully consumed, Next() makes sure the stream is terminated (i.e. the sd blob
// ends with an empty terminating blob) and returns io.EOF
func (e *Encoder) Next() (Blob, error) {
	n, err := e.src.Read(e.buf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			e.ensureTerminated()
		}
		return nil, err
	}

	e.srcLen += n
	e.srcHash.Write(e.buf[:n])
	iv := e.nextIV()

	blob, err := NewBlob(e.buf[:n], e.sd.Key, iv)
	if err != nil {
		return nil, err
	}

	e.sd.addBlob(blob, iv)

	return blob, nil
}

// Stream creates the whole stream in one call
// TODO: Can be refactored to use Encode method
func (e *Encoder) Stream() (Stream, error) {
	s := make(Stream, 1, 1+int(math.Ceil(float64(e.srcSizeHint)/maxBlobDataSize))) // len starts at 1 and cap is +1 to leave room for sd blob

	for {
		blob, err := e.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		s = append(s, blob)
	}

	s[0] = e.SDBlob().ToBlob()

	if cap(s) > len(s) {
		// size hint was too big. copy stream to smaller underlying array to free memory
		// this might be premature optimization...
		s = append(Stream(nil), s[:]...)
	}

	return s, nil
}

// Encode splits the source into blobs and feeds them into handler function
func (e *Encoder) Encode(handler func(string, []byte) error) ([]string, error) {
	manifest := []string{}

	for {
		blob, err := e.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		err = handler(blob.HashHex(), blob)
		if err != nil {
			return nil, fmt.Errorf("cannot process blob: %w", err)
		}
		manifest = append(manifest, blob.HashHex())
	}

	sdb := e.SDBlob().ToBlob()
	h := sdb.HashHex()
	err := handler(h, sdb)
	if err != nil {
		return nil, fmt.Errorf("cannot handle SD blob: %w", err)
	}
	manifest = append([]string{h}, manifest...)

	return manifest, nil
}

// SDBlob returns the sd blob so far
func (e *Encoder) SDBlob() *SDBlob {
	e.sd.updateStreamHash()
	return e.sd
}

// SourceLen returns the number of bytes read from source
func (e *Encoder) SourceLen() int {
	return e.srcLen
}

// SourceLen returns a hash of the bytes read from source
func (e *Encoder) SourceHash() []byte {
	return e.srcHash.Sum(nil)
}

// SourceSizeHint sets a hint about the total size of the source
// This helps allocate RAM more efficiently.
// If the hint is wrong, it still works fine but there will be a small performance penalty.
func (e *Encoder) SourceSizeHint(size int) *Encoder {
	e.srcSizeHint = size
	return e
}

func (e *Encoder) isTerminated() bool {
	return len(e.sd.BlobInfos) >= 1 && e.sd.BlobInfos[len(e.sd.BlobInfos)-1].Length == 0
}

func (e *Encoder) ensureTerminated() {
	if !e.isTerminated() {
		e.sd.addBlob(Blob{}, e.nextIV())
	}
}

// nextIV returns the next preset IV if there is one
func (e *Encoder) nextIV() []byte {
	if len(e.ivs) == 0 {
		return randIV()
	}

	iv := e.ivs[0]
	e.ivs = e.ivs[1:]
	return iv
}
