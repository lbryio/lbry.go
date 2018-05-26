package cmd

import (
	"fmt"
	"testing"
)

/*
func TestMain(m *testing.M) {
	APIURL = os.Getenv("LBRY_API")
	APIToken = os.Getenv("LBRY_API_TOKEN")
}
*/
func TestFetchChannels(t *testing.T) {
	res, err := fetchChannels(StatusQueued)
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
	err := setChannelSyncStatus("UCNQfQvFMPnInwsU_iGYArJQ", StatusSyncing)
	if err != nil {
		t.Error(err)
	}
	err = setChannelSyncStatus("UCNQfQvFMPnInwsU_iGYArJQ", StatusQueued)
	if err != nil {
		t.Error(err)
	}
}
