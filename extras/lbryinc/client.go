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

	"github.com/sirupsen/logrus"
)

// Client stores data about internal-apis call it is about to make.
type Client struct {
	ServerAddress string
	AuthToken     string
	Logger        *logrus.Logger
}

// APIResponse reflects internal-apis JSON response format.
type APIResponse struct {
	Success bool          `json:"success"`
	Error   *string       `json:"error"`
	Data    *ResponseData `json:"data"`
}

// ResponseData is a map containing parsed json response.
type ResponseData map[string]interface{}

const defaultServerAddress = "https://api.lbry.com"

// const defaultServerAddress = "http://127.0.0.1:9000/"
const timeout = 5 * time.Second

const userObjectPath = "user"

// NewClient returns a client instance for internal-apis. It requires authToken to be provided
// for authentication.
func NewClient(authToken string) Client {
	return Client{
		ServerAddress: defaultServerAddress,
		AuthToken:     authToken,
		Logger:        logrus.StandardLogger(),
	}
}

func (c Client) getEndpointURL(object, method string) string {
	return fmt.Sprintf("%s/%s/%s", c.ServerAddress, object, method)
}

func (c Client) prepareParams(params map[string]interface{}) (string, error) {
	form := url.Values{}
	form.Add("auth_token", c.AuthToken)
	for k, v := range params {
		if k == "auth_token" {
			return "", errors.New("extra auth_token supplied in request params")
		}
		form.Add(k, fmt.Sprintf("%v", v))
	}
	return form.Encode(), nil
}

func (c Client) doCall(url string, payload string) (body []byte, err error) {
	c.Logger.Debugf("sending payload: %s", payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return body, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: timeout}
	r, err := client.Do(req)
	if err != nil {
		return body, err
	}
	defer r.Body.Close()

	body, err = ioutil.ReadAll(r.Body)
	return body, err
}

// Call calls a remote internal-apis server, returning a response,
// wrapped into standardized API Response struct.
func (c Client) Call(object, method string, params map[string]interface{}) (rd ResponseData, err error) {
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
		return rd, errors.New(*ar.Error)
	}
	return *ar.Data, err
}

// UserMe returns user details for the user associated with the current auth_token
func (c Client) UserMe() (ResponseData, error) {
	return c.Call(userObjectPath, "me", map[string]interface{}{})
}
