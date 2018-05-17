package cmd

import (
	"os"
	"os/user"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/util"
	sync "github.com/lbryio/lbry.go/ytsync"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	ytSyncCmd.Flags().BoolVar(&skipSpaceCheck, "skip-space-check", false, "Do not perform free space check on startup")
	RootCmd.AddCommand(ytSyncCmd)
}

func ytsync(cmd *cobra.Command, args []string) {
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
	util.SendToSlackInfo("Syncing " + lbryChannelName + " to LBRY!")

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

	if err != nil {
		util.SendToSlackError(errors.FullTrace(err))

	}
	util.SendToSlackInfo("Syncing " + lbryChannelName + " reached an end.")
}
