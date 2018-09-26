package cmd

import (
	sync "github.com/lbryio/lbry.go/ytsync"
	"github.com/lbryio/lbry.go/ytsync/sdk"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	var ytCountCmd = &cobra.Command{
		Use:   "ytcount <youtube_api_key> <youtube_channel_id>",
		Args:  cobra.ExactArgs(2),
		Short: "Count videos in a youtube channel",
		Run:   ytcount,
	}
	RootCmd.AddCommand(ytCountCmd)
}

func ytcount(cmd *cobra.Command, args []string) {
	ytAPIKey := args[0]
	channelID := args[1]

	s := sync.Sync{
		APIConfig: &sdk.APIConfig{
			YoutubeAPIKey: ytAPIKey,
		},
		YoutubeChannelID: channelID,
	}

	count, err := s.CountVideos()
	if err != nil {
		panic(err)
	}

	log.Printf("%d videos in channel %s\n", count, channelID)
}
