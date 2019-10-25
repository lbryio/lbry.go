package stream

import (
	"bytes"
	"math"
	"strings"

	"github.com/lbryio/lbry.go/v2/extras/errors"
)

type Stream []Blob

// -1 to leave room for padding, since there must be at least one byte of pkcs7 padding
const maxBlobDataSize = MaxBlobSize - 1

// New creates a new Stream from a byte slice
func New(data []byte) (Stream, error) {
	key := randIV()
	ivs := make([][]byte, numContentBlobs(data)+1) // +1 for terminating 0-length blob
	for i := range ivs {
		ivs[i] = randIV()
	}

	return makeStream(data, key, ivs, "", "")
}

// Reconstruct creates a stream from the given data using predetermined IVs and key from the SD blob
// NOTE: this will assume that all blobs except the last one are at max length. in theory this is not
// required, but in practice this is always true. if this is false, streams may not match exactly
func Reconstruct(data []byte, sdBlob SDBlob) (Stream, error) {
	ivs := make([][]byte, len(sdBlob.BlobInfos))
	for i := range ivs {
		ivs[i] = sdBlob.BlobInfos[i].IV
	}

	return makeStream(data, sdBlob.Key, ivs, sdBlob.StreamName, sdBlob.SuggestedFileName)
}

func makeStream(data, key []byte, ivs [][]byte, streamName, suggestedFilename string) (Stream, error) {
	var err error

	numBlobs := numContentBlobs(data)
	if len(ivs) != numBlobs+1 { // +1 for terminating 0-length blob
		return nil, errors.Err("incorrect number of IVs provided")
	}

	s := make(Stream, numBlobs+1) // +1 for sd blob
	for i := 0; i < numBlobs; i++ {
		start := i * maxBlobDataSize
		end := start + maxBlobDataSize
		if end > len(data) {
			end = len(data)
		}
		s[i+1], err = NewBlob(data[start:end], key, ivs[i])
		if err != nil {
			return nil, err
		}
	}

	sd := newSdBlob(s[1:], key, ivs, streamName, suggestedFilename)
	jsonSD, err := sd.ToBlob()
	if err != nil {
		return nil, err
	}

	// COMPATIBILITY HACK to make json output match python's json. this can be
	// removed when we implement canonical JSON encoding
	jsonSD = []byte(strings.Replace(string(jsonSD), ",", ", ", -1))
	jsonSD = []byte(strings.Replace(string(jsonSD), ":", ": ", -1))

	s[0] = jsonSD
	return s, nil
}

func (s Stream) Data() ([]byte, error) {
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

//numContentBlobs returns the number of content blobs required to store the data
func numContentBlobs(data []byte) int {
	return int(math.Ceil(float64(len(data)) / float64(maxBlobDataSize)))
}
