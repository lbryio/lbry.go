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

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	defaultServerAddress = "https://api.odysee.tv"
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
	Success bool        `json:"success"`
	Error   *string     `json:"error"`
	Data    interface{} `json:"data"`
}

type data struct {
	obj   map[string]interface{}
	array []interface{}
}

func (d data) IsObject() bool {
	return d.obj != nil
}

func (d data) IsArray() bool {
	return d.array != nil
}

func (d data) Object() (map[string]interface{}, error) {
	if d.obj == nil {
		return nil, errors.New("no object data found")
	}
	return d.obj, nil
}

func (d data) Array() ([]interface{}, error) {
	if d.array == nil {
		return nil, errors.New("no array data found")
	}
	return d.array, nil
}

// APIError wraps errors returned by LBRY API server to discern them from other kinds (like http errors).
type APIError struct {
	Err error
}

func (e APIError) Error() string {
	return fmt.Sprintf("api error: %v", e.Err)
}

// ResponseData is a map containing parsed json response.
type ResponseData interface {
	IsObject() bool
	IsArray() bool
	Object() (map[string]interface{}, error)
	Array() ([]interface{}, error)
}

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

func (c Client) getEndpointURLFromPath(path string) string {
	return fmt.Sprintf("%s%s", c.serverAddress, path)
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

// CallResource calls a remote internal-apis server resource, returning a response,
// wrapped into standardized API Response struct.
func (c Client) CallResource(object, method string, params map[string]interface{}) (ResponseData, error) {
	var d data
	payload, err := c.prepareParams(params)
	if err != nil {
		return d, err
	}

	body, err := c.doCall(c.getEndpointURL(object, method), payload)
	if err != nil {
		return d, err
	}
	var ar APIResponse
	err = json.Unmarshal(body, &ar)
	if err != nil {
		return d, err
	}
	if !ar.Success {
		return d, APIError{errors.New(*ar.Error)}
	}
	if v, ok := ar.Data.([]interface{}); ok {
		d.array = v
	} else if v, ok := ar.Data.(map[string]interface{}); ok {
		d.obj = v
	}
	return d, err
}

// Call calls a remote internal-apis server, returning a response,
// wrapped into standardized API Response struct.
func (c Client) Call(path string, params map[string]interface{}) (ResponseData, error) {
	var d data
	payload, err := c.prepareParams(params)
	if err != nil {
		return d, err
	}

	body, err := c.doCall(c.getEndpointURLFromPath(path), payload)
	if err != nil {
		return d, err
	}
	var ar APIResponse
	err = json.Unmarshal(body, &ar)
	if err != nil {
		return d, err
	}
	if !ar.Success {
		return d, APIError{errors.New(*ar.Error)}
	}
	if v, ok := ar.Data.([]interface{}); ok {
		d.array = v
	} else if v, ok := ar.Data.(map[string]interface{}); ok {
		d.obj = v
	}
	return d, err
}

// UserMe returns user details for the user associated with the current auth_token.
func (c Client) UserMe() (ResponseData, error) {
	return c.CallResource(userObjectPath, userMeMethod, map[string]interface{}{})
}

// UserHasVerifiedEmail calls has_verified_email method.
func (c Client) UserHasVerifiedEmail() (ResponseData, error) {
	return c.CallResource(userObjectPath, userHasVerifiedEmailMethod, map[string]interface{}{})
}
