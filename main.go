package main

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/lbryio/lbry.go/jsonrpc"

	"github.com/go-errors/errors"
	log "github.com/sirupsen/logrus"
)

const maxPrice = float64(10)
const waitForStart = 5 * time.Second
const waitForEnd = 3 * time.Minute

func main() {
	log.SetLevel(log.DebugLevel)
	flag.Parse()
	name := flag.Arg(0)
	if name == "" {
		log.Errorln("Usage: " + os.Args[0] + " URL")
		return
	}

	conn := jsonrpc.NewClient("")
	log.Println("Starting...")

	err := testUri(conn, name)
	if err != nil {
		panic(err)
	}

	/*
		todo:
		 - check multiple names in parallel
		 - limit how many parallel checks run
		 - within each name, check if its done every 20 seconds or so. no need to wait a fixed amount of time
		   - but set a max limit on how much to wait (based on filesize?)
		 - report aggregate stats to slack
	*/
}

func testUri(conn *jsonrpc.Client, url string) error {
	log.Infoln("Testing " + url)

	price, err := conn.StreamCostEstimate(url, nil)
	if err != nil {
		return err
	}

	if price == nil {
		return errors.New("could not get price of " + url)
	}

	if float64(*price) > maxPrice {
		return errors.New("the price of " + url + " is too damn high")
	}

	startTime := time.Now()
	get, err := conn.Get(url, nil, nil)
	if err != nil {
		return err
	} else if get == nil {
		return errors.New("received no response for 'get' of " + url)
	}

	if get.Completed {
		log.Infoln("cannot test " + url + " because we already have it")
		return nil
	}

	getDuration := time.Since(startTime)

	log.Infoln("'get' for " + url + " took " + getDuration.String())

	log.Infoln("waiting " + waitForStart.String() + " to see if " + url + " starts")

	time.Sleep(waitForStart)

	fileStartedResult, err := conn.FileList(jsonrpc.FileListOptions{Outpoint: &get.Outpoint})
	if err != nil {
		return err
	}

	if fileStartedResult == nil || len(*fileStartedResult) < 1 {
		log.Errorln(url + " failed to start in " + waitForStart.String())
	} else if (*fileStartedResult)[0].Completed {
		log.Errorln(url + " already finished after " + waitForStart.String() + ". boom!")
	} else if (*fileStartedResult)[0].WrittenBytes == 0 {
		log.Errorln(url + " says it started, but has 0 bytes downloaded after " + waitForStart.String())
	} else {
		log.Infoln(url + " started, with " + strconv.FormatUint((*fileStartedResult)[0].WrittenBytes, 10) + " bytes downloaded")
	}

	log.Infoln("waiting " + waitForEnd.String() + " for file to finish")

	time.Sleep(waitForEnd)

	fileFinishedResult, err := conn.FileList(jsonrpc.FileListOptions{Outpoint: &get.Outpoint})
	if err != nil {
		return err
	}

	if fileFinishedResult == nil || len(*fileFinishedResult) < 1 {
		log.Errorln(url + " failed to start at all")
	} else if !(*fileFinishedResult)[0].Completed {
		log.Errorln(url + " says it started, but has not finished after " + waitForEnd.String() + " (" + strconv.FormatUint((*fileFinishedResult)[0].WrittenBytes, 10) + " bytes written)")
	} else {
		log.Infoln(url + " finished, with " + strconv.FormatUint((*fileFinishedResult)[0].WrittenBytes, 10) + " bytes downloaded")
	}

	return nil
}
