package api

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
)

// tokenToucher debounces last_used updates for auth tokens. The auth path
// calls Mark(id) on every authenticated request — that is O(1) and lock-only.
// A background goroutine, started by Run, flushes the accumulated set on a
// timer with a single bulk UPDATE.
//
// The previous implementation spawned a new goroutine per authenticated
// request and used context.Background(), which under load created unbounded
// goroutines and detached the writes from the request lifecycle.
type tokenToucher struct {
	sqlite   *db.SQLiteDB
	logger   *slog.Logger
	interval time.Duration

	mu    sync.Mutex
	dirty map[int64]struct{}

	done chan struct{}
}

func newTokenToucher(sqlite *db.SQLiteDB, logger *slog.Logger, interval time.Duration) *tokenToucher {
	return &tokenToucher{
		sqlite:   sqlite,
		logger:   logger,
		interval: interval,
		dirty:    make(map[int64]struct{}),
		done:     make(chan struct{}),
	}
}

// Mark records that a token was just used. Safe to call from any goroutine.
func (t *tokenToucher) Mark(id int64) {
	t.mu.Lock()
	t.dirty[id] = struct{}{}
	t.mu.Unlock()
}

// Run flushes the dirty set on a timer until ctx is cancelled, then performs
// one final flush before exiting.
func (t *tokenToucher) Run(ctx context.Context) {
	defer close(t.done)
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.flush(context.Background())
			return
		case <-ticker.C:
			t.flush(ctx)
		}
	}
}

// Done blocks until the run loop has exited (used for graceful shutdown).
func (t *tokenToucher) Done() <-chan struct{} { return t.done }

func (t *tokenToucher) flush(ctx context.Context) {
	t.mu.Lock()
	if len(t.dirty) == 0 {
		t.mu.Unlock()
		return
	}
	ids := make([]int64, 0, len(t.dirty))
	for id := range t.dirty {
		ids = append(ids, id)
	}
	t.dirty = make(map[int64]struct{})
	t.mu.Unlock()

	if err := bulkTouchTokens(ctx, t.sqlite, ids); err != nil {
		t.logger.Warn("token last_used flush failed; will retry next interval",
			"err", err, "count", len(ids))
		// Re-queue so the next tick retries; we don't want to silently drop
		// last_used updates because of a transient lock contention.
		t.mu.Lock()
		for _, id := range ids {
			t.dirty[id] = struct{}{}
		}
		t.mu.Unlock()
	}
}

// bulkTouchTokens issues a single UPDATE against the supplied ids. SQLite has
// a default parameter limit (999) we don't get near here — token counts are
// in the dozens at most.
func bulkTouchTokens(ctx context.Context, sqlite *db.SQLiteDB, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	_, err := sqlite.DB().ExecContext(ctx,
		"UPDATE auth_tokens SET last_used = CURRENT_TIMESTAMP WHERE id IN ("+placeholders+")",
		args...,
	)
	if err != nil {
		return fmt.Errorf("bulk touch tokens: %w", err)
	}
	return nil
}
