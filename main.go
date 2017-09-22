package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)

	if len(os.Args) < 2 {
		log.Errorln("Usage: " + os.Args[0] + " COMMAND [options]")
	}

	switch os.Args[1] {
	case "ytsync":
		ytsync()
	case "franklin":
		franklin()
	default:
		log.Errorln("Unknown command: " + os.Args[1])
	}
}
