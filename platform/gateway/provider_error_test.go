package gateway

import (
	"errors"
	"fmt"
	"testing"
)

func TestProviderErrorIsSentinel(t *testing.T) {
	err := NewProviderError("midtrans", 502, "upstream boom")
	if !errors.Is(err, ErrProviderStatus) {
		t.Fatalf("expected errors.Is(err, ErrProviderStatus)")
	}
}

func TestProviderErrorWrappedKeepsSentinel(t *testing.T) {
	err := fmt.Errorf("midtrans snap: %w", NewProviderError("midtrans", 400, "bad request"))
	if !errors.Is(err, ErrProviderStatus) {
		t.Fatalf("wrapped error must still match ErrProviderStatus")
	}
}

func TestProviderErrorAsRecoversStatus(t *testing.T) {
	wrapped := fmt.Errorf("nowpayments payment: %w", NewProviderError("nowpayments", 418, "teapot"))
	var pe *ProviderError
	if !errors.As(wrapped, &pe) {
		t.Fatalf("expected errors.As to recover *ProviderError")
	}
	if pe.Provider != "nowpayments" || pe.Status != 418 || pe.Body != "teapot" {
		t.Fatalf("unexpected recovered error: %+v", pe)
	}
}

func TestProviderErrorMessage(t *testing.T) {
	got := NewProviderError("midtrans", 500, "oops").Error()
	want := "midtrans: status 500: oops"
	if got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
}
