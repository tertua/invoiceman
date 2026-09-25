package nowpayments

import (
	"context"
	"encoding/json"
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
