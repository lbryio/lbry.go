package cmd

import (
	"os"

	"time"

	"os/user"

	"github.com/lbryio/lbry.go/util"
	sync "github.com/lbryio/lbry.go/ytsync"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const defaultMaxTries = 3

var (
	stopOnError             bool
	maxTries                int
	takeOverExistingChannel bool
	refill                  int
	limit                   int
	skipSpaceCheck          bool
	syncUpdate              bool
	syncStatus              string
	channelID               string
	syncFrom                int64
	syncUntil               int64
	concurrentJobs          int
	videosLimit             int
	maxVideoSize            int
)

func init() {
	var ytSyncCmd = &cobra.Command{
		Use:   "ytsync",
		Args:  cobra.RangeArgs(0, 0),
		Short: "Publish youtube channels into LBRY network automatically.",
		Run:   ytSync,
	}
	ytSyncCmd.Flags().BoolVar(&stopOnError, "stop-on-error", false, "If a publish fails, stop all publishing and exit")
	ytSyncCmd.Flags().IntVar(&maxTries, "max-tries", defaultMaxTries, "Number of times to try a publish that fails")
	ytSyncCmd.Flags().BoolVar(&takeOverExistingChannel, "takeover-existing-channel", false, "If channel exists and we don't own it, take over the channel")
	ytSyncCmd.Flags().IntVar(&limit, "limit", 0, "limit the amount of channels to sync")
	ytSyncCmd.Flags().BoolVar(&skipSpaceCheck, "skip-space-check", false, "Do not perform free space check on startup")
	ytSyncCmd.Flags().BoolVar(&syncUpdate, "update", false, "Update previously synced channels instead of syncing new ones")
	ytSyncCmd.Flags().StringVar(&syncStatus, "status", "", "Specify which queue to pull from. Overrides --update")
	ytSyncCmd.Flags().StringVar(&channelID, "channelID", "", "If specified, only this channel will be synced.")
	ytSyncCmd.Flags().Int64Var(&syncFrom, "after", time.Unix(0, 0).Unix(), "Specify from when to pull jobs [Unix time](Default: 0)")
	ytSyncCmd.Flags().Int64Var(&syncUntil, "before", time.Now().Unix(), "Specify until when to pull jobs [Unix time](Default: current Unix time)")
	ytSyncCmd.Flags().IntVar(&concurrentJobs, "concurrent-jobs", 1, "how many jobs to process concurrently")
	ytSyncCmd.Flags().IntVar(&videosLimit, "videos-limit", 1000, "how many videos to process per channel")
	ytSyncCmd.Flags().IntVar(&maxVideoSize, "max-size", 2048, "Maximum video size to process (in MB)")

	RootCmd.AddCommand(ytSyncCmd)
}

func ytSync(cmd *cobra.Command, args []string) {
	var hostname string
	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		log.Error("A slack token was not present in env vars! Slack messages disabled!")
	} else {
		var err error
		hostname, err = os.Hostname()
		if err != nil {
			log.Error("could not detect system hostname")
			hostname = "ytsync-unknown"
		}
		util.InitSlack(os.Getenv("SLACK_TOKEN"), os.Getenv("SLACK_CHANNEL"), hostname)
	}

	if syncStatus != "" && !util.InSlice(syncStatus, sync.SyncStatuses) {
		log.Errorf("status must be one of the following: %v\n", sync.SyncStatuses)
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

	apiURL := os.Getenv("LBRY_API")
	apiToken := os.Getenv("LBRY_API_TOKEN")
	youtubeAPIKey := os.Getenv("YOUTUBE_API_KEY")
	blobsDir := os.Getenv("BLOBS_DIRECTORY")
	if apiURL == "" {
		log.Errorln("An API URL was not defined. Please set the environment variable LBRY_API")
		return
	}
	if apiToken == "" {
		log.Errorln("An API Token was not defined. Please set the environment variable LBRY_API_TOKEN")
		return
	}
	if youtubeAPIKey == "" {
		log.Errorln("A Youtube API key was not defined. Please set the environment variable YOUTUBE_API_KEY")
		return
	}
	if blobsDir == "" {
		usr, err := user.Current()
		if err != nil {
			log.Errorln(err.Error())
			return
		}
		blobsDir = usr.HomeDir + "/.lbrynet/blobfiles/"
	}

	sm := sync.SyncManager{
		StopOnError:             stopOnError,
		MaxTries:                maxTries,
		TakeOverExistingChannel: takeOverExistingChannel,
		Refill:                  refill,
		Limit:                   limit,
		SkipSpaceCheck:          skipSpaceCheck,
		SyncUpdate:              syncUpdate,
		SyncStatus:              syncStatus,
		SyncFrom:                syncFrom,
		SyncUntil:               syncUntil,
		ConcurrentJobs:          concurrentJobs,
		ConcurrentVideos:        concurrentJobs,
		HostName:                hostname,
		YoutubeChannelID:        channelID,
		YoutubeAPIKey:           youtubeAPIKey,
		ApiURL:                  apiURL,
		ApiToken:                apiToken,
		BlobsDir:                blobsDir,
		VideosLimit:             videosLimit,
		MaxVideoSize:            maxVideoSize,
	}

	err := sm.Start()
	if err != nil {
		sync.SendErrorToSlack(err.Error())
	}
	sync.SendInfoToSlack("Syncing process terminated!")
}
