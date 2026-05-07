package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error response from the API.
type APIError struct {
	Status  int
	Message string
	Code    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error (%d): %s", e.Status, e.Message)
}

// APIClient is an HTTP client for the Symbiont API.
type APIClient struct {
	BaseURL string
	Token   string
	Source  string // sent as X-Source header; sets initiated_by on mutating operations
	HTTP    *http.Client
}

// NewAPIClient creates a new API client.
func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Token:   token,
		HTTP:    &http.Client{},
	}
}

// WithSource returns the client with Source set to s.
func (c *APIClient) WithSource(s string) *APIClient {
	c.Source = s
	return c
}

// Get sends a GET request and decodes the response into result.
func (c *APIClient) Get(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodGet, path, nil, result)
}

// Put sends a PUT request with a JSON body and decodes the response.
func (c *APIClient) Put(ctx context.Context, path string, body any, result any) error {
	return c.do(ctx, http.MethodPut, path, body, result)
}

// Post sends a POST request with a JSON body and decodes the response.
func (c *APIClient) Post(ctx context.Context, path string, body any, result any) error {
	return c.do(ctx, http.MethodPost, path, body, result)
}

// Delete sends a DELETE request.
func (c *APIClient) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// GetBytes sends a GET request and returns the raw response body.
func (c *APIClient) GetBytes(ctx context.Context, path string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if c.Source != "" {
		req.Header.Set("X-Source", c.Source)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, "", &APIError{Status: resp.StatusCode, Message: "request failed"}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading response: %w", err)
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func (c *APIClient) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if c.Source != "" {
		req.Header.Set("X-Source", c.Source)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil || apiErr.Error == "" {
			apiErr.Error = "unexpected error response"
		}
		return &APIError{
			Status:  resp.StatusCode,
			Message: apiErr.Error,
			Code:    apiErr.Code,
		}
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
