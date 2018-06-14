package dht

import (
	"encoding/hex"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/lbryio/reflector.go/dht/bits"
	"github.com/lyoshenka/bencode"
)

func TestBencodeDecodeStoreArgs(t *testing.T) {
	blobHash := "3214D6C2F77FCB5E8D5FC07EDAFBA614F031CE8B2EAB49F924F8143F6DFBADE048D918710072FB98AB1B52B58F4E1468"
	lbryID := "7CE1B831DEC8689E44F80F547D2DEA171F6A625E1A4FF6C6165E645F953103DABEB068A622203F859C6C64658FD3AA3B"
	port := hex.EncodeToString([]byte("3333"))
	token := "17C2D8E1E48EF21567FE4AD5C8ED944B798D3B65AB58D0C9122AD6587D1B5FED472EA2CB12284CEFA1C21EFF302322BD"
	nodeID := "7CE1B831DEC8689E44F80F547D2DEA171F6A625E1A4FF6C6165E645F953103DABEB068A622203F859C6C64658FD3AA3B"
	selfStore := hex.EncodeToString([]byte("1"))

	raw := "6C" + // start args list
		"3438 3A " + blobHash + // blob hash
		"64" + // start value dict
		"363A6C6272796964 3438 3A " + lbryID + // lbry id
		"343A706F7274 69 " + port + " 65" + // port
		"353A746F6B656E 3438 3A " + token + // token
		"65" + // end value dict
		"3438 3A " + nodeID + // node id
		"69 " + selfStore + " 65" + // self store (integer)
		"65" // end args list

	raw = strings.ToLower(strings.Replace(raw, " ", "", -1))

	data, err := hex.DecodeString(raw)
	if err != nil {
		t.Error(err)
		return
	}

	storeArgs := &storeArgs{}
	err = bencode.DecodeBytes(data, storeArgs)
	if err != nil {
		t.Error(err)
	}

	if storeArgs.BlobHash.Hex() != strings.ToLower(blobHash) {
		t.Error("blob hash mismatch")
	}
	if storeArgs.Value.LbryID.Hex() != strings.ToLower(lbryID) {
		t.Error("lbryid mismatch")
	}
	if hex.EncodeToString([]byte(strconv.Itoa(storeArgs.Value.Port))) != port {
		t.Error("port mismatch")
	}
	if hex.EncodeToString([]byte(storeArgs.Value.Token)) != strings.ToLower(token) {
		t.Error("token mismatch")
	}
	if storeArgs.NodeID.Hex() != strings.ToLower(nodeID) {
		t.Error("node id mismatch")
	}
	if !storeArgs.SelfStore {
		t.Error("selfStore mismatch")
	}

	reencoded, err := bencode.EncodeBytes(storeArgs)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(reencoded, data) {
		t.Error("reencoded data does not match original")
		spew.Dump(reencoded, data)
	}
}

func TestBencodeFindNodesResponse(t *testing.T) {
	res := Response{
		ID:     newMessageID(),
		NodeID: bits.Rand(),
		Contacts: []Contact{
			{ID: bits.Rand(), IP: net.IPv4(1, 2, 3, 4).To4(), Port: 5678},
			{ID: bits.Rand(), IP: net.IPv4(4, 3, 2, 1).To4(), Port: 8765},
		},
	}

	encoded, err := bencode.EncodeBytes(res)
	if err != nil {
		t.Fatal(err)
	}

	var res2 Response
	err = bencode.DecodeBytes(encoded, &res2)
	if err != nil {
		t.Fatal(err)
	}

	compareResponses(t, res, res2)
}

func TestBencodeFindValueResponse(t *testing.T) {
	res := Response{
		ID:           newMessageID(),
		NodeID:       bits.Rand(),
		FindValueKey: bits.Rand().String(),
		Token:        "arst",
		Contacts: []Contact{
			{ID: bits.Rand(), IP: net.IPv4(1, 2, 3, 4).To4(), Port: 5678},
		},
	}

	encoded, err := bencode.EncodeBytes(res)
	if err != nil {
		t.Fatal(err)
	}

	var res2 Response
	err = bencode.DecodeBytes(encoded, &res2)
	if err != nil {
		t.Fatal(err)
	}

	compareResponses(t, res, res2)
}

func TestDecodeSomeStrings(t *testing.T) {
	strs := []string{
		"6469306569306569316532303a359ec440b8236dee0fb2500ebda9c4704ae6741469326534383ade269943996b4bef3ff41176668a0577f86aba7f1ea2996edd18f9c42430802c8085331345c5f0c44a7f352e2ba8ae59693365383a66696e644e6f64656934656c34383ade269943996b4bef3ff41176668a0577f86aba7f1ea2996edd18f9c42430802c8085331345c5f0c44a7f352e2ba8ae596565",
		"64313a30693165313a3132303a31f27ec6174783c6f00e4b9ebccd49c6bdce300e313a3234383a7b3785cd962a56d4d4574262b1b63cc95d062c825b38f73602c7dc297c1e6b8259cdcdb5e7edef216d514e0c31ad8637313a3364353a746f6b656e34383a071fdf9835995889920e9beb876ebe8e36e5d44f5822684b549457405e22a5bf3fe40335743b459f22c347c0e736eba434383a7bcc7f8804db7d0d40fcb9c508547e53ee4f7d4000758105ecce099fc931d9cccfbbf9a30829777afcf41f2b4b0aeb716c35343a627db2b20d05b109195e7d06a855c392fda9a05faa878e69b18cc1212c35371ba5cad6231e049a460cdc263e7f4701afad45b288f48435343a424434ae0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a627db2b20d05b109195e7d06a855c392fda9a05faa878e69b18cc1212c35371ba5cad6231e049a460cdc263e7f4701afad45b288f48435343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a627db2b20d05b109195e7d06a855c392fda9a05faa878e69b18cc1212c35371ba5cad6231e049a460cdc263e7f4701afad45b288f48435343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a424434ae0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a627db2b20d05b109195e7d06a855c392fda9a05faa878e69b18cc1212c35371ba5cad6231e049a460cdc263e7f4701afad45b288f48435343a424434ae0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a490ed98d0d05722bfafc5574fef0b2897737c91ac9e0ba0c8dadf3f8d51abb9cc46d485cad44a2d8d67e729fcf20a9dafdef2176026935343a490ed98d0d05722bfafc5574fef0b2897737c91ac9e0ba0c8dadf3f8d51abb9cc46d485cad44a2d8d67e729fcf20a9dafdef2176026935343a490ed98d0d05722bfafc5574fef0b2897737c91ac9e0ba0c8dadf3f8d51abb9cc46d485cad44a2d8d67e729fcf20a9dafdef2176026935343a424434ae0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a490ed98d0d05722bfafc5574fef0b2897737c91ac9e0ba0c8dadf3f8d51abb9cc46d485cad44a2d8d67e729fcf20a9dafdef2176026935343a424434ae0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a47413e2b0d05bf6244e5edacf43f65ef4cfd4b6eac380244894cc364ec23e8ededfb085999a211e20f8de146d9e081bbb42dc61d9ca035343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a67d9a7ea0d05223c42ef5979c7c8f2444311f7a8266d64d3f98e9d0b21726bc2714a6fea5426c9a46a68c49ea13b54ccc324adf0783035343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a424434ae0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a490ed98d0d05722bfafc5574fef0b2897737c91ac9e0ba0c8dadf3f8d51abb9cc46d485cad44a2d8d67e729fcf20a9dafdef2176026935343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a490ed98d0d05722bfafc5574fef0b2897737c91ac9e0ba0c8dadf3f8d51abb9cc46d485cad44a2d8d67e729fcf20a9dafdef2176026935343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a490ed98d0d05722bfafc5574fef0b2897737c91ac9e0ba0c8dadf3f8d51abb9cc46d485cad44a2d8d67e729fcf20a9dafdef2176026935343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a68fe5c3e0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343abcbae8200d050ffa2a9336bf260ee376194faed30de40e73108beb4158fb6f78da823f29c97b497c45b40872ead7230736deba92bbaf35343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a68fe5c3d0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343abcbae8200d050ffa2a9336bf260ee376194faed30de40e73108beb4158fb6f78da823f29c97b497c45b40872ead7230736deba92bbaf35343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a68fe5c3d0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e635343a627db2b20d05b109195e7d06a855c392fda9a05faa878e69b18cc1212c35371ba5cad6231e049a460cdc263e7f4701afad45b288f48435343a4462ba020d059ed3d31ceb3d1ffb2c025abf065a14516a8942607f9b3fad4b697ccc232766bdf471b6518a3ed716f2712171fd046c4335343a68fe5c3d0d05bfd6e1b98cc921b441fa8b560ac29038c37ec4907ae023617bf2bbc54abd3403226daf479798edcce69743ea6699e6ff35343a4ace39c80d0541ae66325db4f1fce530476942d339f3219913f0a237726e18746a65ff7b3c5558c82c274a6f1ed10bbaf7c1305ca8e6656565",
		"64313a30693165313a3132303aa19e41f1d887db705fbf0a58f120773069d6ff11313a3234383a6911c21c3b5ea6536f4fb170c87fdc8bf4201124c5fe5eeb5f0054ff48e899a1d6e089a30a12ba8683ebf79691d71439313a3364383a636f6e74616374736c6c34383a6dd3dedeec334bae70a3c5c1b58fef2d4a501af320a332cdd5db4b71945c0a90d11aa2013e8d0216258a643f42d9562631323a34362e32382e3230342e3738693434343465656c34383a7ec0edf204d940d21c17e1c979df1c94610711d7773ceda72a4aa925c5f7799bbb741caa8d6d877aa2776399f77c08a031333a36392e3131382e34332e313234693434343465656c34383a6ec15cd8b95718d010bdf9739fda1af9ded7ac8be7da5cced6dc7a08470c8109accc5bfb973e5c1dd711349c62a8a3c331323a34362e32382e3230342e3738693434343465656c34383a6d2603a690712d5a3d5addb228d31a622f798c64ab2d3576d2c3ab3c5a64cb863c05545f37aa384ce83003416ca5cea431323a34362e32382e3230342e3738693434343465656c34383a6c1d92202a7e74ec85dd4e3e1699303174c4b6460b171637bba0a1c068d72f389035e1ba8bbe70f6621c36d70d4d045031323a34362e32382e3230342e3738693434343465656c34383a6dac30fbe94c006515cffe4898bc534a105fd0403bfc3c8692c2751b7946949dbc9e0c9570477cf8f817d1efc9a0e02531343a31382e3232312e3139342e313933693434343465656c34383a6f2ee379c662ea9f6db86beed6023be300cd23b9f9c4610a195dcb0f258392ec77188854ea3a8c0fffe67844849eb6ca31333a32342e3131362e39302e323238693434343465656c34383a6c55b2dbb89010de9db88e0ad3510a79fe60983b394a9e66b0efe139a455973cdc4415beff82cbdfb63d98caad648ab031333a38382e3132392e3234352e33396934343434656565353a746f6b656e34383a51844b0d1e8a613c4ad783c03b323063d3a5ff063a640368d7754bcae277c22b2b06b46c3e5466c30acb773f77b686476565",
		"64313a30693165313a3132303af693334af099a987a4b55af2727472b51c1990db313a3234383ab6928ff25778a7bbb5d258d3b3a06e26db1654f3d2efce8c26681d43f7237cdf2e359a4d309c4473d5d89ec99fb4f573313a3364353a746f6b656e34383a61a45a40080ea118361f919ebf09cdc9d2c63f476e7bc4d193463f52b91dc4b0db4ed1004e32d4fc273645421e58479034383abe463e24af015c0d1ba496b4b1911aa98c8b42119295bbcc5c36829a2e2ff8f149743cbbc0af2c39669cb0750fd778576c35343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e9535343a239911670d05234aa57f90985a97e6e67a5131a4f8fe7dfccaacb4707734294b8ad774c92eb424671aa533e76c3083da796b4db30e95656565",
		"6469306569316569316532303a3e0929c88abb8fe1d718025efc7f4a3cd85de16269326534383a21b2e2d2996b4bef3ff41176668a0577f86aba7f1ea2996edd18f9c42430802c8085331345c5f0c44a7f352e2ba8ae596933656c6565",
	}

	for i, str := range strs {
		raw, err := hex.DecodeString(str)
		if err != nil {
			t.Errorf("error hex-decoding string %d: %s", i, err)
			continue
		}

		var decoded interface{}
		err = bencode.DecodeBytes(raw, &decoded)
		if err != nil {
			t.Errorf("error bencode-decoding string %d: %s", i, err)
			continue
		}
		//
		//t.Error("TODO")
		//continue
		//
		//spew.Dump(decoded)
	}
}

func TestDecodeFindNodeResponseWithNoNodes(t *testing.T) {
	raw, err := hex.DecodeString("6469306569316569316532303a3e0929c88abb8fe1d718025efc7f4a3cd85de16269326534383a21b2e2d2996b4bef3ff41176668a0577f86aba7f1ea2996edd18f9c42430802c8085331345c5f0c44a7f352e2ba8ae596933656c6565")
	if err != nil {
		t.Fatal(err)
	}

	response := Response{}
	err = bencode.DecodeBytes(raw, &response)
	//spew.Dump(response)
	if err != nil {
		t.Fatal(err)
	}
}

func compareResponses(t *testing.T, res, res2 Response) {
	if res.ID != res2.ID {
		t.Errorf("expected ID %s, got %s", res.ID, res2.ID)
	}
	if !res.NodeID.Equals(res2.NodeID) {
		t.Errorf("expected NodeID %s, got %s", res.NodeID.Hex(), res2.NodeID.Hex())
	}
	if res.Data != res2.Data {
		t.Errorf("expected Data %s, got %s", res.Data, res2.Data)
	}
	if res.FindValueKey != res2.FindValueKey {
		t.Errorf("expected FindValueKey %s, got %s", res.FindValueKey, res2.FindValueKey)
	}
	if res.Token != res2.Token {
		t.Errorf("expected Token %s, got %s", res.Token, res2.Token)
	}
	if !reflect.DeepEqual(res.Contacts, res2.Contacts) {
		t.Errorf("expected FindNodeData %s, got %s", spew.Sdump(res.Contacts), spew.Sdump(res2.Contacts))
	}
}
