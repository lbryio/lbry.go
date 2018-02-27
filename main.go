package main

import (
	"math/rand"
	"time"

	"github.com/lbryio/lbry.go/cmd"

	log "github.com/sirupsen/logrus"
)

var Version string

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetLevel(log.DebugLevel)
	cmd.Execute()
}
