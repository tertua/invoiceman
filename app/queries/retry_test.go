package queries

import (
	"errors"
	"testing"
)

func TestDoRetrySuccessFirstTry(t *testing.T) {
	calls := 0
	err := DoRetry(func() error { calls++; return nil })
	if err != nil || calls != 1 {
		t.Errorf("expected 1 call, no error; got %d calls, %v", calls, err)
	}
}

func TestDoRetryFlakyThenSuccess(t *testing.T) {
	calls := 0
	flaky := errors.New("connection reset by peer")
	err := DoRetry(func() error {
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

func TestDoRetryPermanentNoRetry(t *testing.T) {
	calls := 0
	permanent := errors.New(`duplicate key value violates unique constraint "users_email_key"`)
	err := DoRetry(func() error { calls++; return permanent })
	if !errors.Is(err, permanent) || calls != 1 {
		t.Errorf("expected immediate permanent error; got %d calls, %v", calls, err)
	}
}

func TestDoRetryExhausts(t *testing.T) {
	calls := 0
	locked := errors.New("database is locked")
	err := DoRetry(func() error { calls++; return locked })
	if !errors.Is(err, locked) || calls != retryAttempts {
		t.Errorf("expected %d calls then error; got %d calls, %v", retryAttempts, calls, err)
	}
}

func TestDoRetryValue(t *testing.T) {
	calls := 0
	val, err := DoRetryValue(func() (bool, error) {
		calls++
		if calls == 1 {
			return false, errors.New("SQLSTATE 40001 serialization failure")
		}
		return true, nil
	})
	if err != nil || !val || calls != 2 {
		t.Errorf("expected (true, nil) after retry; got (%v, %v) in %d calls", val, err, calls)
	}
}

func TestTransientClassification(t *testing.T) {
	transient := []string{
		"database is locked", "connection refused", "i/o timeout",
		"SQLSTATE 40P01 deadlock detected", "too many clients",
	}
	for _, msg := range transient {
		if !Transient(errors.New(msg)) {
			t.Errorf("expected transient: %q", msg)
		}
	}
	permanent := []string{
		"record not found", "duplicate key", "syntax error",
		"violates foreign key constraint", "",
	}
	for _, msg := range permanent {
		var err error
		if msg != "" {
			err = errors.New(msg)
		}
		if Transient(err) {
			t.Errorf("expected permanent: %q", msg)
		}
	}
}
