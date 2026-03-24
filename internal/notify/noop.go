package notify

import "context"

// NoopNotifier discards all notifications. Used in tests.
type NoopNotifier struct{}

// Send does nothing and returns nil.
func (n *NoopNotifier) Send(_ context.Context, _ Notification) error {
	return nil
}
