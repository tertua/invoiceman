package queries

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoRetryWithContextSuccess(t *testing.T) {
	calls := 0
	err := DoRetryWithContext(context.Background(), func() error { calls++; return nil })
	if err != nil || calls != 1 {
		t.Errorf("expected 1 call, no error; got %d calls, %v", calls, err)
	}
}

func TestDoRetryWithContextFlakyThenSuccess(t *testing.T) {
	calls := 0
	flaky := errors.New("connection reset by peer")
	err := DoRetryWithContext(context.Background(), func() error {
		calls++
		if calls < 3 {
			return flaky
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Errorf("expected 3 calls then success; got %d calls, %v", calls, err)
	}
}

func TestDoRetryWithContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := DoRetryWithContext(ctx, func() error { return errors.New("database is locked") })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("must not sleep past cancellation")
	}
}

func TestDoRetryValueWithContextCancelledZero(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	val, err := DoRetryValueWithContext(ctx, func() (string, error) {
		return "", errors.New("connection refused")
	})
	if !errors.Is(err, context.Canceled) || val != "" {
		t.Errorf("expected zero value + context.Canceled, got (%q, %v)", val, err)
	}
}

func TestBackoffWithJitterBounds(t *testing.T) {
	base := 100 * time.Millisecond
	for i := 0; i < 100; i++ {
		d := backoffWithJitter(base)
		if d < base || d > base+base/2 {
			t.Fatalf("jitter out of [%v, %v]: %v", base, base+base/2, d)
		}
	}
	if backoffWithJitter(0) != 0 {
		t.Errorf("expected zero backoff for zero base")
	}
}
