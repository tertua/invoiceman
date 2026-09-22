package queries

import (
	"strings"
	"time"
)

// Retry policy for transient database faults: 3 attempts, 50ms doubling.
// Delays stay small because this wraps request-path queries; the outbox
// worker has its own (much longer) job-level backoff.
const (
	retryAttempts  = 3
	retryBaseDelay = 50 * time.Millisecond
)

// DoRetry runs fn, retrying transient failures (lost connections, timeouts,
// SQLite locks, PG serialization/deadlock) with exponential backoff.
// Permanent errors (constraints, not-found, syntax) return immediately.
// It is safe to wrap whole GORM transactions: GORM rolls the transaction
// back before the error surfaces, so a retry re-executes cleanly.
func DoRetry(fn func() error) error {
	_, err := DoRetryValue(func() (struct{}, error) { return struct{}{}, fn() })
	return err
}

// DoRetryValue is DoRetry for functions returning a value.
func DoRetryValue[T any](fn func() (T, error)) (T, error) {
	delay := retryBaseDelay
	for attempt := 1; ; attempt++ {
		val, err := fn()
		if err == nil || !Transient(err) || attempt >= retryAttempts {
			return val, err
		}
		time.Sleep(delay)
		delay *= 2
	}
}

// Transient reports whether err looks like a recoverable transport, lock,
// or concurrency fault. Matching is message-based and backend-agnostic on
// purpose: this package must not import database drivers (SQLite vs
// PostgreSQL is decided in platform/database).
func Transient(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, marker := range []string{
		// SQLite locks (WAL concurrency, busy handlers exhausted).
		"database is locked", "database table is locked",
		"database schema is locked", "database is busy",
		// Lost/broken connections, DNS, timeouts (both drivers).
		"connection refused", "connection reset", "connection timed out",
		"broken pipe", "bad connection", "connection failed",
		"server closed the connection", "no such host",
		"network is unreachable", "i/o timeout", "deadline exceeded",
		"too many clients", "too many connections",
		// PostgreSQL serialization/deadlock/resource codes.
		"sqlstate 40001", "sqlstate 40p01",
		"sqlstate 53300", "sqlstate 57p01", "sqlstate 57p02",
	} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}
