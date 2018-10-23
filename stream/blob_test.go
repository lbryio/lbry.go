package stream

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
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

func Test_pkcs7Pad(t *testing.T) {
	blockLen := 16
	tests := map[string]struct {
		data     []byte
		expected []byte
	}{
		"empty": {
			data:     []byte{},
			expected: []byte{16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16},
		},
		"one": {
			data:     []byte{0},
			expected: []byte{0, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15},
		},
		"seven": {
			data:     []byte{1, 2, 3, 4, 5, 6, 7},
			expected: []byte{1, 2, 3, 4, 5, 6, 7, 9, 9, 9, 9, 9, 9, 9, 9, 9},
		},
		"fifteen": {
			data:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		"sixteen": {
			data:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16},
		},
		"twenty": {
			data:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12},
		},
	}

	for name, tt := range tests {
		actual, err := pkcs7Pad(tt.data, blockLen)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}

		if !bytes.Equal(actual, tt.expected) {
			t.Errorf("%s: got %s, expected %s", name, hex.EncodeToString(actual), hex.EncodeToString(tt.expected))
		}

		unpadded, err := pkcs7Unpad(actual, blockLen)
		if err != nil {
			t.Errorf("%s: unpad: %v", name, err)
			continue
		}

		if !bytes.Equal(unpadded, tt.data) {
			t.Errorf("%s: unpad: got %s, expected %s", name, hex.EncodeToString(unpadded), hex.EncodeToString(tt.data))
		}
	}
}

func testdata(t *testing.T, filename string) []byte {
	data, err := ioutil.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestBlob_Encrypt(t *testing.T) {
	tests := map[string]struct {
		key, iv, data, ciphertext string
		err                       bool
	}{
		"no-data": {
			key:        "efad181bb91c18e93a57178559a42f21",
			iv:         "032cb97fa5292b3109a67239f7c626aa",
			data:       "",
			ciphertext: "33adfa34104bf90f0cc1d033104c0cdb",
			err:        true,
		},
		"short-success": {
			key:        "efad181bb91c18e93a57178559a42f21",
			iv:         "032cb97fa5292b3109a67239f7c626aa",
			data:       "abcdefg",
			ciphertext: "33adfa34104bf90f0cc1d033104c0cdb",
		},
		"1024-bytes": {
			key:        "efad181bb91c18e93a57178559a42f21",
			iv:         "032cb97fa5292b3109a67239f7c626aa",
			data:       "xbmmgjjqqzxbwolnawkhcgatpalewqjatfldazvofaiutyxbtooizfprzibwogcsgeisperxqoarovpobbsqcdizvsyhlvbzpoainpvytdgifecvbylratedcntobaksbpebhgzjwgdaayaluuhiormlfyoybxmepzimggvumeaokbppuoylwulczfcbwubmjfdgezrnjashclcswaxgxvyvcguegbqaoudjhnpfbrnezuhdiqawgmgwbitnhvhwieumyebcyhbanwvkyxwcwhrfehtovofygwewawfjfvndmqrtytpgsspwofpocwjqthofdaguyuvdzcsmzdhhfzzucalypvsyvjrrmbqpoyvbgcfkqlqvwqtjluqwbgunuyetogevyrbaxxtggmjydpqjlqgbrasqvclrvicowpnmsrkexbopepyuhopwtpmqihaggynihpikbypcbvsjogcpwpxkjsnruowgryphrwbovbmdnjfeuvrjrpdlqvacmqvzylpincrbhdtqtyuzqvnbnrnxwtmkqanwnmquonmdsqqfzllvoucqjlpzburgciikotssciipllkrkyxzofstrnnhhvfjtjdibzmzvzrcqhkrfwabwzwrzbmwqddadvveiazooeryjstfimlolypkpsflcfalcnceowxrchfawbxsegqnycgqgakggddgqazfppshlygbtsptnbwlwbnyybfoqbuajojhwthmuyrcikbpohdyjqynbbdeegqabwocjqpwsxsifothimhheeoukdkymcvtggdrcywukfvxqjzaafzhdqeewbietsfecshenoowntxhaoaolfksllkrlpatuofiwiohvrsqawodfgelgtsnyzgnxontcqiluanwywiivhzorbkljqcostysmwdgutyexaaqjimbqqejsmgktbwdskjtikrbkjzbakcvwiokeshgjmtewxtvhroeygpbbrbxzvmbyibbwtoqzqzkkgcuggbkhofuecdz",
			ciphertext: "ff808d89979f9057196f9f74008d3561f310e5b8aaf771849d862ee7dad6a14f251c02d3f1d48b9554b706991d0a025e99092c4442e81970119c44dbf3e45aecd51062231336578ef37a732341f1b1b503ee855feeecf531d633075df9df9e6fca9428e2a181f854082f934c6f8523bc7167b3c2524a5d37cc4896da9bca1a02197760e3e407176ce74299db8e2969e433cdeb042a9773defde7ac87b4cd28cb53b6e41a53b6f160e5c24d1706b5917c0dba22c846e922a9054572ae9f190ca796cfd70ad5624e7dda5f4dce024a30a692c16779a38d6967d6d769893ffda4a832d8475449f7aa8de5f5c421b22e609433823f3c3036220a0ee4e1e5590d94cf86864521208ba8f2228f72527a69e7a5146745b1dacf93db720962ecb9c9c008e73f4ff8e31eff99dcef03bbc2c3b6e9c3f8e71a1df6efbf50c79f0fd66aa4a38ac31550d9e07887cb486229f6b9ac7b5f2d41ffd73c24dbbcb7642f49697756621dce838da68f4a0a0037b478b6404afac0f318f2056fee05ea964f10e5f4ce772434cd739b044bd51c58ceb174346cb73eebfc0d6bf14a0d0bc0dbcdd7b242981fb90bb3f93ef4f51a394eefe9638eb75844235c84297e02458fa37cedb5004f765cf5ad65951c210d7a4228e87e24c630482eae9670df5a0e4e1042ef2f909ac63eb41551e667ba994a1d36b85353b79e2919fdaa345e01641614fe424fee0c211ff698b8725e462d8f7ea590fdfb293600d2c526e634c0ad9bfe80d0c4845781ce635b1dce836dcc68bf1a9efbd6396d241a6c055368d1c8178be47af0617c32054ccb7dd1f52edada4e61484b6aa89916d44d7e3e67a563fee06120844d40a1359ee5cb1d54e4a94ed945acb84e006c4261d6831fd53c6ec802c67363435b60232dae8e262aa07693f8ec34d45894fbfa2d0be4a175574a8f633b8eb3063e6e01a563f4178f564d206e46d07e5a8a4ae8354d47b2d3355aa65cc43b3c748766c44147e3552000da76cbd185cd33c0663991ba6624aa250465d8755f8274b66abd6ebcc3005029e375d4e9a2703fdb8cdbe8bc3e70a52df3ad43c61be1993071ce15fa0340a0a901d106bbb4015d8effb89c814a311e3062804c6e6bc6d390869ed995d161ef67a6d4a1819f2aaa4903d80bbca29c2cb32dc8e1fa2330659b05186514dc65cda4b278146f689f9eb874e844537fdb3d110f4b2787934e4964500180c9682d510ffaf5bbf0c74791acedc26832ef9f4b34edcdc843efdf54c874ff119c327f49f5ac0c90b3ed362a3b34fbd79b17656a9dcf48273fd455c421c4b107cf667bf7f6ddf6c2284b7c62494dea6f5e4d10cd78afeafcebdef91211fd7c1ec2eb0801311fc92e04eda0400b7163d51e397281488827c5e4d3314eaab3ed1f4afcbb375e7567e61dc9c47f899b25c9ef4df0558828b36113e275a16d0a76fa9e8021571c661f36e7b7a009",
		},
		"2mb-Xes": {
			key:        "efad181bb91c18e93a57178559a42f21",
			iv:         "032cb97fa5292b3109a67239f7c626aa",
			data:       strings.Repeat("x", 2*1024*1024-1),
			ciphertext: strings.TrimSpace(string(testdata(t, "encoded-2mb-Xes-minus-one"))),
		},
	}

	for testName, tt := range tests {
		key := unhex(t, tt.key)
		iv := unhex(t, tt.iv)
		blob, err := NewBlob([]byte(tt.data), key, iv)
		if err != nil {
			if !tt.err {
				t.Errorf("%s: %v", testName, err)
			}
		} else if tt.err {
			t.Errorf("%s: expected an error but didn't get one", testName)
		} else {
			expected := unhex(t, tt.ciphertext)
			if len(blob) != len(expected) {
				t.Errorf("%s: length mismatch. got %d, expected %d", testName, len(blob), len(expected))
			}
			if !bytes.Equal(blob, expected) {
				t.Errorf("%s: got %s, expected %s (len is %d)", testName,
					hex.EncodeToString(blob)[4194270:],
					tt.ciphertext[4194270:],
					len(tt.ciphertext),
				)
			}
		}
	}
}

func TestBlob_Plaintext(t *testing.T) {
	expected := unhex(t, "2d218ab43c66741d74211076c069f464811de7fe5767009faaa9982171cc57ef")
	key := unhex(t, "b450f70bd285726e470428df6c6ff8d2")
	iv := unhex(t, "0553e3eb17916333d3468286a30738f1")
	blob := Blob(testdata(t, "a2f1841bb9c5f3b583ac3b8c07ee1a5bf9cc48923721c30d5ca6318615776c284e8936d72fa4db7fdda2e4e9598b1e6c"))
	plaintext, err := blob.Plaintext(key, iv)
	if err != nil {
		t.Fatal(err)
	}
	actual := sha256.Sum256(plaintext)
	if !bytes.Equal(actual[:], expected) {
		t.Errorf("hash mismatch. got %s, expected %s", hex.EncodeToString(actual[:]), hex.EncodeToString(expected))
	}
}

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

func unhex(t *testing.T, s string) []byte {
	r, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
