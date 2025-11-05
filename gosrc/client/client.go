package client

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
)

// Maximum retries for http requests.
const MaxRetries = 3

// Client connection to dispatcher.
type Client struct {
	Author        events.PluginEntity
	eventsUrl     string
	dataUrl       string
	deploymentKey string
	summary       events.EventAuthor
	client        *http.Client
	auth          bool
	username      string
	password      string
	UserAgent     string
}

type HttpError struct {
	Body       string
	StatusCode int
}

func (r *HttpError) Error() string {
	if len(r.Body) <= 0 {
		return fmt.Sprintf("http response error %v - no body available", r.StatusCode)
	}
	return fmt.Sprintf("http response error %v - %v", r.StatusCode, string(r.Body))
}

// statusError formats an error for the supplied http response
func statusError(resp *http.Response) *HttpError {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		body = []byte{}
	}
	return &HttpError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}
}

// Return a client connection to dispatcher.
func NewClient(eventsUrl string, dataUrl string, author events.PluginEntity, deploymentKey string) *Client {
	conn := Client{
		Author:        author,
		eventsUrl:     eventsUrl,
		dataUrl:       dataUrl,
		deploymentKey: deploymentKey,
		summary: events.EventAuthor{
			Name:     author.Name,
			Version:  author.Version,
			Category: author.Category,
		},
		client:    &http.Client{},
		auth:      false,
		UserAgent: fmt.Sprintf("dpclient-go-%s", author.Name),
	}
	return &conn
}

func (c *Client) SetAuth(username, password string) {
	c.username = username
	c.password = password
	c.auth = true
}

// Perform the provided http request and if the request results in an error retry up to MaxRetry times
// with a 5 second delay.
// If any status code is returned, including 4xx/5xx, this is not considered an error.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.auth {
		req.SetBasicAuth(c.username, c.password)
	}
	// custom user agent
	req.Header.Add("user-agent", c.UserAgent)
	var resp *http.Response
	var err error

	for i := 0; i < MaxRetries; i++ {
		resp, err = c.client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Second)
	}
	return resp, err
}
