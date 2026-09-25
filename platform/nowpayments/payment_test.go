package nowpayments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tertua/invoiceman/platform/gateway"
)

func TestCreateDirectPayment(t *testing.T) {
	var gotPath, gotKey string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-api-key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payment_id":"pay_usdt_1","pay_address":"TABC123","pay_amount":"25.50","pay_currency":"usdttrc20","price_amount":"25.50","price_currency":"USD","order_id":"INV-1-abc"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	dp, err := CreateDirectPayment(context.Background(), FromEnv(), &DirectPaymentRequest{
		OrderID: "INV-1-abc", PriceAmount: "25.50", PriceCurrency: "USD", PayCurrency: "USDTTTC20",
	})
	if err != nil {
		t.Fatalf("CreateDirectPayment: %v", err)
	}
	if dp.PaymentID != "pay_usdt_1" || dp.PayAddress != "TABC123" || dp.PayAmount != "25.50" {
		t.Fatalf("unexpected payment: %+v", dp)
	}
	if gotPath != "/v1/payment" || gotKey != "test-key" {
		t.Fatalf("unexpected request: %s key=%s", gotPath, gotKey)
	}
	if gotBody["pay_currency"] != "USDTTTC20" || gotBody["price_currency"] != "USD" {
		t.Fatalf("unexpected body: %v", gotBody)
	}
	if v, ok := gotBody["price_amount"].(float64); !ok || v != 25.5 {
		t.Fatalf("price_amount must be a JSON number, got %T %v", gotBody["price_amount"], gotBody["price_amount"])
	}
}

func TestCreateDirectPaymentAmountBelowMinimum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":false,"statusCode":400,"code":"AMOUNT_MINIMAL_ERROR","message":"Crypto amount 10.485047 is less than minimal"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	_, err := CreateDirectPayment(context.Background(), FromEnv(), &DirectPaymentRequest{
		OrderID: "INV-1-small", PriceAmount: "10.50", PriceCurrency: "USD", PayCurrency: "USDTTRC20",
	})
	if !errors.Is(err, gateway.ErrAmountBelowMinimum) {
		t.Fatalf("expected ErrAmountBelowMinimum, got: %v", err)
	}
}

func TestCreateDirectPaymentOtherError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":false,"statusCode":400,"code":"WRONG_PAYLOAD","message":"bad payload"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	_, err := CreateDirectPayment(context.Background(), FromEnv(), &DirectPaymentRequest{
		OrderID: "INV-1-bad", PriceAmount: "10.50", PriceCurrency: "USD", PayCurrency: "USDTTRC20",
	})
	if err == nil || errors.Is(err, gateway.ErrAmountBelowMinimum) {
		t.Fatalf("expected a generic provider error, got: %v", err)
	}
}

func TestCreateDirectPaymentRetriesRateLimit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`<html>429 Too Many Requests</html>`))
			return
		}
		_, _ = w.Write([]byte(`{"payment_id":"pay_retry","pay_address":"TRETRY","pay_amount":"25.50","pay_currency":"usdttrc20"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	dp, err := CreateDirectPayment(context.Background(), FromEnv(), &DirectPaymentRequest{
		OrderID: "INV-1-retry", PriceAmount: "25.50", PriceCurrency: "USD", PayCurrency: "USDTTRC20",
	})
	if err != nil {
		t.Fatalf("expected the retry to recover from 429, got: %v", err)
	}
	if dp.PaymentID != "pay_retry" || calls != 2 {
		t.Fatalf("expected 2 attempts and a payment, got calls=%d payment=%+v", calls, dp)
	}
}

func TestCreateDirectPaymentRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`<html>429 Too Many Requests</html>`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	_, err := CreateDirectPayment(context.Background(), FromEnv(), &DirectPaymentRequest{
		OrderID: "INV-1-limited", PriceAmount: "25.50", PriceCurrency: "USD", PayCurrency: "USDTTRC20",
	})
	if !errors.Is(err, gateway.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
}

func TestCreateDirectPaymentGateway(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payment_id":"pay_usdt_gw","pay_address":"addr1","pay_amount":"100","pay_currency":"usdttrc20","price_amount":"100","price_currency":"USD","order_id":"INV-1"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	deposit := &gateway.CreateTxRequest{OrderID: "INV-1", AmountDecimal: "100", Currency: "USD", PayCurrency: "USDTTTC20"}
	res, err := (Gateway{}).CreateTransaction(context.Background(), deposit)
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if res.Address != "addr1" || res.Token != "pay_usdt_gw" || res.RawPayload != "100" {
		t.Fatalf("unexpected response: %+v", res)
	}
	if res.PaymentURL != "" {
		t.Fatalf("expected no hosted payment_url for direct payment, got %q", res.PaymentURL)
	}
}
