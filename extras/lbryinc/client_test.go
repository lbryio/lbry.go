package lbryinc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMeWrongToken(t *testing.T) {
	c := NewClient("abc", nil)
	r, err := c.UserMe()
	require.NotNil(t, err)
	assert.Equal(t, "could not authenticate user", err.Error())
	assert.Nil(t, r)
}

func launchDummyServer(lastReq **http.Request) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*lastReq = &*r
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		response := []byte(`{
			"success": true,
			"error": null,
			"data": {
				"id": 751365,
				"language": "en",
				"given_name": null,
				"family_name": null,
				"created_at": "2019-01-17T12:13:06Z",
				"updated_at": "2019-05-02T13:57:59Z",
				"invited_by_id": null,
				"invited_at": null,
				"invites_remaining": 0,
				"invite_reward_claimed": false,
				"is_email_enabled": true,
				"manual_approval_user_id": 837139,
				"reward_status_change_trigger": "manual",
				"primary_email": "andrey@lbry.com",
				"has_verified_email": true,
				"is_identity_verified": false,
				"is_reward_approved": true,
				"groups": []
			}
			}`)
		w.Write(response)
	}))
}

func TestUserMe(t *testing.T) {
	var req *http.Request
	ts := launchDummyServer(&req)
	defer ts.Close()

	c := NewClient("realToken", &ClientOpts{ServerAddress: ts.URL})
	r, err := c.UserMe()
	assert.Nil(t, err)
	assert.Equal(t, r["primary_email"], "andrey@lbry.com")
}

func TestRemoteIP(t *testing.T) {
	var req *http.Request
	ts := launchDummyServer(&req)
	defer ts.Close()

	c := NewClient("realToken", &ClientOpts{ServerAddress: ts.URL, RemoteIP: "8.8.8.8"})
	_, err := c.UserMe()
	assert.Nil(t, err)
	assert.Equal(t, []string{"8.8.8.8"}, req.Header["X-Forwarded-For"])
}
