package main

import (
	"math/rand"
	"time"

	"github.com/lbryio/lbry.go/jsonrpc"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.Println("Starting...")

	conn := jsonrpc.NewClient("")

	response, err := conn.Get("one", nil, nil)
	if err != nil {
		panic(err)
	}
	spew.Dump(response)

}
