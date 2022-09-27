package util

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

var defaultChannel string
var defaultUsername string
var slackApi *slack.Client

// InitSlack Initializes a slack client with the given token and sets the default channel.
func InitSlack(token string, channel string, username string) {
	c := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	slackApi = slack.New(token, slack.OptionHTTPClient(c))
	defaultChannel = channel
	defaultUsername = username
}

// SendToSlackUser Sends message to a specific user.
func SendToSlackUser(user, username, format string, a ...interface{}) error {
	message := format
	if len(a) > 0 {
		message = fmt.Sprintf(format, a...)
	}
	if !strings.HasPrefix(user, "@") {
		user = "@" + user
	}
	return sendToSlack(user, username, message)
}

// SendToSlackChannel Sends message to a specific channel.
func SendToSlackChannel(channel, username, format string, a ...interface{}) error {
	message := format
	if len(a) > 0 {
		message = fmt.Sprintf(format, a...)
	}
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}
	return sendToSlack(channel, username, message)
}

// SendToSlack Sends message to the default channel.
func SendToSlack(format string, a ...interface{}) error {
	message := format
	if len(a) > 0 {
		message = fmt.Sprintf(format, a...)
	}
	if defaultChannel == "" {
		return errors.Err("no default slack channel set")
	}

	return sendToSlack(defaultChannel, defaultUsername, message)
}

func sendToSlack(channel, username, message string) error {
	var err error

	if slackApi == nil {
		err = errors.Err("no slack token provided")
	} else {
		log.Debugln("slack: " + channel + ": " + message)
		_, _, err = slackApi.PostMessage(channel, slack.MsgOptionText(message, false), slack.MsgOptionUsername(username))
	}

	if err != nil {
		log.Errorln("error sending to slack: " + err.Error())
		return err
	}

	return nil
}
