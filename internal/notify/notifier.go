package notify

import "context"

// Notification holds the content for a notification delivery.
type Notification struct {
	Title    string
	Body     string
	Priority string   // "default", "high", "urgent"
	Tags     []string // e.g. "warning", "thermometer"
}

// Notifier is the interface for notification delivery backends.
type Notifier interface {
	Send(ctx context.Context, n Notification) error
}
