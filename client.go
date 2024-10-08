// Package tinkoff allows sending token-signed requests to Tinkoff Acquiring API and parse incoming HTTP notifications
package tinkoff

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	httpClient httpDoer

	terminalKey string
	password    string
	baseURL     string
}

func WithTerminalKey(terminalKey string) func(*Config) {
	return func(config *Config) {
		config.terminalKey = terminalKey
	}
}

func WithPassword(password string) func(*Config) {
	return func(config *Config) {
		config.password = password
	}
}

func WithBaseURL(baseURL string) func(*Config) {
	return func(config *Config) {
		config.baseURL = baseURL
	}
}

func WithHTTPClient(c httpDoer) func(*Config) {
	return func(config *Config) {
		config.httpClient = c
	}
}

// Client is the main entity which executes requests against the Tinkoff Acquiring API endpoint
type Client struct {
	Config
}

// NewClient returns new Client instance
func NewClient(terminalKey, password string) *Client {
	return NewClientWithOptions(
		WithTerminalKey(terminalKey),
		WithPassword(password),
	)
}

func NewClientWithOptions(cfgOption ...func(*Config)) *Client {
	defaultConfig := Config{
		httpClient: http.DefaultClient,
		baseURL:    "https://securepay.tinkoff.ru/v2",
	}
	cfg := defaultConfig

	for _, opt := range cfgOption {
		opt(&cfg)
	}

	return &Client{
		Config: cfg,
	}
}

// SetBaseURL allows to change default API endpoint
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *Client) decodeResponse(response *http.Response, result interface{}) error {
	return json.NewDecoder(response.Body).Decode(result)
}

// Deprecated: use PostRequestWithContext instead
func (c *Client) PostRequest(url string, request RequestInterface) (*http.Response, error) {
	return c.PostRequestWithContext(context.Background(), url, request)
}

// PostRequestWithContext will automatically sign the request with token
// Use BaseRequest type to implement any API request
func (c *Client) PostRequestWithContext(ctx context.Context, url string, request RequestInterface) (*http.Response, error) {
	c.secureRequest(request)
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if !c.isSuccessResponseCode(resp) {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, c.returnErrOnResponse(resp.StatusCode, body)
	}

	return resp, nil
}

func (c *Client) returnErrOnResponse(code int, body []byte) error {
	return errors.New("error code: " + strconv.Itoa(code) + "message: " + string(body[:]))
}

func (c *Client) isSuccessResponseCode(resp *http.Response) bool {
	return resp.StatusCode == 200 || resp.StatusCode == 201 || resp.StatusCode == 204
}

func (c *Client) secureRequest(request RequestInterface) {
	request.SetTerminalKey(c.terminalKey)

	v := request.GetValuesForToken()
	v["TerminalKey"] = c.terminalKey
	v["Password"] = c.password
	request.SetToken(generateToken(v))
}

func generateToken(v map[string]string) string {
	keys := make([]string, 0)
	for key := range v {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b bytes.Buffer
	for _, key := range keys {
		b.WriteString(v[key])
	}
	sum := sha256.Sum256(b.Bytes())
	return fmt.Sprintf("%x", sum)
}
