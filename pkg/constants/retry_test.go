package constants

import (
	"net/http"
	"testing"
	"time"
)

func TestParseRetryAfterSeconds(t *testing.T) {
	d, ok := ParseRetryAfter(http.Header{"Retry-After": {"2"}})
	if !ok || d != 2*time.Second {
		t.Errorf("expected (2s, true), got (%v, %v)", d, ok)
	}
}

func TestParseRetryAfterMissingAndInvalid(t *testing.T) {
	for _, v := range []string{"", "garbage", "-1"} {
		h := http.Header{}
		if v != "" {
			h.Set("Retry-After", v)
		}
		if _, ok := ParseRetryAfter(h); ok {
			t.Errorf("expected false for %q", v)
		}
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	future := time.Now().Add(5 * time.Second).UTC().Format(http.TimeFormat)
	d, ok := ParseRetryAfter(http.Header{"Retry-After": {future}})
	if !ok || d <= 0 || d > 5*time.Second {
		t.Errorf("expected ~5s wait, got (%v, %v)", d, ok)
	}
}

func TestRetryDelayHonorsHeaderExactly(t *testing.T) {
	d := RetryDelayWithHeader(http.Header{"Retry-After": {"5"}}, 1)
	if d != 5*time.Second {
		t.Errorf("header delay must be honored exactly, got %v", d)
	}
}

func TestRetryDelayCapsHeader(t *testing.T) {
	d := RetryDelayWithHeader(http.Header{"Retry-After": {"120"}}, 1)
	if d != RetryAfterCap {
		t.Errorf("expected cap %v, got %v", RetryAfterCap, d)
	}
}

func TestRetryDelayFallbackJittered(t *testing.T) {
	base := 2 * NowpaymentsRetryBase
	d := RetryDelayWithHeader(http.Header{}, 2)
	if d < base || d > base+base/2 {
		t.Errorf("fallback out of [%v, %v]: %v", base, base+base/2, d)
	}
}
