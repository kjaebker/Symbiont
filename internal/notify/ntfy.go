package notify

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NtfyNotifier sends notifications via ntfy.sh.
type NtfyNotifier struct {
	topicURL string
	client   *http.Client
}

// NewNtfy creates a new ntfy.sh notifier for the given topic URL.
func NewNtfy(topicURL string) *NtfyNotifier {
	return &NtfyNotifier{
		topicURL: topicURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send delivers a notification to the ntfy.sh topic.
// It retries once on transient failures (5xx, network error).
func (n *NtfyNotifier) Send(ctx context.Context, notif Notification) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("ntfy send cancelled during retry: %w", ctx.Err())
			case <-time.After(5 * time.Second):
			}
		}

		err := n.doSend(ctx, notif)
		if err == nil {
			return nil
		}
		lastErr = err

		// Only retry on transient errors (5xx or network).
		if isClientError(err) {
			return err
		}
	}
	return fmt.Errorf("ntfy send failed after retry: %w", lastErr)
}

func (n *NtfyNotifier) doSend(ctx context.Context, notif Notification) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.topicURL, strings.NewReader(notif.Body))
	if err != nil {
		return fmt.Errorf("creating ntfy request: %w", err)
	}

	req.Header.Set("Title", notif.Title)
	if notif.Priority != "" {
		req.Header.Set("Priority", notif.Priority)
	}
	if len(notif.Tags) > 0 {
		req.Header.Set("Tags", strings.Join(notif.Tags, ","))
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &ntfyError{StatusCode: resp.StatusCode}
	}
	return nil
}

// ntfyError represents an HTTP error from the ntfy server.
type ntfyError struct {
	StatusCode int
}

func (e *ntfyError) Error() string {
	return fmt.Sprintf("ntfy returned status %d", e.StatusCode)
}

// isClientError returns true if the error is a 4xx client error (not transient).
func isClientError(err error) bool {
	if e, ok := err.(*ntfyError); ok {
		return e.StatusCode >= 400 && e.StatusCode < 500
	}
	return false
}
