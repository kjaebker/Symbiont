package apex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client defines the interface for communicating with a Neptune Apex controller.
type Client interface {
	// Status fetches the full status snapshot from the Apex.
	Status(ctx context.Context) (*StatusResponse, error)

	// SetOutlet changes an outlet's runtime state without modifying its program.
	// Only ON and OFF are supported — the Apex REST API has no programmatic
	// way to clear a manual override and return to program control (AUTO).
	SetOutlet(ctx context.Context, did string, state OutletState) error
}

// client implements Client using the Apex local REST API.
type client struct {
	baseURL  string
	username string
	password string
	http     *http.Client

	mu     sync.Mutex
	cookie string // connect.sid session cookie value
}

// NewClient creates a new Apex client. Authentication is lazy — the first
// request triggers a login. The baseURL should be the Apex's local IP
// (e.g. "http://192.168.1.100").
func NewClient(baseURL, user, pass string) Client {
	return &client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: user,
		password: pass,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// login authenticates with the Apex and stores the session cookie.
func (c *client) login(ctx context.Context) error {
	body := LoginRequest{
		Login:      c.username,
		Password:   c.password,
		RememberMe: true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/rest/login", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sending login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: status %d", resp.StatusCode)
	}

	// Try to extract connect.sid from Set-Cookie header first.
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "connect.sid" {
			c.cookie = cookie.Value
			return nil
		}
	}

	// Fall back to response body JSON.
	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("decoding login response: %w", err)
	}
	if loginResp.ConnectSID == "" {
		return fmt.Errorf("login response missing connect.sid")
	}
	c.cookie = loginResp.ConnectSID
	return nil
}

// do executes an HTTP request with session cookie authentication.
// If no cookie exists, login is called first. On 401, login is retried once.
func (c *client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	needsLogin := c.cookie == ""
	c.mu.Unlock()

	if needsLogin {
		c.mu.Lock()
		err := c.login(ctx)
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("initial login: %w", err)
		}
	}

	c.mu.Lock()
	cookie := c.cookie
	c.mu.Unlock()

	req.Header.Set("Cookie", "connect.sid="+cookie)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	// If 401, re-authenticate and retry once.
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		c.mu.Lock()
		err := c.login(ctx)
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("re-authentication: %w", err)
		}

		// Rebuild the request for retry — the original body may be consumed.
		c.mu.Lock()
		cookie = c.cookie
		c.mu.Unlock()

		req.Header.Set("Cookie", "connect.sid="+cookie)

		// For requests with a body, we need a fresh body. The caller must use
		// GetBody if set, or the request must be re-created. For our use cases
		// (GET with no body, PUT with GetBody), this works.
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("getting request body for retry: %w", err)
			}
			req.Body = body
		}

		resp, err = c.http.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// Status fetches the full status snapshot from the Apex controller.
func (c *client) Status(ctx context.Context) (*StatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/rest/status", nil)
	if err != nil {
		return nil, fmt.Errorf("creating status request: %w", err)
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetching status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status request failed: status %d, body %s", resp.StatusCode, body)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decoding status response: %w", err)
	}
	return &status, nil
}

// SetOutlet changes an outlet's runtime state. This is a safe toggle that
// preserves the outlet's program — it does NOT overwrite outlet configuration.
func (c *client) SetOutlet(ctx context.Context, did string, state OutletState) error {
	controlReq := NewOutletControlRequest(did, state)
	payload, err := json.Marshal(controlReq)
	if err != nil {
		return fmt.Errorf("marshaling outlet control request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/rest/status/outputs/"+did, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating outlet control request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Set GetBody so the request body can be replayed on 401 retry.
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		return fmt.Errorf("setting outlet %s to %s: %w", did, state, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("outlet control failed: status %d, body %s", resp.StatusCode, body)
	}

	return nil
}

