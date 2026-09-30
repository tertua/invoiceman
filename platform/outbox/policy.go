package outbox

import (
	"time"

	"github.com/tertua/tupay/pkg/configs"
)

// Retry policy for outbox jobs: exponential backoff 1m, 2m, 4m, ... capped at
// OUTBOX_MAX_BACKOFF_MINUTES, dead after OUTBOX_MAX_ATTEMPTS. Both come from
// config so a busy deployment can widen the window without a rebuild.

// Backoff returns the delay before attempt n (1-based).
func Backoff(n int) time.Duration {
	maxBackoff := configs.Get().Outbox.MaxBackoff()
	d := time.Minute << (n - 1)
	if d <= 0 || d > maxBackoff {
		return maxBackoff
	}
	return d
}

// nextRetryAt returns nil when attempts are exhausted (row goes dead).
func nextRetryAt(attempt int, now time.Time) *time.Time {
	if attempt >= configs.Get().Outbox.MaxAttempts {
		return nil
	}
	at := now.Add(Backoff(attempt))
	return &at
}
