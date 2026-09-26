package nowpayments

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateDirectPaymentHonorsRetryAfter(t *testing.T) {
	calls := 0
	var at []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		at = append(at, time.Now())
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`busy`))
			return
		}
		_, _ = w.Write([]byte(`{"payment_id":"pay_ra","pay_address":"TRA","pay_amount":"25.50","pay_currency":"usdttrc20"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	dp, err := CreateDirectPayment(context.Background(), FromEnv(), &DirectPaymentRequest{
		OrderID: "INV-1-ra", PriceAmount: "25.50", PriceCurrency: "USD", PayCurrency: "USDTTRC20",
	})
	if err != nil {
		t.Fatalf("expected retry to recover from 429, got: %v", err)
	}
	if dp.PaymentID != "pay_ra" || calls != 2 {
		t.Fatalf("expected 2 attempts and a payment, got calls=%d payment=%+v", calls, dp)
	}
	if gap := at[1].Sub(at[0]); gap < time.Second {
		t.Fatalf("must wait out Retry-After: 1, gap was %v", gap)
	}
}

func TestCreateDirectPaymentRetryAbortsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`busy`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := CreateDirectPayment(ctx, FromEnv(), &DirectPaymentRequest{
		OrderID: "INV-1-cancel", PriceAmount: "25.50", PriceCurrency: "USD", PayCurrency: "USDTTRC20",
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}
