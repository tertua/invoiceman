package gateway

import (
	"errors"
	"fmt"
)

// ErrProviderStatus is the sentinel wrapped by every non-2xx provider
// answer, so callers branch with errors.Is instead of matching provider
// message strings (which drift between providers and drivers).
var ErrProviderStatus = errors.New("payment gateway returned an error status")

// ProviderError is one failed HTTP response from a provider. Body is already
// truncated by the caller for logging; the raw provider payload is never
// returned to end users. errors.As recovers the status code, errors.Is(err,
// ErrProviderStatus) matches any provider failure.
type ProviderError struct {
	Provider string
	Status   int
	Body     string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s: status %d: %s", e.Provider, e.Status, e.Body)
}

// Unwrap makes errors.Is(err, ErrProviderStatus) true for every provider
// HTTP failure.
func (e *ProviderError) Unwrap() error { return ErrProviderStatus }

// NewProviderError builds a typed provider status error.
func NewProviderError(provider string, status int, body string) *ProviderError {
	return &ProviderError{Provider: provider, Status: status, Body: body}
}
