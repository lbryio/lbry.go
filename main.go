package main

import (
	"github.com/lbryio/lbry.go/cmd"

	log "github.com/sirupsen/logrus"
)

var Version string

func main() {
	log.SetLevel(log.DebugLevel)
	cmd.Execute()
}
