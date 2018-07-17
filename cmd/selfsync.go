package cmd

import (
	"os"

	"time"

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
)

func init() {
	var selfSyncCmd = &cobra.Command{
		Use:   "selfsync",
		Args:  cobra.RangeArgs(0, 0),
		Short: "Publish youtube channels into LBRY network automatically.",
		Run:   selfSync,
	}
	selfSyncCmd.Flags().BoolVar(&stopOnError, "stop-on-error", false, "If a publish fails, stop all publishing and exit")
	selfSyncCmd.Flags().IntVar(&maxTries, "max-tries", defaultMaxTries, "Number of times to try a publish that fails")
	selfSyncCmd.Flags().BoolVar(&takeOverExistingChannel, "takeover-existing-channel", false, "If channel exists and we don't own it, take over the channel")
	selfSyncCmd.Flags().IntVar(&limit, "limit", 0, "limit the amount of channels to sync")
	selfSyncCmd.Flags().BoolVar(&skipSpaceCheck, "skip-space-check", false, "Do not perform free space check on startup")
	selfSyncCmd.Flags().BoolVar(&syncUpdate, "update", false, "Update previously synced channels instead of syncing new ones (short for --status synced)")
	selfSyncCmd.Flags().StringVar(&syncStatus, "status", sync.StatusQueued, "Specify which queue to pull from. Overrides --update (Default: queued)")
	selfSyncCmd.Flags().StringVar(&channelID, "channelID", "", "If specified, only this channel will be synced.")
	selfSyncCmd.Flags().Int64Var(&syncFrom, "after", time.Unix(0, 0).Unix(), "Specify from when to pull jobs [Unix time](Default: 0)")
	selfSyncCmd.Flags().Int64Var(&syncUntil, "before", time.Now().Unix(), "Specify until when to pull jobs [Unix time](Default: current Unix time)")
	selfSyncCmd.Flags().IntVar(&concurrentJobs, "concurrent-jobs", 1, "how many jobs to process concurrently (Default: 1)")

	RootCmd.AddCommand(selfSyncCmd)
}

func selfSync(cmd *cobra.Command, args []string) {
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

	if !util.InSlice(syncStatus, sync.SyncStatuses) {
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
	}

	err := sm.Start()
	if err != nil {
		util.SendErrorToSlack(err.Error())
	}
	util.SendInfoToSlack("Syncing process terminated!")
}
