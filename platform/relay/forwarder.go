package relay

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/tertua/invoiceman/pkg/constants"
)

// ForwardResult is the outcome of one webhook forward attempt.
type ForwardResult struct {
	StatusCode int
	Body       string
}

// Forward POSTs payload with relay signature headers; callers persist the result for audit and retry.
func Forward(ctx context.Context, targetURL, projectSlug, eventID string, payload []byte, secret string) (*ForwardResult, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.WebhookForwardTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", constants.RelayUserAgent())
	req.Header.Set("X-Relay-Signature", SignPayload(payload, secret))
	req.Header.Set("X-Relay-Event-Id", eventID)
	req.Header.Set("X-Project-Slug", projectSlug)

	resp, err := constants.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return &ForwardResult{StatusCode: resp.StatusCode}, err
	}
	body := string(raw[:min(len(raw), 4000)])
	return &ForwardResult{StatusCode: resp.StatusCode, Body: body}, nil
}
