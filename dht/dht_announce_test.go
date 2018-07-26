package dht

import (
	"testing"
)

func TestDHT_Announce(t *testing.T) {
	t.Skip("NEED SOME TESTS FOR ANNOUNCING")

	// tests
	// - max rate
	// - new announces get ahead of old announces
	// - announcer blocks correctly (when nothing to announce, when next announce time is in the future) and unblocks correctly (when waiting to announce next and a new hash is added)
	// thought: what happens when you're waiting to announce a hash and it gets removed? probably nothing, since later hashes will be announced later. but still good to test this
	//

	//bs, dhts := TestingCreateNetwork(t, 2, true, true)
	//defer func() {
	//	for _, d := range dhts {
	//		go d.Shutdown()
	//	}
	//	bs.Shutdown()
	//	time.Sleep(1 * time.Second)
	//}()
	//
	//announcer := dhts[0]
	//receiver := dhts[1]

}
