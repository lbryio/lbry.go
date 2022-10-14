package lbrycrd

import "testing"

func TestDecodeAddress(t *testing.T) {
	addr := "bMUxfQVUeDi7ActVeZJZHzHKBceai7kHha"
	btcAddr, err := DecodeAddress(addr, &MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if btcAddr.EncodeAddress() != addr {
		t.Errorf("expected: %s, actual: %s", addr, btcAddr.EncodeAddress())
	}
}
