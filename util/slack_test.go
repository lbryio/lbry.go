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
	SendToSlack("This is a test :)")
}
