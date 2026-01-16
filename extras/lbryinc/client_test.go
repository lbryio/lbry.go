package lbryinc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func launchDummyServer(lastReq **http.Request, path, response string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if lastReq != nil {
			*lastReq = &*r
		}
		authT := r.FormValue("auth_token")
		if authT == "" {
			accessT := r.Header.Get("Authorization")
			if accessT == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		if r.URL.Path != path {
			fmt.Printf("path doesn't match: %v != %v", r.URL.Path, path)
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			w.Write([]byte(response))
		}
	}))
}

func TestUserMe(t *testing.T) {
	ts := launchDummyServer(nil, makeMethodPath(userObjectPath, userMeMethod), userMeResponse, http.StatusOK)
	defer ts.Close()

	c := NewClient("realToken", &ClientOpts{ServerAddress: ts.URL})
	r, err := c.UserMe()
	assert.Nil(t, err)
	robj, err := r.Object()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "user@lbry.tv", robj["primary_email"])
}

func TestListFiltered(t *testing.T) {
	ts := launchDummyServer(nil, "/file/list_filtered", listFilteredResponse, http.StatusOK)
	defer ts.Close()

	c := NewClient("realToken", &ClientOpts{ServerAddress: ts.URL})
	r, err := c.CallResource("file", "list_filtered", map[string]interface{}{"with_claim_id": "true"})
	assert.Nil(t, err)
	assert.True(t, r.IsArray())
	_, err = r.Array()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUserHasVerifiedEmail(t *testing.T) {
	ts := launchDummyServer(nil, makeMethodPath(userObjectPath, userHasVerifiedEmailMethod), userHasVerifiedEmailResponse, http.StatusOK)
	defer ts.Close()

	c := NewClient("realToken", &ClientOpts{ServerAddress: ts.URL})
	r, err := c.UserHasVerifiedEmail()
	assert.Nil(t, err)
	robj, err := r.Object()
	if err != nil {
		t.Error(err)
	}
	assert.EqualValues(t, 12345, robj["user_id"])
	assert.Equal(t, true, robj["has_verified_email"])
}

func TestUserHasVerifiedEmailOAuth(t *testing.T) {
	ts := launchDummyServer(nil, makeMethodPath(userObjectPath, userHasVerifiedEmailMethod), userHasVerifiedEmailResponse, http.StatusOK)
	defer ts.Close()

	c := NewOauthClient(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "Test-Access-Token"}), &ClientOpts{ServerAddress: ts.URL})
	r, err := c.UserHasVerifiedEmail()
	assert.Nil(t, err)
	robj, err := r.Object()
	if err != nil {
		t.Error(err)
	}
	assert.EqualValues(t, 12345, robj["user_id"])
	assert.Equal(t, true, robj["has_verified_email"])
}

func TestRemoteIP(t *testing.T) {
	var req *http.Request
	ts := launchDummyServer(&req, makeMethodPath(userObjectPath, userMeMethod), userMeResponse, http.StatusOK)
	defer ts.Close()

	c := NewClient("realToken", &ClientOpts{ServerAddress: ts.URL, RemoteIP: "8.8.8.8"})
	_, err := c.UserMe()
	assert.Nil(t, err)
	assert.Equal(t, []string{"8.8.8.8"}, req.Header["X-Forwarded-For"])
}

func TestWrongToken(t *testing.T) {
	c := NewClient("zcasdasc", nil)

	r, err := c.UserHasVerifiedEmail()
	assert.False(t, r.IsObject())
	assert.EqualError(t, err, "api error: could not authenticate user")
	assert.ErrorAs(t, err, &APIError{})
}

func TestHTTPError(t *testing.T) {
	c := NewClient("zcasdasc", &ClientOpts{ServerAddress: "http://lolcathost"})

	r, err := c.UserHasVerifiedEmail()
	assert.False(t, r.IsObject())
	assert.EqualError(t, err, `Post "http://lolcathost/user/has_verified_email": dial tcp: lookup lolcathost: no such host`)
}

func TestGatewayError(t *testing.T) {
	var req *http.Request
	ts := launchDummyServer(&req, makeMethodPath(userObjectPath, userHasVerifiedEmailMethod), "", http.StatusBadGateway)
	defer ts.Close()
	c := NewClient("zcasdasc", &ClientOpts{ServerAddress: ts.URL})

	r, err := c.UserHasVerifiedEmail()
	assert.False(t, r.IsObject())
	assert.EqualError(t, err, `server returned non-OK status: 502`)
}

const userMeResponse = `{
	"success": true,
	"error": null,
	"data": {
		"id": 12345,
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
		"manual_approval_user_id": 654,
		"reward_status_change_trigger": "manual",
		"primary_email": "user@lbry.tv",
		"has_verified_email": true,
		"is_identity_verified": false,
		"is_reward_approved": true,
		"groups": []
	}
}`

const userHasVerifiedEmailResponse = `{
	"success": true,
	"error": null,
	"data": {
	  "user_id": 12345,
	  "has_verified_email": true
	}
}`

const listFilteredResponse = `{
  "success": true,
  "error": null,
  "data": [
    {
      "claim_id": "322ce77e9085d9da42279c790f7c9755b4916fca",
      "outpoint": "20e04af21a569061ced7aa1801a43b4ed4839dfeb79919ea49a4059c7fe114c5:0"
    },
    {
      "claim_id": "61496c567badcd98b82d9a700a8d56fd8a5fa8fb",
      "outpoint": "657e4ec774524b326f9d3ecb9f468ea085bd1f3d450565f0330feca02e8fd25b:0"
    }
  ]
}`
