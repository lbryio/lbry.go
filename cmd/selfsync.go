package cmd

import (
	"github.com/lbryio/lbry.go/errors"
	sync "github.com/lbryio/lbry.go/ytsync"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/lbryio/lbry.go/null"
	"github.com/lbryio/lbry.go/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	var selfSyncCmd = &cobra.Command{
		Use:   "selfsync <youtube_api_key> <auth_token>",
		Args:  cobra.RangeArgs(2, 2),
		Short: "Publish youtube channels into LBRY network automatically.",
		Run:   selfSync,
	}
	selfSyncCmd.Flags().BoolVar(&stopOnError, "stop-on-error", false, "If a publish fails, stop all publishing and exit")
	selfSyncCmd.Flags().IntVar(&maxTries, "max-tries", defaultMaxTries, "Number of times to try a publish that fails")
	selfSyncCmd.Flags().BoolVar(&takeOverExistingChannel, "takeover-existing-channel", false, "If channel exists and we don't own it, take over the channel")
	selfSyncCmd.Flags().IntVar(&limit, "limit", 0, "limit the amount of channels to sync")
	RootCmd.AddCommand(selfSyncCmd)
}

type APIJobsResponse struct {
	Success bool                `json:"success"`
	Error   null.String         `json:"error"`
	Data    []APIYoutubeChannel `json:"data"`
}

type APIYoutubeChannel struct {
	ChannelId          string      `json:"channel_id"`
	TotalVideos        uint        `json:"total_videos"`
	DesiredChannelName string      `json:"desired_channel_name"`
	SyncServer         null.String `json:"sync_server"`
}

//PoC
func fetchChannels(authToken string) ([]APIYoutubeChannel, error) {
	url := "http://localhost:8080/yt/jobs"
	payload := strings.NewReader("------WebKitFormBoundary7MA4YWxkTrZu0gW\r\nContent-Disposition: form-data; name=\"auth_token\"\r\n\r\n" + authToken + "\r\n------WebKitFormBoundary7MA4YWxkTrZu0gW--")
	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("content-type", "multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW")
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	//fmt.Println(res)
	//fmt.Println(string(body))
	var response APIJobsResponse
	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

type APISyncUpdateResponse struct {
	Success bool        `json:"success"`
	Error   null.String `json:"error"`
	Data    null.String `json:"data"`
}

func setChannelSyncStatus(authToken string, channelID string, status string) error {
	host, err := os.Hostname()
	if err != nil {
		return errors.Err("could not detect system hostname")
	}
	url := "http://localhost:8080/yt/sync_update"
	payload := strings.NewReader("------WebKitFormBoundary7MA4YWxkTrZu0gW\r\nContent-Disposition: form-data;" +
		" name=\"channel_id\"\r\n\r\n" + channelID + "\r\n------WebKitFormBoundary7MA4YWxkTrZu0gW\r\n" +
		"Content-Disposition: form-data; name=\"sync_server\"\r\n\r\n" + host + "\r\n------WebKitFormBoundary7MA4YWxkTrZu0gW\r\n" +
		"Content-Disposition: form-data; name=\"auth_token\"\r\n\r\n" + authToken + "\r\n------WebKitFormBoundary7MA4YWxkTrZu0gW\r\n" +
		"Content-Disposition: form-data; name=\"sync_status\"\r\n\r\n" + status + "\r\n------WebKitFormBoundary7MA4YWxkTrZu0gW--")
	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("content-type", "multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW")
	req.Header.Add("Cache-Control", "no-cache")
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	//fmt.Println(res)
	//fmt.Println(string(body))
	var response APISyncUpdateResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if !response.Error.IsNull() {
		return errors.Err(response.Error.String)
	}
	if !response.Data.IsNull() && response.Data.String == "ok" {
		return nil
	}
	return errors.Err("invalid API response")
}

func selfSync(cmd *cobra.Command, args []string) {
	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		log.Error("A slack token was not present in env vars! Slack messages disabled!")
	} else {
		util.InitSlack(os.Getenv("SLACK_TOKEN"))
	}

	ytAPIKey := args[0]
	authToken := args[1]

	if stopOnError && maxTries != defaultMaxTries {
		log.Errorln("--stop-on-error and --max-tries are mutually exclusive")
		return
	}
	if maxTries < 1 {
		log.Errorln("setting --max-tries less than 1 doesn't make sense")
		return
	}

	if limit < 0 {
		log.Errorln("setting --limit less than 0 (unlimited) doesn't make sense")
		return
	}
	channelsToSync, err := fetchChannels(authToken)
	if err != nil {
		msg := fmt.Sprintf("failed to fetch channels: %v", err)
		log.Errorln(msg)
		util.SendToSlack(msg)
		return
	}

	for loops := 0; loops < len(channelsToSync); loops++ {
		//avoid dereferencing
		channel := channelsToSync[loops]
		channelID := channel.ChannelId
		lbryChannelName := channel.DesiredChannelName
		if channel.TotalVideos < 1 {
			msg := fmt.Sprintf("Channnel %s has no videos. Skipping", lbryChannelName)
			util.SendToSlack(msg)
			log.Debugln(msg)
			continue
		}
		if !channel.SyncServer.IsNull() {
			msg := fmt.Sprintf("Channnel %s is being synced by another server: %s", lbryChannelName, channel.SyncServer.String)
			util.SendToSlack(msg)
			log.Debugln(msg)
			continue
		}

		//acquire the lock on the channel
		err := setChannelSyncStatus(authToken, channelID, StatusSyncing)
		if err != nil {
			msg := fmt.Sprintf("Failed aquiring sync rights for channel %s: %v", lbryChannelName, err)
			util.SendToSlack(msg)
			log.Error(msg)
			continue
		}
		msg := fmt.Sprintf("Syncing %s to LBRY! (iteration %d)", lbryChannelName, loops)
		util.SendToSlack(msg)
		log.Debugln(msg)

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

		err = s.FullCycle()
		util.SendToSlack("Syncing " + lbryChannelName + " reached an end.")
		if err != nil {
			log.Error(errors.FullTrace(err))
			util.SendToSlack(errors.FullTrace(err))
			//mark video as failed
			err := setChannelSyncStatus(authToken, channelID, StatusFailed)
			if err != nil {
				msg := fmt.Sprintf("Failed setting failed state for channel %s: %v", lbryChannelName, err)
				util.SendToSlack(msg)
				util.SendToSlack("@Nikooo777 this requires manual intervention! Panicing...")
				log.Error(msg)
				panic(msg)
			}
			break
		}
		//mark video as synced
		err = setChannelSyncStatus(authToken, channelID, StatusSynced)
		if err != nil {
			msg := fmt.Sprintf("Failed setting synced state for channel %s: %v", lbryChannelName, err)
			util.SendToSlack(msg)
			util.SendToSlack("@Nikooo777 this requires manual intervention! Panicing...")
			log.Error(msg)
			//this error is very bad. it requires manual intervention
			panic(msg)
			continue
		}

		if limit != 0 && loops >= limit {
			msg := fmt.Sprintf("limit of %d reached! Stopping", limit)
			util.SendToSlack(msg)
			log.Debugln(msg)
			break
		}
	}
	util.SendToSlack("Syncing process terminated!")
	log.Debugln("Syncing process terminated!")
}
