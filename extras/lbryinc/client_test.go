package lbryinc

import (
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMeWrongToken(t *testing.T) {
	c := NewClient("abc")
	r, err := c.UserMe()
	require.NotNil(t, err)
	assert.Equal(t, "could not authenticate user", err.Error())
	assert.Nil(t, r)
}

const dummyServerURL = "http://127.0.0.1:59999"

func launchDummyServer() {
	s := &http.Server{
		Addr: "127.0.0.1:59999",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		}),
	}
	log.Fatal(s.ListenAndServe())
}

func TestClient_Set_GetServerAddress(t *testing.T) {
	c := NewClient("realToken")
	assert.Equal(t, defaultAPIHost, c.GetServerAddress())
	c.SetServerAddress("http://host.com/api")
	assert.Equal(t, "http://host.com/api", c.GetServerAddress())
}

func TestUserMe(t *testing.T) {
	go launchDummyServer()
	c := NewClient("realToken")
	c.SetServerAddress(dummyServerURL)
	r, err := c.UserMe()
	assert.Nil(t, err)
	assert.Equal(t, r["primary_email"], "andrey@lbry.com")
}
