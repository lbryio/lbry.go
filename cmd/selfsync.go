package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"os/user"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/null"
	"github.com/lbryio/lbry.go/util"
	sync "github.com/lbryio/lbry.go/ytsync"
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
	selfSyncCmd.Flags().BoolVar(&skipSpaceCheck, "skip-space-check", false, "Do not perform free space check on startup")

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
	url := "https://api.lbry.io/yt/jobs"
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
	url := "https://api.lbry.io/yt/sync_update"
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
	usr, err := user.Current()
	if err != nil {
		util.SendToSlackError(err.Error())
		return
	}
	usedPctile, err := util.GetUsedSpace(usr.HomeDir + "/.lbrynet/blobfiles/")
	if err != nil {
		util.SendToSlackError(err.Error())
		return
	}
	if usedPctile > 0.9 && !skipSpaceCheck {
		util.SendToSlackError("more than 90%% of the space has been used. use --skip-space-check to ignore. Used: %.1f%%", usedPctile*100)
		return
	}
	util.SendToSlackInfo("disk usage: %.1f%%", usedPctile*100)

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
		util.SendToSlackError("failed to fetch channels: %v", err)
		return
	}
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	for loops := 0; loops < len(channelsToSync) && (limit != 0 && loops < limit); loops++ {
		//avoid dereferencing
		channel := channelsToSync[loops]
		channelID := channel.ChannelId
		lbryChannelName := channel.DesiredChannelName
		if channel.TotalVideos < 1 {
			util.SendToSlackInfo("Channnel %s has no videos. Skipping", lbryChannelName)
			continue
		}
		if !channel.SyncServer.IsNull() && channel.SyncServer.String != host {
			util.SendToSlackInfo("Channnel %s is being synced by another server: %s", lbryChannelName, channel.SyncServer.String)
			continue
		}

		//acquire the lock on the channel
		err := setChannelSyncStatus(authToken, channelID, StatusSyncing)
		if err != nil {
			util.SendToSlackError("Failed aquiring sync rights for channel %s: %v", lbryChannelName, err)
			continue
		}
		util.SendToSlackInfo("Syncing %s to LBRY! (iteration %d)", lbryChannelName, loops)

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
		util.SendToSlackInfo("Syncing " + lbryChannelName + " reached an end.")
		if err != nil {
			util.SendToSlackError(errors.FullTrace(err))
			fatalErrors := []string{
				"default_wallet already exists",
				"WALLET HAS NOT BEEN MOVED TO THE WALLET BACKUP DIR",
			}
			if util.InSliceContains(err.Error(), fatalErrors) {
				util.SendToSlackError("@Nikooo777 this requires manual intervention! Exiting...")
				break
			}
			//mark video as failed
			err := setChannelSyncStatus(authToken, channelID, StatusFailed)
			if err != nil {
				msg := fmt.Sprintf("Failed setting failed state for channel %s: %v", lbryChannelName, err)
				util.SendToSlackError(msg)
				util.SendToSlackError("@Nikooo777 this requires manual intervention! Exiting...")
				break
			}
			continue
		}
		//mark video as synced
		err = setChannelSyncStatus(authToken, channelID, StatusSynced)
		if err != nil {
			msg := fmt.Sprintf("Failed setting synced state for channel %s: %v", lbryChannelName, err)
			util.SendToSlackError(msg)
			util.SendToSlackError("@Nikooo777 this requires manual intervention! Exiting...")
			//this error is very bad. it requires manual intervention
			break
		}
	}
	util.SendToSlackInfo("Syncing process terminated!")
}
