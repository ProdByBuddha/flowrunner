// Package utils provides utility functions and abstractions for flowrunner.
package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// HTTPClient provides a reusable HTTP client with common functionality
type HTTPClient struct {
	client *http.Client
}

// HTTPRequest represents an HTTP request
type HTTPRequest struct {
	URL            string                 `json:"url"`
	Method         string                 `json:"method"`
	Headers        map[string]string      `json:"headers,omitempty"`
	QueryParams    map[string]string      `json:"query_params,omitempty"`
	Body           interface{}            `json:"body,omitempty"`
	Timeout        time.Duration          `json:"timeout,omitempty"`
	Auth           map[string]interface{} `json:"auth,omitempty"`
	FollowRedirect bool                   `json:"follow_redirect,omitempty"`
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int                    `json:"status_code"`
	Headers    map[string][]string    `json:"headers"`
	Body       interface{}            `json:"body"`
	RawBody    []byte                 `json:"raw_body,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetTimeout sets the timeout for the HTTP client
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}

// Do executes an HTTP request
func (c *HTTPClient) Do(req *HTTPRequest) (*HTTPResponse, error) {
	// Set default method if not provided
	if req.Method == "" {
		req.Method = "GET"
	}

	// Create request body if provided
	var bodyReader io.Reader
	if req.Body != nil {
		switch body := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(body)
		case []byte:
			bodyReader = bytes.NewBuffer(body)
		default:
			// Marshal JSON body
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonBody)
		}
	}

	// Create HTTP request with query parameters
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters if any
	if len(req.QueryParams) > 0 {
		// Get existing query values or create new ones
		q := parsedURL.Query()

		// Add query parameters
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}

		// Update URL with query parameters
		parsedURL.RawQuery = q.Encode()
	}

	reqURL := parsedURL.String()

	httpReq, err := http.NewRequest(req.Method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range req.Headers {
		httpReq.Header.Add(key, value)
	}

	// Set content type if not provided and body is not nil
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add authentication if provided
	if req.Auth != nil {
		if username, ok := req.Auth["username"].(string); ok {
			if password, ok := req.Auth["password"].(string); ok {
				httpReq.SetBasicAuth(username, password)
			}
		} else if token, ok := req.Auth["token"].(string); ok {
			httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		} else if apiKey, ok := req.Auth["api_key"].(string); ok {
			if keyName, ok := req.Auth["key_name"].(string); ok {
				httpReq.Header.Set(keyName, apiKey)
			} else {
				httpReq.Header.Set("X-API-Key", apiKey)
			}
		}
	}

	// Set client timeout if provided
	originalTimeout := c.client.Timeout
	if req.Timeout > 0 {
		c.client.Timeout = req.Timeout
		defer func() { c.client.Timeout = originalTimeout }()
	}

	// Configure redirect policy
	originalCheckRedirect := c.client.CheckRedirect
	if !req.FollowRedirect {
		c.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		defer func() { c.client.CheckRedirect = originalCheckRedirect }()
	}

	// Record start time for timing information
	startTime := time.Now()

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Calculate request duration
	requestDuration := time.Since(startTime)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response body based on content type
	var parsedBody interface{}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && (contentType == "application/json" || contentType == "application/json; charset=utf-8") {
		if err := json.Unmarshal(body, &parsedBody); err != nil {
			// If JSON parsing fails, use the raw body
			parsedBody = string(body)
		}
	} else {
		// Use raw body for non-JSON responses
		parsedBody = string(body)
	}

	// Create response
	httpResp := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       parsedBody,
		RawBody:    body,
		Metadata: map[string]interface{}{
			"content_type":   contentType,
			"content_length": resp.ContentLength,
			"request_url":    req.URL,
			"request_method": req.Method,
			"timing":         requestDuration,
			"timing_ms":      requestDuration.Milliseconds(),
		},
	}

	return httpResp, nil
}
