package address

import "testing"


func TestDecodeAddress(t *testing.T) {
	addr := "bUc9gyCJPKu2CBYpTvJ98MdmsLb68utjP6"
	correct := [25]byte{85, 174, 41, 64, 245, 110, 91, 239, 43, 208, 32, 73, 115, 20, 70, 204, 83, 199, 3,
		206, 210, 176, 194, 188, 193}
	result, err := DecodeAddress(addr)
	if err != nil {
		t.Error(err)
	}
	if result != correct {
		t.Error("Mismatch")
	}
}

func TestEncodeAddress(t *testing.T) {
	addr := [25]byte{85, 174, 41, 64, 245, 110, 91, 239, 43, 208, 32, 73, 115, 20, 70, 204, 83, 199, 3,
		206, 210, 176, 194, 188, 193}
	correct := "bUc9gyCJPKu2CBYpTvJ98MdmsLb68utjP6"
	result, err := EncodeAddress(addr)
	if err != nil {
		t.Error(err)
	}
	if result != correct {
		t.Error("Mismatch")
	}
}
