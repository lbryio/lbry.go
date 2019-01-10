package stream

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

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

func testdata(t *testing.T, filename string) []byte {
	data, err := ioutil.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func unhex(t *testing.T, s string) []byte {
	r, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
