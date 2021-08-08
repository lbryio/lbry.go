package lbryinc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"

	log "github.com/sirupsen/logrus"
)

const (
	defaultServerAddress = "https://api.lbry.com"
	timeout              = 5 * time.Second
	headerForwardedFor   = "X-Forwarded-For"

	userObjectPath             = "user"
	userMeMethod               = "me"
	userHasVerifiedEmailMethod = "has_verified_email"
)

// Client stores data about internal-apis call it is about to make.
type Client struct {
	AuthToken     string
	OAuthToken    oauth2.TokenSource
	Logger        *log.Logger
	serverAddress string
	extraHeaders  map[string]string
}

// ClientOpts allow to provide extra parameters to NewClient:
// - ServerAddress
// - RemoteIP — to forward the IP of a frontend client making the request
type ClientOpts struct {
	ServerAddress string
	RemoteIP      string
}

// APIResponse reflects internal-apis JSON response format.
type APIResponse struct {
	Success bool          `json:"success"`
	Error   *string       `json:"error"`
	Data    *ResponseData `json:"data"`
}

// APIError wraps errors returned by LBRY API server to discern them from other kinds (like http errors).
type APIError struct {
	Err error
}

func (e APIError) Error() string {
	return fmt.Sprintf("api error: %v", e.Err)
}

// ResponseData is a map containing parsed json response.
type ResponseData map[string]interface{}

func makeMethodPath(obj, method string) string {
	return fmt.Sprintf("/%s/%s", obj, method)
}

// NewClient returns a client instance for internal-apis. It requires authToken to be provided
// for authentication.
func NewClient(authToken string, opts *ClientOpts) Client {
	c := Client{
		serverAddress: defaultServerAddress,
		extraHeaders:  make(map[string]string),
		AuthToken:     authToken,
		Logger:        log.StandardLogger(),
	}
	if opts != nil {
		if opts.ServerAddress != "" {
			c.serverAddress = opts.ServerAddress
		}
		if opts.RemoteIP != "" {
			c.extraHeaders[headerForwardedFor] = opts.RemoteIP
		}
	}

	return c
}

// NewOauthClient returns a client instance for internal-apis. It requires Oauth Token Source to be provided
// for authentication.
func NewOauthClient(token oauth2.TokenSource, opts *ClientOpts) Client {
	c := Client{
		serverAddress: defaultServerAddress,
		extraHeaders:  make(map[string]string),
		OAuthToken:    token,
		Logger:        log.StandardLogger(),
	}
	if opts != nil {
		if opts.ServerAddress != "" {
			c.serverAddress = opts.ServerAddress
		}
		if opts.RemoteIP != "" {
			c.extraHeaders[headerForwardedFor] = opts.RemoteIP
		}
	}

	return c
}

func (c Client) getEndpointURL(object, method string) string {
	return fmt.Sprintf("%s%s", c.serverAddress, makeMethodPath(object, method))
}

func (c Client) prepareParams(params map[string]interface{}) (string, error) {
	form := url.Values{}
	if c.AuthToken != "" {
		form.Add("auth_token", c.AuthToken)
	} else if c.OAuthToken == nil {
		return "", errors.New("oauth token source must be supplied")
	}
	for k, v := range params {
		if k == "auth_token" {
			return "", errors.New("extra auth_token supplied in request params")
		}
		form.Add(k, fmt.Sprintf("%v", v))
	}
	return form.Encode(), nil
}

func (c Client) doCall(url string, payload string) ([]byte, error) {
	var body []byte
	c.Logger.Debugf("sending payload: %s", payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return body, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if c.OAuthToken != nil {
		t, err := c.OAuthToken.Token()
		if err != nil {
			return nil, err
		}
		if t.Type() != "Bearer" {
			return nil, errors.New("internal-apis requires an oAuth token of type 'Bearer'")
		}
		t.SetAuthHeader(req)
	}

	for k, v := range c.extraHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: timeout}
	r, err := client.Do(req)
	if err != nil {
		return body, err
	}
	if r.StatusCode >= 500 {
		return body, fmt.Errorf("server returned non-OK status: %v", r.StatusCode)
	}
	defer r.Body.Close()
	return ioutil.ReadAll(r.Body)
}

// Call calls a remote internal-apis server, returning a response,
// wrapped into standardized API Response struct.
func (c Client) Call(object, method string, params map[string]interface{}) (ResponseData, error) {
	var rd ResponseData
	payload, err := c.prepareParams(params)
	if err != nil {
		return rd, err
	}

	body, err := c.doCall(c.getEndpointURL(object, method), payload)
	if err != nil {
		return rd, err
	}
	var ar APIResponse
	err = json.Unmarshal(body, &ar)
	if err != nil {
		return rd, err
	}
	if !ar.Success {
		return rd, APIError{errors.New(*ar.Error)}
	}
	return *ar.Data, err
}

// UserMe returns user details for the user associated with the current auth_token.
func (c Client) UserMe() (ResponseData, error) {
	return c.Call(userObjectPath, userMeMethod, map[string]interface{}{})
}

// UserHasVerifiedEmail calls has_verified_email method.
func (c Client) UserHasVerifiedEmail() (ResponseData, error) {
	return c.Call(userObjectPath, userHasVerifiedEmailMethod, map[string]interface{}{})
}
