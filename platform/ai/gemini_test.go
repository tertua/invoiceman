package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// stubTransport returns canned provider responses regardless of URL (the
// client hardcodes the Google endpoint, so tests mock the transport).
type stubTransport func(*http.Request) (*http.Response, error)

func (f stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func testClient(status int, body string) *GeminiClient {
	return &GeminiClient{
		APIKey: "test-key",
		Model:  "test-model",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: stubTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
}

// A 429 must surface as ErrRateLimited so handlers can answer HTTP 429
// (retry shortly) instead of a generic 502.
func TestGenerateRateLimited(t *testing.T) {
	c := testClient(http.StatusTooManyRequests, `{"error":{"message":"quota exceeded"}}`)
	_, err := c.Generate(context.Background(), "hi")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
}

func TestGenerateSuccess(t *testing.T) {
	c := testClient(http.StatusOK, `{"candidates":[{"content":{"parts":[{"text":"  halo  "}]}}]}`)
	got, err := c.Generate(context.Background(), "hi")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if got != "halo" {
		t.Fatalf("expected trimmed text, got %q", got)
	}
}

func TestGenerateServerError(t *testing.T) {
	c := testClient(http.StatusBadGateway, `{"error":{"message":"overload"}}`)
	_, err := c.Generate(context.Background(), "hi")
	if err == nil || errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected generic failure, got: %v", err)
	}
}
