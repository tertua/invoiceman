package midtrans

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

// Midtrans answers an unknown order with HTTP 200 carrying the
// {"status_code":"404","status_message":"Transaction doesn't exist."}
// envelope (verified live against sandbox): no transaction_status, no
// gross_amount. That must surface as ErrOrderNotFound, not a decimal
// decode error, so the reconciler can terminally fail the row.
func TestFetchStatusUnknownOrderEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":"404","status_message":"Transaction doesn't exist.","id":"2a4c6f52-0522-4267-b371-968a13e681e8"}`))
	}))
	defer srv.Close()

	_, err := FetchStatus(context.Background(), Config{ServerKey: "test", CoreBase: srv.URL}, "PAY-NOPE-1")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("unknown order envelope must be ErrOrderNotFound, got %v", err)
	}
}

func TestFetchStatusHTTPNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := FetchStatus(context.Background(), Config{ServerKey: "test", CoreBase: srv.URL}, "PAY-NOPE-2")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("HTTP 404 must be ErrOrderNotFound, got %v", err)
	}
}

func TestFetchStatusSettlement(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v2/PAY-OK-1/status" {
			t.Errorf("unexpected status path %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":"200","order_id":"PAY-OK-1","transaction_status":"settlement","transaction_id":"txn-9","payment_type":"qris","gross_amount":"32000.00"}`))
	}))
	defer srv.Close()

	st, err := FetchStatus(context.Background(), Config{ServerKey: "test", CoreBase: srv.URL}, "PAY-OK-1")
	if err != nil {
		t.Fatalf("settlement fetch failed: %v", err)
	}
	if st.Status != "success" || st.TransactionID != "txn-9" || st.PaymentType != "qris" {
		t.Fatalf("unexpected parsed status: %+v", st)
	}
	if !st.GrossAmount.Equal(decimal.RequireFromString("32000")) {
		t.Fatalf("unexpected gross amount: %s", st.GrossAmount.String())
	}
}
