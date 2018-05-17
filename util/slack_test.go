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
	InitSlack(slackToken)
	SendToSlackInfo("This is a test :) Working %.2f%%", 1.01*100)
	SendToSlackError("This is a test :) Working %.2f%%", 0.01*100)
}
