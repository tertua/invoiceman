package nowpayments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/platform/gateway"
)

func TestMinAmountFetchesAndCaches(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/min-amount" || r.Header.Get("x-api-key") != "test-key" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("currency_from") != "usd" || r.URL.Query().Get("currency_to") != DefaultPayCurrency {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"currency_from":"usd","currency_to":"` + DefaultPayCurrency + `","min_amount":18.80979977}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")
	resetMinAmountCache()

	gw := Gateway{}
	got, err := gw.MinAmount(context.Background(), "USD", "")
	if err != nil {
		t.Fatalf("MinAmount: %v", err)
	}
	if !got.Equal(decimal.RequireFromString("18.80979977")) {
		t.Fatalf("min = %s, want 18.80979977", got)
	}
	if _, err := gw.MinAmount(context.Background(), "usd", ""); err != nil {
		t.Fatalf("MinAmount (cached): %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("server calls = %d, want 1 (second read must hit cache)", calls.Load())
	}
}

// The minimum belongs to the asset the payer picked: an empty pay currency
// falls back to the canonical payout, "LTC" asks for the LTC minimum.
func TestMinAmountUsesChosenAsset(t *testing.T) {
	var lastTo atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/min-amount" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		to := r.URL.Query().Get("currency_to")
		lastTo.Store(to)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"currency_from":"usd","currency_to":"` + to + `","min_amount":0.5}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")
	resetMinAmountCache()

	gw := Gateway{}
	got, err := gw.MinAmount(context.Background(), "USD", " LTC ")
	if err != nil {
		t.Fatalf("MinAmount: %v", err)
	}
	if !got.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf("min = %s, want 0.5", got)
	}
	if to := lastTo.Load(); to != "ltc" {
		t.Fatalf("currency_to = %v, want ltc", to)
	}
	if _, err := gw.MinAmount(context.Background(), "USD", ""); err != nil {
		t.Fatalf("MinAmount (default payout): %v", err)
	}
	if to := lastTo.Load(); to != DefaultPayCurrency {
		t.Fatalf("currency_to = %v, want %s", to, DefaultPayCurrency)
	}
}

func TestMinAmountNotConfigured(t *testing.T) {
	t.Setenv("NOWPAYMENTS_API_KEY", "")
	resetMinAmountCache()
	gw := Gateway{}
	if _, err := gw.MinAmount(context.Background(), "USD", ""); err != gateway.ErrNotConfigured {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestMinAmountAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")
	resetMinAmountCache()

	gw := Gateway{}
	if _, err := gw.MinAmount(context.Background(), "IDR", ""); err == nil {
		t.Fatal("expected error for unknown currency, got nil")
	}
}
