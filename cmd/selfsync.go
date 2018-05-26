package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	url2 "net/url"
	"os"

	"os/user"

	"time"

	"strconv"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/null"
	"github.com/lbryio/lbry.go/util"
	sync "github.com/lbryio/lbry.go/ytsync"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var APIURL string
var APIToken string

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
	selfSyncCmd.Flags().BoolVar(&syncUpdate, "update", false, "Update previously synced channels instead of syncing new ones (short for --status synced)")
	selfSyncCmd.Flags().StringVar(&syncStatus, "status", StatusQueued, "Specify which queue to pull from. Overrides --update (Default: queued)")

	RootCmd.AddCommand(selfSyncCmd)
	APIURL = os.Getenv("LBRY_API")
	APIToken = os.Getenv("LBRY_API_TOKEN")
	if APIURL == "" {
		log.Errorln("An API URL was not defined. Please set the environment variable LBRY_API")
		return
	}
	if APIToken == "" {
		log.Errorln("An API Token was not defined. Please set the environment variable LBRY_API_TOKEN")
		return
	}
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

func fetchChannels(status string) ([]APIYoutubeChannel, error) {
	url := APIURL + "/yt/jobs"
	res, _ := http.PostForm(url, url2.Values{
		"auth_token":  {APIToken},
		"sync_status": {status},
		"after":       {strconv.Itoa(int(time.Now().AddDate(0, 0, -1).Unix()))},
	})
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	var response APIJobsResponse
	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, errors.Err(response.Error)
	}
	return response.Data, nil
}

type APISyncUpdateResponse struct {
	Success bool        `json:"success"`
	Error   null.String `json:"error"`
	Data    null.String `json:"data"`
}

func setChannelSyncStatus(channelID string, status string) error {
	host, err := os.Hostname()
	if err != nil {
		return errors.Err("could not detect system hostname")
	}
	url := APIURL + "/yt/sync_update"

	res, _ := http.PostForm(url, url2.Values{
		"channel_id":  {channelID},
		"sync_server": {host},
		"auth_token":  {APIToken},
		"sync_status": {status},
	})
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
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

func spaceCheck() error {
	usr, err := user.Current()
	if err != nil {
		return err
	}
	usedPctile, err := util.GetUsedSpace(usr.HomeDir + "/.lbrynet/blobfiles/")
	if err != nil {
		return err
	}
	if usedPctile > 0.90 && !skipSpaceCheck {
		return errors.Err("more than 90%% of the space has been used. use --skip-space-check to ignore. Used: %.1f%%", usedPctile*100)
	}
	util.SendToSlackInfo("disk usage: %.1f%%", usedPctile*100)
	return nil
}

func selfSync(cmd *cobra.Command, args []string) {
	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		log.Error("A slack token was not present in env vars! Slack messages disabled!")
	} else {
		util.InitSlack(os.Getenv("SLACK_TOKEN"))
	}
	err := spaceCheck()
	if err != nil {
		util.SendToSlackError(err.Error())
		return
	}
	ytAPIKey := args[0]
	//authToken := args[1]

	if !util.InSlice(syncStatus, SyncStatuses) {
		log.Errorf("status must be one of the following: %v\n", SyncStatuses)
		return
	}

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
	syncCount := 0
	if syncStatus == StatusQueued {
	mainLoop:
		for {
			//before processing the queued ones first clear the pending ones (if any)
			//TODO: extract method
			queuesToSync := []string{
				StatusSyncing,
				StatusQueued,
			}
			if syncUpdate {
				queuesToSync = append(queuesToSync, StatusSynced)
			}
			for _, v := range queuesToSync {
				interruptedByUser, err := processQueue(v, ytAPIKey, &syncCount)
				if err != nil {
					util.SendToSlackError(err.Error())
					break mainLoop
				}
				if interruptedByUser {
					util.SendToSlackInfo("interrupted by user!")
					break mainLoop
				}
			}
		}
	} else {
		// sync whatever was specified
		_, err := processQueue(syncStatus, ytAPIKey, &syncCount)
		if err != nil {
			util.SendToSlackError(err.Error())
			return
		}
	}
	util.SendToSlackInfo("Syncing process terminated!")
}

func processQueue(queueStatus string, ytAPIKey string, syncCount *int) (bool, error) {
	util.SendToSlackInfo("Syncing %s channels", queueStatus)
	channelsToSync, err := fetchChannels(queueStatus)
	if err != nil {
		return false, errors.Prefix("failed to fetch channels", err)
	}
	util.SendToSlackInfo("Finished syncing %s channels", queueStatus)
	return syncChannels(channelsToSync, ytAPIKey, syncCount)
}

// syncChannels processes a slice of youtube channels (channelsToSync) and returns a bool that indicates whether
// the execution finished by itself or was interrupted by the user and an error if anything happened
func syncChannels(channelsToSync []APIYoutubeChannel, ytAPIKey string, syncCount *int) (bool, error) {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	for ; *syncCount < len(channelsToSync) && (limit == 0 || *syncCount < limit); *syncCount++ {
		err = spaceCheck()
		if err != nil {
			return false, err
		}
		//avoid dereferencing
		channel := channelsToSync[*syncCount]
		channelID := channel.ChannelId
		lbryChannelName := channel.DesiredChannelName
		if channel.TotalVideos < 1 {
			util.SendToSlackInfo("Channel %s has no videos. Skipping", lbryChannelName)
			continue
		}
		if !channel.SyncServer.IsNull() && channel.SyncServer.String != host {
			util.SendToSlackInfo("Channel %s is being synced by another server: %s", lbryChannelName, channel.SyncServer.String)
			continue
		}

		//acquire the lock on the channel
		err := setChannelSyncStatus(channelID, StatusSyncing)
		if err != nil {
			util.SendToSlackError("Failed acquiring sync rights for channel %s: %v", lbryChannelName, err)
			continue
		}
		util.SendToSlackInfo("Syncing %s to LBRY! (iteration %d)", lbryChannelName, *syncCount)

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
				return s.IsInterrupted(), errors.Prefix("@Nikooo777 this requires manual intervention! Exiting...", err)
			}
			//mark video as failed
			err := setChannelSyncStatus(channelID, StatusFailed)
			if err != nil {
				msg := fmt.Sprintf("Failed setting failed state for channel %s. \n@Nikooo777 this requires manual intervention! Exiting...", lbryChannelName)
				return s.IsInterrupted(), errors.Prefix(msg, err)
			}
			continue
		}
		if s.IsInterrupted() {
			return true, nil
		}
		//mark video as synced
		err = setChannelSyncStatus(channelID, StatusSynced)
		if err != nil {
			msg := fmt.Sprintf("Failed setting failed state for channel %s. \n@Nikooo777 this requires manual intervention! Exiting...", lbryChannelName)
			return false, errors.Prefix(msg, err)
		}
	}
	return false, nil
}
