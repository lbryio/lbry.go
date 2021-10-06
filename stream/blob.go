package stream

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/hex"

	"github.com/cockroachdb/errors"
)

const (
	MaxBlobSize       = 2097152 // 2mb, or 2 * 2^20
	BlobHashSize      = sha512.Size384
	BlobHashHexLength = BlobHashSize * 2 // in hex, each byte is 2 chars
)

type Blob []byte

var ErrBlobTooBig = errors.Newf("blob must be at most %d bytes", MaxBlobSize)
var ErrBlobEmpty = errors.New("blob is empty")

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

// HashHex returns the blob hash as a hex string
func (b Blob) HashHex() string {
	return hex.EncodeToString(b.Hash())
}

// ValidForSend returns true if the blob size is within the limits
func (b Blob) ValidForSend() error {
	if b.Size() > MaxBlobSize {
		return errors.WithStack(ErrBlobTooBig)
	}
	if b.Size() == 0 {
		return errors.WithStack(ErrBlobEmpty)
	}
	return nil
}

func NewBlob(data, key, iv []byte) (Blob, error) {
	if len(data) == 0 {
		// this is here to match python behavior. in theory we could encrypt an empty blob
		return nil, errors.WithStack(errors.New("cannot encrypt empty slice"))
	}
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(iv) != blockCipher.BlockSize() {
		return nil, errors.WithStack(errors.New("IV length must equal to block size"))
	}

	cbc := cipher.NewCBCEncrypter(blockCipher, iv)
	plaintext, err := pkcs7Pad(data, blockCipher.BlockSize())
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	cbc.CryptBlocks(ciphertext, plaintext)
	return ciphertext, nil
}

// DecryptBlob decrypts a blob
func DecryptBlob(b Blob, key, iv []byte) ([]byte, error) {
	return b.Plaintext(key, iv)
}

func (b Blob) Plaintext(key, iv []byte) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(iv) != blockCipher.BlockSize() {
		return nil, errors.WithStack(errors.New("IV length must equal to block size"))
	}

	cbc := cipher.NewCBCDecrypter(blockCipher, iv)
	plaintext := make([]byte, len(b))
	cbc.CryptBlocks(plaintext, b)

	plaintext, err = pkcs7Unpad(plaintext, blockCipher.BlockSize())
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// https://github.com/fullsailor/pkcs7/blob/master/pkcs7.go#L468
func pkcs7Pad(data []byte, blockLen int) ([]byte, error) {
	if blockLen < 1 {
		return nil, errors.WithStack(errors.Newf("invalid block length %d", blockLen))
	}
	padLen := blockLen - (len(data) % blockLen)
	if padLen == 0 {
		padLen = blockLen
	}
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)
	copy(padded[len(padded)-padLen:], bytes.Repeat([]byte{byte(padLen)}, padLen))
	return padded, nil
}

func pkcs7Unpad(data []byte, blockLen int) ([]byte, error) {
	if blockLen < 1 {
		return nil, errors.WithStack(errors.Newf("invalid block length %d", blockLen))
	}
	if len(data)%blockLen != 0 || len(data) == 0 {
		return nil, errors.WithStack(errors.Newf("invalid data length %d", len(data)))
	}

	// the last byte is the length of padding
	padLen := int(data[len(data)-1])

	// check padding integrity, all bytes should be the same
	pad := data[len(data)-padLen:]
	for _, padbyte := range pad {
		if padbyte != byte(padLen) {
			return nil, errors.WithStack(errors.New("invalid padding"))
		}
	}

	return data[:len(data)-padLen], nil
}
