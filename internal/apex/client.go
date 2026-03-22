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
	SetOutlet(ctx context.Context, did string, state OutletState) error

	// SetOutletAuto returns an outlet to program control (AUTO) using the
	// legacy CGI endpoint, which supports state=0 for AUTO. The outletName
	// must be the Apex outlet name (not DID).
	SetOutletAuto(ctx context.Context, outletName string) error
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

// SetOutletAuto returns an outlet to program control using the legacy CGI
// endpoint: POST /cgi-bin/status.cgi with {name}_state=0. The REST API
// (AOS 5.x+) has no equivalent — this CGI endpoint is the only known way
// to programmatically return an outlet to AUTO.
func (c *client) SetOutletAuto(ctx context.Context, outletName string) error {
	c.mu.Lock()
	needsLogin := c.cookie == ""
	c.mu.Unlock()

	if needsLogin {
		c.mu.Lock()
		err := c.login(ctx)
		c.mu.Unlock()
		if err != nil {
			return fmt.Errorf("login for auto: %w", err)
		}
	}

	// noResponse=1 tells the Apex to skip sending a response body.
	// This avoids parsing a full HTML status page, but the Apex sends back
	// a bare "0" which is not valid HTTP — Go's client returns a transport
	// error. We treat that as success since the command was accepted.
	payload := outletName + "_state=0&noResponse=1"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/cgi-bin/status.cgi", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating CGI auto request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.mu.Lock()
	cookie := c.cookie
	c.mu.Unlock()
	req.Header.Set("Cookie", "connect.sid="+cookie)

	resp, err := c.http.Do(req)
	if err != nil {
		// The Apex with noResponse=1 sends a malformed HTTP response ("0").
		// Go's HTTP client returns a transport error, but the command succeeded.
		if strings.Contains(err.Error(), "malformed HTTP response") {
			return nil
		}
		return fmt.Errorf("sending CGI auto request: %w", err)
	}
	defer resp.Body.Close()

	// If we got a real HTTP response, check the status.
	if resp.StatusCode == http.StatusUnauthorized {
		// Retry with basic auth.
		resp.Body.Close()
		req2, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/cgi-bin/status.cgi", strings.NewReader(payload))
		if err != nil {
			return fmt.Errorf("creating CGI auto retry request: %w", err)
		}
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req2.SetBasicAuth(c.username, c.password)

		resp, err = c.http.Do(req2)
		if err != nil {
			if strings.Contains(err.Error(), "malformed HTTP response") {
				return nil
			}
			return fmt.Errorf("sending CGI auto retry request: %w", err)
		}
		defer resp.Body.Close()
	}

	return nil
}

