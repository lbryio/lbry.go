package cmd

import (
	"strconv"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/jsonrpc"

	"github.com/go-errors/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	var franklinCmd = &cobra.Command{
		Use:   "franklin",
		Short: "Test availability of homepage content",
		Run: func(cmd *cobra.Command, args []string) {
			franklin()
		},
	}
	RootCmd.AddCommand(franklinCmd)
}

const (
	maxPrice         = float64(999)
	waitForStart     = 5 * time.Second
	waitForEnd       = 60 * time.Minute
	maxParallelTests = 5
)

type Result struct {
	started  bool
	finished bool
}

func franklin() {
	conn := jsonrpc.NewClient("")

	var wg sync.WaitGroup
	queue := make(chan string)

	var mutex sync.Mutex
	results := map[string]Result{}

	for i := 0; i < maxParallelTests; i++ {
		go func() {
			wg.Add(1)
			defer wg.Done()
			for {
				url, more := <-queue
				if !more {
					return
				}

				res, err := doURL(conn, url)
				mutex.Lock()
				results[url] = res
				mutex.Unlock()
				if err != nil {
					log.Errorln(url + ": " + err.Error())
				}
			}
		}()
	}

	urls := []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	for _, url := range urls {
		queue <- url
	}
	close(queue)

	wg.Wait()

	countStarted := 0
	countFinished := 0
	for _, r := range results {
		if r.started {
			countStarted++
		}
		if r.finished {
			countFinished++
		}
	}

	log.Println("Started: " + strconv.Itoa(countStarted) + " of " + strconv.Itoa(len(results)))
	log.Println("Finished: " + strconv.Itoa(countFinished) + " of " + strconv.Itoa(len(results)))
}

func doURL(conn *jsonrpc.Client, url string) (Result, error) {
	log.Infoln(url + ": Starting")

	result := Result{}

	price, err := conn.StreamCostEstimate(url, nil)
	if err != nil {
		return result, err
	}

	if price == nil {
		return result, errors.New("could not get price of " + url)
	}

	if float64(*price) > maxPrice {
		return result, errors.New("the price of " + url + " is too damn high")
	}

	startTime := time.Now()
	get, err := conn.Get(url, nil, nil)
	if err != nil {
		return result, err
	} else if get == nil {
		return result, errors.New("received no response for 'get' of " + url)
	}

	if get.Completed {
		log.Infoln(url + ": cannot test because we already have it")
		return result, nil
	}

	log.Infoln(url + ": get took " + time.Since(startTime).String())

	log.Infoln(url + ": waiting " + waitForStart.String() + " to see if it starts")

	time.Sleep(waitForStart)

	fileStartedResult, err := conn.FileList(jsonrpc.FileListOptions{Outpoint: &get.Outpoint})
	if err != nil {
		return result, err
	}

	if fileStartedResult == nil || len(*fileStartedResult) < 1 {
		log.Errorln(url + ": failed to start in " + waitForStart.String())
	} else if (*fileStartedResult)[0].Completed {
		log.Infoln(url + ": already finished after " + waitForStart.String() + ". boom!")
		result.started = true
		result.finished = true
		return result, nil
	} else if (*fileStartedResult)[0].WrittenBytes == 0 {
		log.Errorln(url + ": says it started, but has 0 bytes downloaded after " + waitForStart.String())
	} else {
		log.Infoln(url + ": started, with " + strconv.FormatUint((*fileStartedResult)[0].WrittenBytes, 10) + " bytes downloaded")
		result.started = true
	}

	log.Infoln(url + ": waiting up to " + waitForEnd.String() + " for file to finish")

	var fileFinishedResult *jsonrpc.FileListResponse
	ticker := time.NewTicker(15 * time.Second)
	// todo: timeout should be based on file size
	timeout := time.After(waitForEnd)

WaitForFinish:
	for {
		select {
		case <-ticker.C:
			fileFinishedResult, err = conn.FileList(jsonrpc.FileListOptions{Outpoint: &get.Outpoint})
			if err != nil {
				return result, err
			}
			if fileFinishedResult != nil && len(*fileFinishedResult) > 0 {
				if (*fileFinishedResult)[0].Completed {
					ticker.Stop()
					break WaitForFinish
				} else {
					log.Infoln(url + ": " + strconv.FormatUint((*fileFinishedResult)[0].WrittenBytes, 10) + " bytes downloaded after " + time.Since(startTime).String())
				}
			}
		case <-timeout:
			ticker.Stop()
			break WaitForFinish
		}
	}

	if fileFinishedResult == nil || len(*fileFinishedResult) < 1 {
		log.Errorln(url + ": failed to start at all")
	} else if !(*fileFinishedResult)[0].Completed {
		log.Errorln(url + ": says it started, but has not finished after " + waitForEnd.String() + " (" + strconv.FormatUint((*fileFinishedResult)[0].WrittenBytes, 10) + " bytes written)")
	} else {
		log.Infoln(url + ": finished after " + time.Since(startTime).String() + " , with " + strconv.FormatUint((*fileFinishedResult)[0].WrittenBytes, 10) + " bytes downloaded")
		result.finished = true
	}

	return result, nil
}
