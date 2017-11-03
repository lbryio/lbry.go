package cmd

import (
	sync "github.com/lbryio/lbry.go/ytsync"

	"github.com/go-errors/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	var ytSyncCmd = &cobra.Command{
		Use:   "ytsync <youtube_api_key> <youtube_channel_id> [<lbry_channel_name>]",
		Args:  cobra.RangeArgs(2, 3),
		Short: "Publish youtube channel into LBRY network.",
		Run:   ytsync,
	}
	ytSyncCmd.Flags().BoolVar(&stopOnError, "stop-on-error", false, "If a publish fails, stop all publishing and exit")
	ytSyncCmd.Flags().IntVar(&maxTries, "max-tries", defaultMaxTries, "Number of times to try a publish that fails")
	RootCmd.AddCommand(ytSyncCmd)
}

const defaultMaxTries = 1

var (
	stopOnError bool
	maxTries    int
)

func ytsync(cmd *cobra.Command, args []string) {
	ytAPIKey := args[0]
	channelID := args[1]
	lbryChannelName := ""
	if len(args) > 2 {
		lbryChannelName = args[2]
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
		YoutubeAPIKey:    ytAPIKey,
		YoutubeChannelID: channelID,
		LbryChannelName:  lbryChannelName,
		StopOnError:      stopOnError,
		MaxTries:         maxTries,
		ConcurrentVideos: 1,
	}

	err := s.FullCycle()
	if err != nil {
		if wrappedError, ok := err.(*errors.Error); ok {
			log.Error(wrappedError.Error() + "\n" + string(wrappedError.Stack()))
		} else {
			panic(err)
		}
	}
}
