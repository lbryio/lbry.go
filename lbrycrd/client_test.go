package lbrycrd_test

import (
	"testing"

	"github.com/lbryio/lbry.go/v3/lbrycrd"
)

func TestNew(t *testing.T) {
	_, err := lbrycrd.New("rpc://xxxx:yyzy@localhost:9245", "")
	if err == nil {
		t.Errorf("wtf")
	}
}
