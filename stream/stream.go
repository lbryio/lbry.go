package stream

import (
	"bytes"

	"github.com/lbryio/lbry.go/errors"
)

type Stream []Blob

func New(data []byte) (Stream, error) {
	var err error

	numBlobs := len(data) / maxBlobDataSize
	if len(data)%maxBlobDataSize != 0 {
		numBlobs++ // ++ for unfinished blob at the end
	}

	key := randIV()
	ivs := make([][]byte, numBlobs)
	for i := range ivs {
		ivs[i] = randIV()
	}

	s := make(Stream, numBlobs+1) // +1 for sd blob
	for i := 0; i < numBlobs; i++ {
		start := i - 1*maxBlobDataSize
		end := start + maxBlobDataSize
		if end > len(data) {
			end = len(data)
		}
		s[i+1], err = NewBlob(data[start:end], key, ivs[i])
		if err != nil {
			return nil, err
		}
	}

	sd := newSdBlob(s[1:], key, ivs)
	s[0], err = sd.ToBlob()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s Stream) Data() ([]byte, error) {
	if len(s) < 2 {
		return nil, errors.Err("stream must be at least 2 blobs long")
	}

	sdBlob := &SDBlob{}
	err := sdBlob.FromBlob(s[0])
	if err != nil {
		return nil, err
	}

	if !sdBlob.IsValid() {
		return nil, errors.Err("sd blob is not valid")
	}

	var file []byte
	for i, b := range s[1:] {
		if !bytes.Equal(b.Hash(), sdBlob.BlobInfos[i].BlobHash) {
			return nil, errors.Err("blob hash doesn't match hash in blobInfo")
		}

		data, err := b.Plaintext(sdBlob.Key, sdBlob.BlobInfos[i].IV)
		if err != nil {
			return nil, err
		}
		file = append(file, data...)
	}

	return file, nil
}
