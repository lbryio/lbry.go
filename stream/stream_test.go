package stream

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

var testdataBlobHashes = []string{
	"1bf7d39c45d1a38ffa74bff179bf7f67d400ff57fa0b5a0308963f08d01712b3079530a8c188e8c89d9b390c6ee06f05", // sd hash
	"a2f1841bb9c5f3b583ac3b8c07ee1a5bf9cc48923721c30d5ca6318615776c284e8936d72fa4db7fdda2e4e9598b1e6c",
	"0c9675ad7f40f29dcd41883ed9cf7e145bbb13976d9b83ab9354f4f61a87f0f7771a56724c2aa7a5ab43c68d7942e5cb",
	"a4d07d442b9907036c75b6c92db316a8b8428733bf5ec976627a48a7c862bf84db33075d54125a7c0b297bd2dc445f1c",
	"dcd2093f4a3eca9f6dd59d785d0bef068fee788481986aa894cf72ed4d992c0ff9d19d1743525de2f5c3c62f5ede1c58",
}

func TestStreamToFile(t *testing.T) {
	stream := make(Stream, len(testdataBlobHashes))
	for i, hash := range testdataBlobHashes {
		stream[i] = testdata(t, hash)
	}

	data, err := stream.Decode()
	if err != nil {
		t.Fatal(err)
	}

	expectedLen := 6990951
	actualLen := len(data)

	if actualLen != expectedLen {
		t.Errorf("file length mismatch. got %d, expected %d", actualLen, expectedLen)
	}

	expectedFileHash := sha512.Sum384(data)

	expectedSha256 := unhex(t, "51e4d03bd6d69ea17d1be3ce01fdffa44ffe053f2dbce8d42a50283b2890fea2")
	actualSha256 := sha256.Sum256(data)

	if !bytes.Equal(actualSha256[:], expectedSha256) {
		t.Errorf("file hash mismatch. got %s, expected %s", hex.EncodeToString(actualSha256[:]), hex.EncodeToString(expectedSha256))
	}

	sdBlob := &SDBlob{}
	err = sdBlob.FromBlob(stream[0])
	if err != nil {
		t.Fatal(err)
	}

	enc := NewEncoderFromSD(bytes.NewBuffer(data), sdBlob)
	newStream, err := enc.Stream()
	if err != nil {
		t.Fatal(err)
	}

	if len(newStream) != len(testdataBlobHashes) {
		t.Fatalf("stream length mismatch. got %d blobs, expected %d", len(newStream), len(testdataBlobHashes))
	}

	if enc.SourceLen() != expectedLen {
		t.Errorf("reconstructed file length mismatch. got %d, expected %d", enc.SourceLen(), expectedLen)
	}

	if !bytes.Equal(enc.SourceHash(), expectedFileHash[:]) {
		t.Errorf("reconstructed file hash mismatch. got %s, expected %s", hex.EncodeToString(enc.SourceHash()), hex.EncodeToString(expectedFileHash[:]))
	}

	for i, hash := range testdataBlobHashes {
		if newStream[i].HashHex() != hash {
			t.Errorf("blob %d hash mismatch. got %s, expected %s", i, newStream[i].HashHex(), hash)
		}
	}
}

func TestMakeStream(t *testing.T) {
	blobsToRead := 3
	totalBlobs := blobsToRead + 3

	data := make([]byte, ((totalBlobs-1)*maxBlobDataSize)+1000) // last blob is partial
	_, err := rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(data)

	enc := NewEncoder(buf)

	stream := make(Stream, blobsToRead+1) // +1 for sd blob
	for i := 1; i < blobsToRead+1; i++ {  // start at 1 to skip sd blob
		stream[i], err = enc.Next()
		if err != nil {
			t.Fatal(err)
		}
	}

	sdBlob := enc.SDBlob()

	if len(sdBlob.BlobInfos) != blobsToRead {
		t.Errorf("expected %d blobs in partial sdblob, got %d", blobsToRead, len(sdBlob.BlobInfos))
	}
	if enc.SourceLen() != maxBlobDataSize*blobsToRead {
		t.Errorf("expected length of %d , got %d", maxBlobDataSize*blobsToRead, enc.SourceLen())
	}

	// now finish the stream, reusing key and IVs

	buf = bytes.NewBuffer(data) // rewind to the beginning of the data

	enc = NewEncoderFromSD(buf, sdBlob)

	reconstructedStream, err := enc.Stream()
	if err != nil {
		t.Fatal(err)
	}

	if len(reconstructedStream) != totalBlobs+1 { // +1 for the terminating blob at the end
		t.Errorf("expected %d blobs in stream, got %d", totalBlobs+1, len(reconstructedStream))
	}
	if enc.SourceLen() != len(data) {
		t.Errorf("expected length of %d , got %d", len(data), enc.SourceLen())
	}

	reconstructedSDBlob := enc.SDBlob()

	for i := 0; i < len(sdBlob.BlobInfos); i++ {
		if !bytes.Equal(sdBlob.BlobInfos[i].IV, reconstructedSDBlob.BlobInfos[i].IV) {
			t.Errorf("blob info %d of reconstructed sd blobd does not match original sd blob", i)
		}
	}
	for i := 1; i < len(stream); i++ { // start at 1 to skip sd blob
		if !bytes.Equal(stream[i], reconstructedStream[i]) {
			t.Errorf("blob %d of reconstructed stream does not match original stream", i)
		}
	}
}

func TestEncode(t *testing.T) {
	blobsToRead := 3
	totalBlobs := blobsToRead + 3

	data := make([]byte, ((totalBlobs-1)*maxBlobDataSize)+1000) // last blob is partial
	_, err := rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(data)

	enc := NewEncoder(buf)

	stream := make(Stream, blobsToRead+1) // +1 for sd blob
	for i := 1; i < blobsToRead+1; i++ {  // start at 1 to skip sd blob
		stream[i], err = enc.Next()
		if err != nil {
			t.Fatal(err)
		}
	}

	sdBlob := enc.SDBlob()

	if len(sdBlob.BlobInfos) != blobsToRead {
		t.Errorf("expected %d blobs in partial sdblob, got %d", blobsToRead, len(sdBlob.BlobInfos))
	}
	if enc.SourceLen() != maxBlobDataSize*blobsToRead {
		t.Errorf("expected length of %d , got %d", maxBlobDataSize*blobsToRead, enc.SourceLen())
	}

	// now finish the stream, reusing key and IVs
	buf = bytes.NewBuffer(data) // rewind to the beginning of the data

	enc = NewEncoderFromSD(buf, sdBlob)

	outPath := t.TempDir()
	handler := func(h string, b []byte) error {
		return os.WriteFile(path.Join(outPath, h), b, os.ModePerm)
	}
	writtenManifest, err := enc.Encode(handler)
	if err != nil {
		t.Fatal(err)
	}

	if len(writtenManifest) != totalBlobs+1 { // +1 for the terminating blob at the end
		t.Errorf("expected %d blobs in stream, got %d", totalBlobs+1, len(writtenManifest))
	}
	if enc.SourceLen() != len(data) {
		t.Errorf("expected length of %d , got %d", len(data), enc.SourceLen())
	}

	sdb, err := ioutil.ReadFile(path.Join(outPath, writtenManifest[0]))
	if err != nil {
		t.Fatal(err)
	}
	osdb := enc.SDBlob().ToBlob()

	if !bytes.Equal(osdb, sdb) {
		t.Errorf("written sd blob does not match original sd blob")
	}
	for i := 1; i < len(stream); i++ { // start at 1 to skip sd blob
		b, err := ioutil.ReadFile(path.Join(outPath, writtenManifest[i]))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(stream[i], b) {
			t.Errorf("blob %d of reconstructed stream does not match original stream", i)
		}
	}
}

func TestEmptyStream(t *testing.T) {
	enc := NewEncoder(bytes.NewBuffer(nil))
	_, err := enc.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF, got %v", err)
	}
	sd := enc.SDBlob()
	if len(sd.BlobInfos) != 1 {
		t.Errorf("expected 1 blobinfos in sd blob, got %d", len(sd.BlobInfos))
	}
	if sd.BlobInfos[0].Length != 0 {
		t.Errorf("first and only blob to be the terminator blob")
	}
}

func TestTermination(t *testing.T) {
	b := make([]byte, 12)

	enc := NewEncoder(bytes.NewBuffer(b))

	_, err := enc.Next()
	if err != nil {
		t.Error(err)
	}
	if enc.isTerminated() {
		t.Errorf("stream should not terminate until after EOF")
	}

	_, err = enc.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF, got %v", err)
	}
	if !enc.isTerminated() {
		t.Errorf("stream should be terminated after EOF")
	}

	_, err = enc.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF on all subsequent reads, got %v", err)
	}
	sd := enc.SDBlob()
	if len(sd.BlobInfos) != 2 {
		t.Errorf("expected 2 blobinfos in sd blob, got %d", len(sd.BlobInfos))
	}
}

func TestSizeHint(t *testing.T) {
	b := make([]byte, 12)

	newStream, err := NewEncoder(bytes.NewBuffer(b)).SourceSizeHint(5 * maxBlobDataSize).Stream()
	if err != nil {
		t.Fatal(err)
	}

	if cap(newStream) != 2 { // 1 for sd blob, 1 for the 12 bytes of the actual stream
		t.Fatalf("expected 2 blobs allocated, got %d", cap(newStream))
	}
}

func TestNew(t *testing.T) {
	t.Skip("TODO: test new stream creation and decryption")
}

func TestNewEncoderFromFile(t *testing.T) {
	sketchyFile := filepath.Join(t.TempDir(), `new "encoder" from file.whatever...`)
	file, err := os.OpenFile(sketchyFile, os.O_RDONLY|os.O_CREATE, 0644)
	require.NoError(t, err)
	file.Close()
	file, err = os.Open(sketchyFile)
	require.NoError(t, err)

	e := NewEncoderFromFile(file)

	if e.sd.SuggestedFileName != "new encoder from file.whatever" {
		t.Error("wrong or missing suggested_file_name in sd blob")
	}
}

func TestSetFilename(t *testing.T) {
	enc := NewEncoder(bytes.NewBuffer(nil))
	enc.SetFilename(`filename "sketchy" string`)

	assert.Equal(t, "filename sketchy string", enc.sd.SuggestedFileName)
	assert.Equal(t, `filename "sketchy" string`, enc.sd.StreamName)
}
