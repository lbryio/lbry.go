package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/jsonrpc"

	"github.com/garyburd/redigo/redis"
	ytdl "github.com/kkdai/youtube"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

const (
	concurrentVideos = 1
	redisHashKey     = "ytsync"
	redisSyncedVal   = "t"
)

type video struct {
	id           string
	channelID    string
	channelTitle string
	title        string
	description  string
}

func (v video) getFilename() string {
	return videoDirectory + "/" + v.id + ".mp4"
}

var (
	daemon          *jsonrpc.Client
	channelID       string
	lbryChannelName string
	claimAddress    string
	videoDirectory  string
	ytAPIKey        string
	redisPool       *redis.Pool
)

func ytsync() {
	var err error
	flag.StringVar(&ytAPIKey, "ytApiKey", "", "Youtube API key (required)")
	flag.StringVar(&channelID, "channelID", "", "ID of the youtube channel to sync (required)")
	flag.StringVar(&lbryChannelName, "lbryChannel", "", "Publish videos into this channel")
	flag.Parse()

	if channelID == "" || ytAPIKey == "" {
		flag.Usage()
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
		panic(fmt.Errorf("Could not find an address in wallet"))
	}
	claimAddress = (*addresses)[0]
	if claimAddress == "" {
		panic(fmt.Errorf("Found blank claim address"))
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
				err := processVideo(v)
				if err != nil {
					log.Errorln("error processing video: " + err.Error())
				}
			}
		}()
	}

	err = enqueueVideosFromChannel(channelID, &videoQueue)
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
		return fmt.Errorf("No channels")
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

func enqueueVideosFromChannel(channelID string, videoChan *chan video) error {
	client := &http.Client{
		Transport: &transport.APIKey{Key: ytAPIKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return fmt.Errorf("Error creating YouTube service: %v", err)
	}

	response, err := service.Channels.List("contentDetails").Id(channelID).Do()
	if err != nil {
		return fmt.Errorf("Error getting channels: %v", err)
	}

	if len(response.Items) < 1 {
		return fmt.Errorf("Youtube channel not found")
	}

	if response.Items[0].ContentDetails.RelatedPlaylists == nil {
		return fmt.Errorf("No related playlists")
	}

	playlistID := response.Items[0].ContentDetails.RelatedPlaylists.Uploads
	if playlistID == "" {
		return fmt.Errorf("No channel playlist")
	}

	firstRequest := true
	nextPageToken := ""

	for firstRequest || nextPageToken != "" {
		req := service.PlaylistItems.List("snippet").PlaylistId(playlistID).MaxResults(50)
		if nextPageToken != "" {
			req.PageToken(nextPageToken)
		}

		playlistResponse, err := req.Do()
		if err != nil {
			return fmt.Errorf("Error getting playlist items: %v", err)
		}

		if len(playlistResponse.Items) < 1 {
			return fmt.Errorf("Playlist items not found")
		}

		for _, item := range playlistResponse.Items {
			// todo: there's thumbnail info here. why did we need lambda???
			*videoChan <- video{
				id:           item.Snippet.ResourceId.VideoId,
				channelID:    channelID,
				title:        item.Snippet.Title,
				description:  item.Snippet.Description,
				channelTitle: item.Snippet.ChannelTitle,
			}
		}

		nextPageToken = playlistResponse.NextPageToken
		firstRequest = false
	}

	return nil
}

func processVideo(v video) error {
	log.Println("Processing " + v.id)

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
		Description:  strPtr(limitDescription(v.description)),
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
