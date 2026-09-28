package constants

import (
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Retry policy for provider 429 answers. postWithRetry honors the
// Retry-After header (seconds or HTTP-date) capped below, falling back to
// linear backoff with jitter so concurrent callers do not wake in lockstep.
const (
	NowpaymentsMaxAttempts = 3
	NowpaymentsRetryBase   = time.Second
	NowpaymentsRetryMax    = 30 * time.Second
	RetryAfterCap          = 30 * time.Second

	MaxRelayResponseSize = 64 << 10
	MaxRelayBodyLog      = 4000
)

// TruncateLog caps a logged response body or error string at MaxRelayBodyLog, so the forwarder, the outbox worker and the webhook controller store the same shape.
func TruncateLog(s string) string {
	if len(s) > MaxRelayBodyLog {
		return s[:MaxRelayBodyLog]
	}
	return s
}

// ParseRetryAfter extracts the provider's requested wait from a
// Retry-After header value (delay seconds or HTTP-date). It reports false
// when the header is missing or unparsable so callers fall back to backoff.
func ParseRetryAfter(header http.Header) (time.Duration, bool) {
	raw := strings.TrimSpace(header.Get("Retry-After"))
	if raw == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if at, err := http.ParseTime(raw); err == nil {
		d := time.Until(at)
		if d < 0 {
			return 0, false
		}
		return d, true
	}
	return 0, false
}

// RetryDelayWithHeader returns the wait before the next attempt: the
// Retry-After request when present (capped, honored exactly), otherwise
// linear backoff with jitter so concurrent callers do not wake in lockstep.
func RetryDelayWithHeader(header http.Header, attempt int) time.Duration {
	if d, ok := ParseRetryAfter(header); ok {
		if d > RetryAfterCap {
			return RetryAfterCap
		}
		return d
	}
	d := time.Duration(attempt) * NowpaymentsRetryBase
	if d > NowpaymentsRetryMax {
		d = NowpaymentsRetryMax
	}
	return d + time.Duration(rand.Int64N(int64(d)/2+1)) // #nosec G404 -- backoff jitter, not security-sensitive
}
