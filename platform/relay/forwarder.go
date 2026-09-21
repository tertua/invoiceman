package relay

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// ForwardResult is the outcome of one webhook forward attempt.
type ForwardResult struct {
	StatusCode int
	Body       string
}

// Forward POSTs payload to targetURL with relay signature headers.
// Callers persist the result in webhook_deliveries for audit and retry.
func Forward(ctx context.Context, targetURL, projectSlug, eventID string, payload []byte, secret string) (*ForwardResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Relay-Signature", SignPayload(payload, secret))
	req.Header.Set("X-Relay-Event-Id", eventID)
	req.Header.Set("X-Project-Slug", projectSlug)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return &ForwardResult{StatusCode: resp.StatusCode}, err
	}
	body := string(raw)
	if len(body) > 4000 {
		body = body[:4000]
	}
	return &ForwardResult{StatusCode: resp.StatusCode, Body: body}, nil
}
