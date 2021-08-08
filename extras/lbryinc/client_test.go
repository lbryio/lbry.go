package lbryinc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, "user@lbry.tv", r["primary_email"])
}

func TestUserHasVerifiedEmail(t *testing.T) {
	ts := launchDummyServer(nil, makeMethodPath(userObjectPath, userHasVerifiedEmailMethod), userHasVerifiedEmailResponse, http.StatusOK)
	defer ts.Close()

	c := NewClient("realToken", &ClientOpts{ServerAddress: ts.URL})
	r, err := c.UserHasVerifiedEmail()
	assert.Nil(t, err)
	assert.EqualValues(t, 12345, r["user_id"])
	assert.Equal(t, true, r["has_verified_email"])
}

func TestUserHasVerifiedEmailOAuth(t *testing.T) {
	ts := launchDummyServer(nil, makeMethodPath(userObjectPath, userHasVerifiedEmailMethod), userHasVerifiedEmailResponse, http.StatusOK)
	defer ts.Close()

	c := NewOauthClient(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "Test-Access-Token"}), &ClientOpts{ServerAddress: ts.URL})
	r, err := c.UserHasVerifiedEmail()
	assert.Nil(t, err)
	assert.EqualValues(t, 12345, r["user_id"])
	assert.Equal(t, true, r["has_verified_email"])
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
	assert.Nil(t, r)
	assert.EqualError(t, err, "api error: could not authenticate user")
	assert.ErrorAs(t, err, &APIError{})
}

func TestHTTPError(t *testing.T) {
	c := NewClient("zcasdasc", &ClientOpts{ServerAddress: "http://lolcathost"})

	r, err := c.UserHasVerifiedEmail()
	assert.Nil(t, r)
	assert.EqualError(t, err, `Post "http://lolcathost/user/has_verified_email": dial tcp: lookup lolcathost: no such host`)
}

func TestGatewayError(t *testing.T) {
	var req *http.Request
	ts := launchDummyServer(&req, makeMethodPath(userObjectPath, userHasVerifiedEmailMethod), "", http.StatusBadGateway)
	defer ts.Close()
	c := NewClient("zcasdasc", &ClientOpts{ServerAddress: ts.URL})

	r, err := c.UserHasVerifiedEmail()
	assert.Nil(t, r)
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
