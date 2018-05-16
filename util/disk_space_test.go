package util

import (
	"os/user"
	"testing"
)

func TestGetUsedSpace(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Error(err)
	}
	usedPctile, err := GetUsedSpace(usr.HomeDir + "/.lbrynet/blobfiles/")
	if err != nil {
		t.Error(err)
	}
	if usedPctile > 1 {
		t.Errorf("over 1: %.2f", usedPctile)
	}
	t.Logf("used space: %.2f", usedPctile)
}
