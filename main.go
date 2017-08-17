package main

import (
	"fmt"
	"github.com/lbryio/lbry.go/dht"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	port := 49449 // + (rand.Int() % 10)

	config := dht.NewStandardConfig()
	config.Address = "127.0.0.1:" + strconv.Itoa(port)
	config.PrimeNodes = []string{
		"127.0.0.1:10001",
	}

	d := dht.New(config)
	log.Info("Starting...")
	go d.Run()

	time.Sleep(5 * time.Second)

	for {
		peers, err := d.FindNode("012b66fc7052d9a0c8cb563b8ede7662003ba65f425c2661b5c6919d445deeb31469be8b842d6faeea3f2b3ebcaec845")
		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}

		fmt.Println("Found peers:", peers)
		break
	}

	time.Sleep(1 * time.Hour)
}
