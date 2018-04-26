package cmd

import (
	"github.com/lbryio/lbry.go/errors"
	sync "github.com/lbryio/lbry.go/ytsync"

	"fmt"
	"github.com/lbryio/lbry.go/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func init() {
	var ytSyncCmd = &cobra.Command{
		Use:   "ytsync <youtube_api_key> <lbry_channel_name> [<youtube_channel_id>]",
		Args:  cobra.RangeArgs(2, 3),
		Short: "Publish youtube channel into LBRY network.",
		Run:   ytsync,
	}
	ytSyncCmd.Flags().BoolVar(&stopOnError, "stop-on-error", false, "If a publish fails, stop all publishing and exit")
	ytSyncCmd.Flags().IntVar(&maxTries, "max-tries", defaultMaxTries, "Number of times to try a publish that fails")
	ytSyncCmd.Flags().BoolVar(&takeOverExistingChannel, "takeover-existing-channel", false, "If channel exists and we don't own it, take over the channel")
	ytSyncCmd.Flags().IntVar(&refill, "refill", 0, "Also add this many credits to the wallet")
	RootCmd.AddCommand(ytSyncCmd)
	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		log.Error("A slack token was not present in env vars! Slack messages disabled!")
	} else {
		util.InitSlack(os.Getenv("SLACK_TOKEN"))
	}
}

const defaultMaxTries = 3

var (
	stopOnError             bool
	maxTries                int
	takeOverExistingChannel bool
	refill                  int
)

//PoC
func fetchChannels() {
	url := "http://localhost:8080/yt/jobs"
	payload := strings.NewReader("------WebKitFormBoundary7MA4YWxkTrZu0gW\r\nContent-Disposition: form-data; name=\"auth_token\"\r\n\r\n620280\r\n------WebKitFormBoundary7MA4YWxkTrZu0gW--")
	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("content-type", "multipart/form-data")
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Println(res)
	fmt.Println(string(body))
}

func ytsync(cmd *cobra.Command, args []string) {
	ytAPIKey := args[0]
	lbryChannelName := args[1]
	if string(lbryChannelName[0]) != "@" {
		log.Errorln("LBRY channel name must start with an @")
		return
	}

	channelID := ""
	if len(args) > 2 {
		channelID = args[2]
	}

	if stopOnError && maxTries != defaultMaxTries {
		log.Errorln("--stop-on-error and --max-tries are mutually exclusive")
		return
	}
	if maxTries < 1 {
		log.Errorln("setting --max-tries less than 1 doesn't make sense")
		return
	}

	s := sync.Sync{
		YoutubeAPIKey:           ytAPIKey,
		YoutubeChannelID:        channelID,
		LbryChannelName:         lbryChannelName,
		StopOnError:             stopOnError,
		MaxTries:                maxTries,
		ConcurrentVideos:        1,
		TakeOverExistingChannel: takeOverExistingChannel,
		Refill:                  refill,
	}

	err := s.FullCycle()

	if err != nil {
		log.Error(errors.FullTrace(err))
	}
}
