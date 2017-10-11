package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lbryio/lbry.go/jsonrpc"

	"github.com/garyburd/redigo/redis"
	ytdl "github.com/kkdai/youtube"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
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

const (
	concurrentVideos = 1
	redisHashKey     = "ytsync"
	redisSyncedVal   = "t"
	defaultMaxTries  = 1
)

type video struct {
	id               string
	channelID        string
	channelTitle     string
	title            string
	description      string
	playlistPosition int64
	publishedAt      time.Time
}

func (v video) getFilename() string {
	return videoDirectory + "/" + v.id + ".mp4"
}

// sorting videos
type byPublishedAt []video

func (a byPublishedAt) Len() int           { return len(a) }
func (a byPublishedAt) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPublishedAt) Less(i, j int) bool { return a[i].publishedAt.Before(a[j].publishedAt) }

type byPlaylistPosition []video

func (a byPlaylistPosition) Len() int           { return len(a) }
func (a byPlaylistPosition) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPlaylistPosition) Less(i, j int) bool { return a[i].playlistPosition < a[j].playlistPosition }

var (
	ytAPIKey        string
	channelID       string
	lbryChannelName string
	stopOnError     bool
	maxTries        int

	daemon         *jsonrpc.Client
	claimAddress   string
	videoDirectory string
	redisPool      *redis.Pool
)

func ytsync(cmd *cobra.Command, args []string) {
	var err error

	ytAPIKey = args[0]
	channelID = args[1]
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

	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 5 * time.Minute,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", ":6379") },
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	var wg sync.WaitGroup
	videoQueue := make(chan video)

	stopEnqueuing := make(chan struct{})
	sendStopEnqueuing := sync.Once{}

	var videoErrored atomic.Value
	videoErrored.Store(false)
	if stopOnError {
		log.Println("Will stop publishing if an error is detected")
	}

	daemon = jsonrpc.NewClient("")
	videoDirectory, err = ioutil.TempDir("", "ytsync")
	if err != nil {
		panic(err)
	}

	if lbryChannelName != "" {
		err = ensureChannelOwnership()
		if err != nil {
			panic(err)
		}
	}

	addresses, err := daemon.WalletList()
	if err != nil {
		panic(err)
	} else if addresses == nil || len(*addresses) == 0 {
		panic(fmt.Errorf("could not find an address in wallet"))
	}
	claimAddress = (*addresses)[0]
	if claimAddress == "" {
		panic(fmt.Errorf("found blank claim address"))
	}

	for i := 0; i < concurrentVideos; i++ {
		go func() {
			wg.Add(1)
			defer wg.Done()

			for {
				v, more := <-videoQueue
				if !more {
					return
				}
				if stopOnError && videoErrored.Load().(bool) {
					log.Println("Video errored. Exiting")
					return
				}

				tryCount := 0
				for {
					tryCount++
					err := processVideo(v)

					if err != nil {
						log.Errorln("error processing video: " + err.Error())
						if stopOnError {
							videoErrored.Store(true)
							sendStopEnqueuing.Do(func() {
								stopEnqueuing <- struct{}{}
							})
						} else if maxTries != defaultMaxTries {
							if strings.Contains(err.Error(), "non 200 status code received") ||
								strings.Contains(err.Error(), " reason: 'This video contains content from") {
								log.Println("This error should not be retried at all")
							} else if tryCount >= maxTries {
								log.Println("Video failed after " + strconv.Itoa(maxTries) + " retries, moving on")
							} else {
								log.Println("Retrying")
								continue
							}
						}
					}
					break
				}
			}
		}()
	}

	err = enqueueVideosFromChannel(channelID, &videoQueue, &stopEnqueuing)
	if err != nil {
		panic(err)
	}
	close(videoQueue)

	wg.Wait()
}

func ensureChannelOwnership() error {
	channels, err := daemon.ChannelListMine()
	if err != nil {
		return err
	} else if channels == nil {
		return fmt.Errorf("no channels")
	}

	for _, channel := range *channels {
		if channel.Name == lbryChannelName {
			return nil
		}
	}

	resolveResp, err := daemon.Resolve(lbryChannelName)
	if err != nil {
		return err
	}

	channelNotFound := (*resolveResp)[lbryChannelName].Error == nil || strings.Contains(*((*resolveResp)[lbryChannelName].Error), "cannot be resolved")

	if !channelNotFound {
		return fmt.Errorf("Channel exists and we don't own it. Pick another channel.")
	}

	_, err = daemon.ChannelNew(lbryChannelName, 0.01)
	if err != nil {
		return err
	}

	// niko's code says "unfortunately the queues in the daemon are not yet merged so we must give it some time for the channel to go through"
	wait := 15 * time.Second
	log.Println("Waiting " + wait.String() + " for channel claim to go through")
	time.Sleep(wait)

	return nil
}

func enqueueVideosFromChannel(channelID string, videoChan *chan video, stopEnqueuing *chan struct{}) error {
	client := &http.Client{
		Transport: &transport.APIKey{Key: ytAPIKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return fmt.Errorf("error creating YouTube service: %v", err)
	}

	response, err := service.Channels.List("contentDetails").Id(channelID).Do()
	if err != nil {
		return fmt.Errorf("error getting channels: %v", err)
	}

	if len(response.Items) < 1 {
		return fmt.Errorf("youtube channel not found")
	}

	if response.Items[0].ContentDetails.RelatedPlaylists == nil {
		return fmt.Errorf("no related playlists")
	}

	playlistID := response.Items[0].ContentDetails.RelatedPlaylists.Uploads
	if playlistID == "" {
		return fmt.Errorf("no channel playlist")
	}

	videos := []video{}

	nextPageToken := ""
	for {
		req := service.PlaylistItems.List("snippet").
			PlaylistId(playlistID).
			MaxResults(50).
			PageToken(nextPageToken)

		playlistResponse, err := req.Do()
		if err != nil {
			return fmt.Errorf("error getting playlist items: %v", err)
		}

		if len(playlistResponse.Items) < 1 {
			return fmt.Errorf("playlist items not found")
		}

		for _, item := range playlistResponse.Items {
			// todo: there's thumbnail info here. why did we need lambda???
			publishedAt, err := time.Parse(time.RFC3339Nano, item.Snippet.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse time: %v", err.Error())
			}

			// normally we'd send the video into the channel here, but youtube api doesn't have sorting
			// so we have to get ALL the videos, then sort them, then send them in
			videos = append(videos, video{
				id:               item.Snippet.ResourceId.VideoId,
				channelID:        channelID,
				title:            item.Snippet.Title,
				description:      item.Snippet.Description,
				channelTitle:     item.Snippet.ChannelTitle,
				playlistPosition: item.Snippet.Position,
				publishedAt:      publishedAt,
			})
		}

		log.Infoln("Got info for " + strconv.Itoa(len(videos)) + " videos from youtube API")

		nextPageToken = playlistResponse.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	sort.Sort(byPublishedAt(videos))
	//or sort.Sort(sort.Reverse(byPlaylistPosition(videos)))

	for _, v := range videos {
		select {
		case *videoChan <- v:
		case <-*stopEnqueuing:
			return nil
		}
	}

	return nil
}

func processVideo(v video) error {
	log.Println("========================================")
	log.Println("Processing " + v.id + " (" + strconv.Itoa(int(v.playlistPosition)) + " in channel)")

	conn := redisPool.Get()
	defer conn.Close()

	alreadyPublished, err := redis.String(conn.Do("HGET", redisHashKey, v.id))
	if err != nil && err != redis.ErrNil {
		return fmt.Errorf("redis error: %s", err.Error())
	}
	if alreadyPublished == redisSyncedVal {
		log.Println(v.id + " already published")
		return nil
	}

	//download and thumbnail can be done in parallel
	err = downloadVideo(v)
	if err != nil {
		return fmt.Errorf("download error: %s", err.Error())
	}

	err = triggerThumbnailSave(v.id)
	if err != nil {
		return fmt.Errorf("thumbnail error: %s", err.Error())
	}

	err = publish(v, conn)
	if err != nil {
		return fmt.Errorf("publish error: %s", err.Error())
	}

	return nil
}

func downloadVideo(v video) error {
	verbose := false
	videoPath := v.getFilename()

	_, err := os.Stat(videoPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		log.Println(v.id + " already exists at " + videoPath)
		return nil
	}

	downloader := ytdl.NewYoutube(verbose)
	err = downloader.DecodeURL("https://www.youtube.com/watch?v=" + v.id)
	if err != nil {
		return err
	}
	err = downloader.StartDownload(videoPath)
	if err != nil {
		return err
	}
	log.Debugln("Downloaded " + v.id)
	return nil
}

func triggerThumbnailSave(videoID string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	params, err := json.Marshal(map[string]string{"videoid": videoID})
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPut, "https://jgp4g1qoud.execute-api.us-east-1.amazonaws.com/prod/thumbnail", bytes.NewBuffer(params))
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var decoded struct {
		error   int    `json:"error"`
		url     string `json:"url,omitempty"`
		message string `json:"message,omitempty"`
	}
	err = json.Unmarshal(contents, &decoded)
	if err != nil {
		return err
	}

	if decoded.error != 0 {
		return fmt.Errorf("error creating thumbnail: " + decoded.message)
	}

	log.Debugln("Created thumbnail for " + videoID)

	return nil
}

func strPtr(s string) *string { return &s }

func titleToClaimName(name string) string {
	maxLen := 40
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)

	chunks := strings.Split(strings.ToLower(strings.Trim(reg.ReplaceAllString(name, "-"), "-")), "-")

	name = chunks[0]
	if len(name) > maxLen {
		return name[:maxLen]
	}

	for _, chunk := range chunks[1:] {
		tmpName := name + "-" + chunk
		if len(tmpName) > maxLen {
			if len(name) < 20 {
				name = tmpName[:maxLen]
			}
			break
		}
		name = tmpName
	}

	return name
}

func limitDescription(description string) string {
	maxLines := 10
	description = strings.TrimSpace(description)
	if strings.Count(description, "\n") < maxLines {
		return description
	}
	return strings.Join(strings.Split(description, "\n")[:maxLines], "\n") + "\n..."
}

func publish(v video, conn redis.Conn) error {
	options := jsonrpc.PublishOptions{
		Title:        &v.title,
		Author:       &v.channelTitle,
		Description:  strPtr(limitDescription(v.description) + "\nhttps://www.youtube.com/watch?v=" + v.id),
		Language:     strPtr("en"),
		ClaimAddress: &claimAddress,
		Thumbnail:    strPtr("http://berk.ninja/thumbnails/" + v.id),
		License:      strPtr("Copyrighted (contact author)"),
	}
	if lbryChannelName != "" {
		options.ChannelName = &lbryChannelName
	}

	_, err := daemon.Publish(titleToClaimName(v.title), v.getFilename(), 0.01, options)
	if err != nil {
		return err
	}

	_, err = redis.Bool(conn.Do("HSET", redisHashKey, v.id, redisSyncedVal))
	if err != nil {
		return fmt.Errorf("redis error: %s", err.Error())
	}

	return nil
}
