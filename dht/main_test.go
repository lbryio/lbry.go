package dht

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestDHT(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	port := 49449 // + (rand.Int() % 10)

	config := NewStandardConfig()
	config.Address = "127.0.0.1:" + strconv.Itoa(port)
	config.PrimeNodes = []string{
		"127.0.0.1:10001",
	}

	d := New(config)
	t.Log("Starting...")
	go d.Run()

	time.Sleep(2 * time.Second)

	for {
		peers, err := d.FindNode("012b66fc7052d9a0c8cb563b8ede7662003ba65f425c2661b5c6919d445deeb31469be8b842d6faeea3f2b3ebcaec845")
		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}

		t.Log("Found peers:", peers)
		break
	}

	t.Error("failed")
}
