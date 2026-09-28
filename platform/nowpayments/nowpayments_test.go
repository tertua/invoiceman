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

func TestMapStatus(t *testing.T) {
	cases := map[string]string{
		"finished":       gateway.StatusSuccess,
		"failed":         gateway.StatusFailed,
		"expired":        gateway.StatusExpired,
		"refunded":       gateway.StatusRefunded,
		"waiting":        gateway.StatusPending,
		"confirming":     gateway.StatusPending,
		"confirmed":      gateway.StatusPending,
		"sending":        gateway.StatusPending,
		"partially_paid": gateway.StatusPending,
		"unknown-xyz":    gateway.StatusPending,
	}
	for in, want := range cases {
		if got := MapStatus(in); got != want {
			t.Fatalf("MapStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCreateInvoice(t *testing.T) {
	var gotPath, gotKey string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-api-key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"inv_1","invoice_url":"https://nowpayments.io/payment/?iid=1"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	inv, err := CreateInvoice(context.Background(), FromEnv(), &gateway.CreateTxRequest{
		OrderID:       "INV-1-abc",
		AmountDecimal: "25.50",
		Currency:      "usd",
	})
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	if inv.ID != "inv_1" || inv.InvoiceURL != "https://nowpayments.io/payment/?iid=1" {
		t.Fatalf("unexpected invoice: %+v", inv)
	}
	if gotPath != "/v1/invoice" || gotKey != "test-key" {
		t.Fatalf("unexpected request: %s key=%s", gotPath, gotKey)
	}
	if gotBody["order_id"] != "INV-1-abc" || gotBody["price_currency"] != "USD" {
		t.Fatalf("unexpected body: %v", gotBody)
	}
	if _, ok := gotBody["ipn_callback_url"].(string); !ok {
		t.Fatalf("missing ipn_callback_url: %v", gotBody)
	}
}

func TestCreateInvoiceNotConfigured(t *testing.T) {
	t.Setenv("NOWPAYMENTS_API_KEY", "")
	_, err := CreateInvoice(context.Background(), FromEnv(), &gateway.CreateTxRequest{OrderID: "x", AmountMinor: 1000})
	if !errors.Is(err, gateway.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestCreateInvoiceAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid api key"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "bad-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	_, err := CreateInvoice(context.Background(), FromEnv(), &gateway.CreateTxRequest{OrderID: "x", AmountMinor: 1000})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestVerifyNotification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment/pay_1" || r.Header.Get("x-api-key") != "test-key" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"payment_id": "pay_1", "order_id": "INV-1-abc",
			"payment_status": "finished", "pay_address": "addr1",
			"price_amount": 150000, "price_currency": "idr",
			"pay_amount": 0.00025, "pay_currency": "btc"
		}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	res, err := VerifyNotification(context.Background(), FromEnv(), []byte(`{
		"payment_id": "pay_1", "order_id": "INV-1-abc", "payment_status": "finished"
	}`))
	if err != nil {
		t.Fatalf("VerifyNotification: %v", err)
	}
	if res.OrderID != "INV-1-abc" || res.TransactionID != "pay_1" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Status != gateway.StatusSuccess {
		t.Fatalf("status = %q, want success", res.Status)
	}
	if res.GrossMinor != 150000 {
		t.Fatalf("GrossMinor = %d, want 150000", res.GrossMinor)
	}
	if res.Currency != "BTC" || res.GrossDecimal == "" {
		t.Fatalf("unexpected crypto fields: %+v", res)
	}
}

func TestVerifyNotificationOrderMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payment_id": "pay_1", "order_id": "OTHER", "payment_status": "finished"}`))
	}))
	defer srv.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", srv.URL+"/v1")

	_, err := VerifyNotification(context.Background(), FromEnv(), []byte(`{
		"payment_id": "pay_1", "order_id": "INV-1-abc", "payment_status": "finished"
	}`))
	if !errors.Is(err, gateway.ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}
}

func TestVerifyNotificationNotConfigured(t *testing.T) {
	t.Setenv("NOWPAYMENTS_API_KEY", "")
	_, err := VerifyNotification(context.Background(), FromEnv(), []byte(`{}`))
	if !errors.Is(err, gateway.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestVerifyNotificationBadJSON(t *testing.T) {
	t.Setenv("NOWPAYMENTS_API_KEY", "test-key")
	_, err := VerifyNotification(context.Background(), FromEnv(), []byte(`not-json`))
	if !errors.Is(err, gateway.ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}
}

func TestGatewayInterface(t *testing.T) {
	_ = gateway.Gateway(Gateway{})
	if (Gateway{}).Name() != GatewayName {
		t.Fatalf("Name = %q, want %q", (Gateway{}).Name(), GatewayName)
	}
}
