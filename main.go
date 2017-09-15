package main

import (
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)

	//franklin()

	err := ytsync()
	if err != nil {
		panic(err)
	}
}
