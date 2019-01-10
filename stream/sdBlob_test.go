package stream

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestSdBlob_NullBlobHash(t *testing.T) {
	expected, _ := hex.DecodeString("cda3ab14d0de147d2380637c644afbadff072b12353d1acaf79af718d39bff501e47c47e926a2680d477f5fbb9f3b5ce")
	b := BlobInfo{IV: NullIV()}
	if !bytes.Equal(expected, b.Hash()) {
		t.Errorf("null blob has wrong hash. expected %s, got %s", hex.EncodeToString(expected), hex.EncodeToString(b.Hash()))
	}
}

func TestSdBlob_Hash(t *testing.T) {
	expected, _ := hex.DecodeString("2c8cb2893668ef3ad30bda5b3361c0736d746d82fb16155d1510c4d2c5e4481d49ee747f155b2f1156849d422f13a7be")
	b := BlobInfo{
		BlobHash: unhex(t, "f774e1adb8b1fc18b037015844c2469e2166006fcd739e1befca81ccd3df537dfe61041904187e1b88e7636c1848baca"),
		BlobNum:  24,
		IV:       unhex(t, "30303030303030303030303030303235"),
		Length:   1761808,
	}
	if !bytes.Equal(expected, b.Hash()) {
		t.Errorf("blob has wrong hash. expected %s, got %s", hex.EncodeToString(expected), hex.EncodeToString(b.Hash()))
	}
}

func TestSdBlob_NullStreamHash(t *testing.T) {
	// {"stream_name": "", "blobs": [{"length": 0, "blob_num": 0, "iv": "00000000000000000000000000000000"}], "stream_type": "lbryfile", "key": "00000000000000000000000000000000", "suggested_file_name": "", "stream_hash": "4d9a9ce3d72af9f171c4233738e08440937cf906eb506a5d573c0e5500c58500b0a6cbaedc9be2c863750859c01d9954"
	expected, _ := hex.DecodeString("4d9a9ce3d72af9f171c4233738e08440937cf906eb506a5d573c0e5500c58500b0a6cbaedc9be2c863750859c01d9954")
	b := SDBlob{Key: NullIV()}
	b.addBlob(Blob{}, NullIV())
	b.updateStreamHash()
	if !bytes.Equal(b.StreamHash, expected) {
		//fmt.Println(string(b.ToBlob()))
		t.Errorf("null stream has wrong hash. expected %s, got %s", hex.EncodeToString(expected), hex.EncodeToString(b.StreamHash))
	}
}

func TestSdBlob_UnmarshalJSON(t *testing.T) {
	rawBlob := `{"stream_name": "746573745f66696c65", "blobs": [{"length": 2097152, "blob_num": 0, "blob_hash": "e6063cf9656e3ff24a197c5abdc2e5832d166de3b045d789b3f61526f1e82ff64e863a96dced804078dccc65bda6f7b8", "iv": "30303030303030303030303030303031"}, {"length": 2097152, "blob_num": 1, "blob_hash": "c88035438670cfe41c21ebde3cde9641d5b9ec886532b99fee294387174f0399060cb9e0d952dd604549a9e18b89c9e9", "iv": "30303030303030303030303030303032"}, {"length": 2097152, "blob_num": 2, "blob_hash": "95e36178c995510308427942312a2d936ad0c4e307a4df77b654202f62d55178dfea33effd49c9b38fb8531f660147a7", "iv": "30303030303030303030303030303033"}, {"length": 2097152, "blob_num": 3, "blob_hash": "31d7cad3fccc6d0dff8a83fa32fac85a0ff7840461a6d11acbd5eb267cde69065e00949722d86499c3daa1df9f731344", "iv": "30303030303030303030303030303034"}, {"length": 2097152, "blob_num": 4, "blob_hash": "c0369883cb4e97b159c158e37350fad13a4b958fccceb1a2be1b6e0de4afd7f5e3330937113f23edff99c80db40ac8fa", "iv": "30303030303030303030303030303035"}, {"length": 2097152, "blob_num": 5, "blob_hash": "a493e912809640fd4840d8104fddf73cce176be0ee25e5132137dad1491f5789b32c0abff52a5eba53bca5c788bd783d", "iv": "30303030303030303030303030303036"}, {"length": 2097152, "blob_num": 6, "blob_hash": "ca8cd2d526a21c6e7a8197c9d389be8f4b5760d6353ae503995b4ccc67203e551c08f896fc07744abdb9905e02ae883d", "iv": "30303030303030303030303030303037"}, {"length": 2097152, "blob_num": 7, "blob_hash": "3f259aa9b9233c7a53554f603cac2a6564ec88f26de3a5ab07e9abec90d94402e65a2e6cf03d7160cfc8eea2261be7e3", "iv": "30303030303030303030303030303038"}, {"length": 2097152, "blob_num": 8, "blob_hash": "89e551949ee9dc6e64d2d7cd24a8f86fab46432c35ad47478970b0b7beacd8f8e74d8868708309d20285492489829a49", "iv": "30303030303030303030303030303039"}, {"length": 2097152, "blob_num": 9, "blob_hash": "19e24ec6fa0a44e3dcbcb8ed00c78a4a6f991d5f200bccfb8247b7986b6432a9b9f97483421ab08224e568c658544c04", "iv": "30303030303030303030303030303130"}, {"length": 2097152, "blob_num": 10, "blob_hash": "2cf6faaf28058963f062530f3282e883a2f10892574bb78ab7ea059c2f90a676d6a85b83935b87c5e9a1990725207fd1", "iv": "30303030303030303030303030303131"}, {"length": 2097152, "blob_num": 11, "blob_hash": "a7778b9ea7485cf00dd2095c68ceb539d30fc25657995b74e82b3e7f1272d7cac9ce3c609b25181f7a29fb9542392dd9", "iv": "30303030303030303030303030303132"}, {"length": 2097152, "blob_num": 12, "blob_hash": "84bdaa6dc85510d955c5a414445fab05db3062f69c56ca6fa192b7657b118e6335de652602b3a39e750617e1c83c5d24", "iv": "30303030303030303030303030303133"}, {"length": 2097152, "blob_num": 13, "blob_hash": "c48ab21a9726095382561bf9436921d4283d565c06051f6780344eb154b292d494558d3edcd616dda83c09d757969544", "iv": "30303030303030303030303030303134"}, {"length": 2097152, "blob_num": 14, "blob_hash": "e1a669b27068da9de91ad13306b9777776e4cdfa47a6e5085dd5fa317ba716147621dda3bbab0e0fd2a6cc2adbb7cfa0", "iv": "30303030303030303030303030303135"}, {"length": 2097152, "blob_num": 15, "blob_hash": "04ce226f8dac8c3e3b9be65e96028da06d80b43b7a95da87ae5c0bd8ab32b56567897f32cb946366ad94c8e15db35a58", "iv": "30303030303030303030303030303136"}, {"length": 2097152, "blob_num": 16, "blob_hash": "829b58efc9a58062efb2657ce3186bf520858dfabda3588a0e89a4685d19a3da41a0ca11fa926ed2b380c392f8e4cfff", "iv": "30303030303030303030303030303137"}, {"length": 2097152, "blob_num": 17, "blob_hash": "8eece30ffd260d35bbc4c60b6156a56eadffd0e640b1311b680787023f0d8c3e8034a387819b3b6be6add96654fd5837", "iv": "30303030303030303030303030303138"}, {"length": 2097152, "blob_num": 18, "blob_hash": "f6fc9812eec25fef22fbde88957ce700ac0dc4975231ef887a42a314cdcd9360e86ba25fab15f4d2c31f9acb45a3e567", "iv": "30303030303030303030303030303139"}, {"length": 2097152, "blob_num": 19, "blob_hash": "d1f7d2759ec752472b7300f80b1e9795cd3f9098715acd593d0e80aae72cacfccdead47ec9137c510b83983b86e19b26", "iv": "30303030303030303030303030303230"}, {"length": 2097152, "blob_num": 20, "blob_hash": "4eb2952f84f500777057fd904c225b15bdafdabd2d5acab51c185f42f12756b7ede81ae4bc0ae48e59456cc548ce04df", "iv": "30303030303030303030303030303231"}, {"length": 2097152, "blob_num": 21, "blob_hash": "9dcd4b81a28a6fe2b01401f0f2bd5e86f75ec7d81efd87b7f371ec7aafba18290f58b2b75e5e26cf82fb02ccfde13714", "iv": "30303030303030303030303030303232"}, {"length": 2097152, "blob_num": 22, "blob_hash": "4e3dfc7044b119e2a747103c5db97b8518dc1fd6e203beda623b146e03db16132734b5e836ac718530bd3f2b0280ec1b", "iv": "30303030303030303030303030303233"}, {"length": 2097152, "blob_num": 23, "blob_hash": "fd3e76976628650e349e8818fe5ddcbb576a5877cba9ea7e6e115beb12026f49536390f47db33f0d6e99671caa478e93", "iv": "30303030303030303030303030303234"}, {"length": 1761808, "blob_num": 24, "blob_hash": "f774e1adb8b1fc18b037015844c2469e2166006fcd739e1befca81ccd3df537dfe61041904187e1b88e7636c1848baca", "iv": "30303030303030303030303030303235"}, {"length": 0, "blob_num": 25, "iv": "30303030303030303030303030303236"}], "stream_type": "lbryfile", "key": "30313233343536373031323334353637", "suggested_file_name": "746573745f66696c65", "stream_hash": "4fcd4064713bf639362248d3ac0c0ee527a93a08ce4991954d6e11b0317e79b6beedb6833e18e7ae8b0f14ddf258e386"}`
	// remove whitespace. safe because all text is hex-encoded. should update python to use canonical json
	// can MAYBE use https://godoc.org/github.com/docker/go/canonical/json#Encoder.Canonical
	rawBlob = strings.Replace(rawBlob, " ", "", -1)

	sdBlob := SDBlob{}
	err := json.Unmarshal([]byte(rawBlob), &sdBlob)
	if err != nil {
		t.Fatal(err)
	}

	if !sdBlob.IsValid() {
		t.Fatalf("decoded blob is not valid. expected stream hash %s, got %s",
			hex.EncodeToString(sdBlob.StreamHash), hex.EncodeToString(sdBlob.computeStreamHash()))
	}

	reEncoded, err := json.Marshal(sdBlob)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(reEncoded, []byte(rawBlob)) {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(rawBlob, string(reEncoded), false)
		fmt.Println(dmp.DiffPrettyText(diffs))
		t.Fatal("re-encoded string is not equal to original string")
	}
}
