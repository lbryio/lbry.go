package util

import (
	"os"
	"testing"
)

func TestSendToSlack(t *testing.T) {
	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		t.Error("A slack token was not provided")
	}
	host, err := os.Hostname()
	if err != nil {
		host = "ytsync-unknown"
	}
	InitSlack(os.Getenv("SLACK_TOKEN"), os.Getenv("SLACK_CHANNEL"), host)
	SendInfoToSlack("This is a test :) Working %.2f%%", 1.01*100)
	SendErrorToSlack("This is a test :) Working %.2f%%", 0.01*100)
}
