package notify

import (
	"context"
	"errors"
	"fmt"
)

// MultiNotifier sends notifications to multiple backends.
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMulti creates a MultiNotifier from a list of notifiers.
func NewMulti(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

// Send delivers the notification to all backends, collecting errors.
func (m *MultiNotifier) Send(ctx context.Context, n Notification) error {
	var errs []error
	for _, notifier := range m.notifiers {
		if err := notifier.Send(ctx, n); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("notification delivery failed: %w", errors.Join(errs...))
	}
	return nil
}

// Count returns the number of configured notifiers.
func (m *MultiNotifier) Count() int {
	return len(m.notifiers)
}
