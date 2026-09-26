package queries

import (
	"context"
	"math/rand/v2"
	"time"
)

// DoRetryWithContext is DoRetry with cancellation: the backoff wait aborts
// early when ctx is done, so a cancelled request never sleeps past its
// deadline. Delays use exponential backoff with jitter to avoid thundering
// herds on shared faults.
func DoRetryWithContext(ctx context.Context, fn func() error) error {
	_, err := DoRetryValueWithContext(ctx, func() (struct{}, error) { return struct{}{}, fn() })
	return err
}

// DoRetryValueWithContext is DoRetryValue with cancellation and jitter.
func DoRetryValueWithContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	delay := retryBaseDelay
	for attempt := 1; ; attempt++ {
		val, err := fn()
		if err == nil || !Transient(err) || attempt >= retryAttempts {
			return val, err
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(backoffWithJitter(delay)):
		}
		delay *= 2
	}
}

// backoffWithJitter spreads a base delay by up to +50% so concurrent
// retries on the same fault do not wake in lockstep.
func backoffWithJitter(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	return base + time.Duration(rand.Int64N(int64(base)/2+1)) // #nosec G404 -- backoff jitter, not security-sensitive
}
