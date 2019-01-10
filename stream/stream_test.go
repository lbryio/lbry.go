package stream

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestStreamToFile(t *testing.T) {
	blobHashes := []string{
		"1bf7d39c45d1a38ffa74bff179bf7f67d400ff57fa0b5a0308963f08d01712b3079530a8c188e8c89d9b390c6ee06f05", // sd hash
		"a2f1841bb9c5f3b583ac3b8c07ee1a5bf9cc48923721c30d5ca6318615776c284e8936d72fa4db7fdda2e4e9598b1e6c",
		"0c9675ad7f40f29dcd41883ed9cf7e145bbb13976d9b83ab9354f4f61a87f0f7771a56724c2aa7a5ab43c68d7942e5cb",
		"a4d07d442b9907036c75b6c92db316a8b8428733bf5ec976627a48a7c862bf84db33075d54125a7c0b297bd2dc445f1c",
		"dcd2093f4a3eca9f6dd59d785d0bef068fee788481986aa894cf72ed4d992c0ff9d19d1743525de2f5c3c62f5ede1c58",
	}

	stream := make(Stream, len(blobHashes))
	for i, hash := range blobHashes {
		stream[i] = testdata(t, hash)
	}

	data, err := stream.Data()
	if err != nil {
		t.Fatal(err)
	}

	expectedLen := 6990951
	actualLen := len(data)

	if actualLen != expectedLen {
		t.Errorf("file length mismatch. got %d, expected %d", actualLen, expectedLen)
	}

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

	newStream, err := Reconstruct(data, *sdBlob)
	if err != nil {
		t.Fatal(err)
	}

	if len(newStream) != len(blobHashes) {
		t.Fatalf("stream length mismatch. got %d blobs, expected %d", len(newStream), len(blobHashes))
	}

	for i, hash := range blobHashes {
		if newStream[i].HashHex() != hash {
			t.Errorf("blob %d hash mismatch. got %s, expected %s", i, newStream[i].HashHex(), hash)
		}
	}
}

func TestNew(t *testing.T) {
	t.Skip("TODO: test new stream creation and decryption")
}
