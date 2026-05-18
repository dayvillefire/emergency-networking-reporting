// Package enapi is a Go client for the Emergency Networking Department API.
//
// Base URL: https://app.emergencynetworking.com/department-api
// Auth: Bearer token in Authorization header
package enapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// DefaultBaseURL is the default Emergency Networking Department API base URL.
const DefaultBaseURL = "https://app.emergencynetworking.com/department-api"

// Client is the Emergency Networking Department API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	userAgent  string
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// NewClient creates a new API client with the given options.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: http.DefaultClient,
		userAgent:  "enapi-go/1.0",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithToken sets the Bearer token explicitly.
func WithToken(token string) ClientOption {
	return func(c *Client) { c.token = token }
}

// WithTokenFromEnv reads the API token from the API_TOKEN environment variable.
func WithTokenFromEnv() ClientOption {
	return func(c *Client) { c.token = os.Getenv("API_TOKEN") }
}

// HTTPClient returns the underlying *http.Client.
func (c *Client) HTTPClient() *http.Client { return c.httpClient }

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("enapi: create request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) doRequest(req *http.Request, dest any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("enapi: %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("enapi: %s %s: read body: %w", req.Method, req.URL.Path, err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if dest != nil {
			if err := json.Unmarshal(body, dest); err != nil {
				return fmt.Errorf("enapi: %s %s: decode: %w", req.Method, req.URL.Path, err)
			}
		}
		return nil
	}

	apiErr := &APIError{StatusCode: resp.StatusCode}

	// Try to decode as validation error first
	var errResp struct {
		Message string            `json:"message"`
		Errors  map[string]string `json:"errors"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
		apiErr.Message = errResp.Message
		apiErr.Errors = errResp.Errors
		return apiErr
	}

	// Plain text error or unrecognized JSON
	apiErr.Message = string(body)
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	return apiErr
}

func (c *Client) doGet(ctx context.Context, path string, query url.Values, dest any) error {
	u := path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	return c.doRequest(req, dest)
}

func (c *Client) doPost(ctx context.Context, path string, body any, dest any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return fmt.Errorf("enapi: encode body: %w", err)
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, &buf)
	if err != nil {
		return err
	}
	return c.doRequest(req, dest)
}

func (c *Client) doPut(ctx context.Context, path string, body any, dest any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return fmt.Errorf("enapi: encode body: %w", err)
	}
	req, err := c.newRequest(ctx, http.MethodPut, path, &buf)
	if err != nil {
		return err
	}
	return c.doRequest(req, dest)
}
