package cmd

import (
	"fmt"
	"testing"
)

func TestFetchChannels(t *testing.T) {
	res, err := fetchChannels("620280")
	if err != nil {
		t.Error(err)
	}
	if res == nil {
		t.Error("empty response")
	}
	fmt.Println(res)
}

// warning this test will actually set sync_server on the db entry for this test channel (mine)
// such field should be reset to null if the test must be run on a different machine (different hostname)
// and obviously the auth token must be appropriate
func TestSetChannelSyncStatus(t *testing.T) {
	err := setChannelSyncStatus("620280", "UCNQfQvFMPnInwsU_iGYArJQ", StatusSyncing)
	if err != nil {
		t.Error(err)
	}
	err = setChannelSyncStatus("620280", "UCNQfQvFMPnInwsU_iGYArJQ", StatusQueued)
	if err != nil {
		t.Error(err)
	}
}
