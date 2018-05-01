package dht

import "strconv"

func TestingCreateDHT(numNodes int) []*DHT {
	if numNodes < 1 {
		return nil
	}

	ip := "127.0.0.1"
	firstPort := 21000
	dhts := make([]*DHT, numNodes)

	for i := 0; i < numNodes; i++ {
		seeds := []string{}
		if i > 0 {
			seeds = []string{ip + ":" + strconv.Itoa(firstPort)}
		}

		dht, err := New(&Config{Address: ip + ":" + strconv.Itoa(firstPort+i), NodeID: RandomBitmapP().Hex(), SeedNodes: seeds})
		if err != nil {
			panic(err)
		}

		go dht.Start()
		dht.WaitUntilJoined()
		dhts[i] = dht
	}

	return dhts
}
