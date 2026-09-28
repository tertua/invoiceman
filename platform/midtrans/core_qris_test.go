package midtrans

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tertua/tupay/platform/gateway"
)

// The Core API charge body must request QRIS settled by the Gopay acquirer —
// the exact combination Snap cannot express through enabled_payments.
func TestQRISCorePayloadUsesGopayAcquirer(t *testing.T) {
	raw, err := qrisCorePayload("PAY-1-abc", 32000)
	if err != nil {
		t.Fatalf("qrisCorePayload: %v", err)
	}
	var body struct {
		PaymentType string `json:"payment_type"`
		QRIS        struct {
			Acquirer string `json:"acquirer"`
		} `json:"qris"`
		Details struct {
			OrderID     string `json:"order_id"`
			GrossAmount int64  `json:"gross_amount"`
		} `json:"transaction_details"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.PaymentType != "qris" || body.QRIS.Acquirer != "gopay" {
		t.Fatalf("payload = %s, want qris/gopay", raw)
	}
	if body.Details.OrderID != "PAY-1-abc" || body.Details.GrossAmount != 32000 {
		t.Fatalf("details = %+v", body.Details)
	}
}

// An on-page QRIS charge returns the QR image URL and an RFC3339 expiry, and
// must not carry a Snap token (the client would otherwise call snap.pay).
func TestCreateQRISChargeMapsQRCode(t *testing.T) {
	var sawPath, sawAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		sawAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"transaction_id":"txn-1","order_id":"PAY-1-abc","payment_type":"qris",
			"transaction_status":"pending","expiry_time":"2026-09-25 10:30:00",
			"qr_string":"000201PAY-1-abc-EMVCO",
			"actions":[
				{"name":"generate-qr-code","method":"GET","url":"https://api.test/qris/txn-1/qr-code"},
				{"name":"deeplink-redirect","method":"GET","url":"https://api.test/qris/txn-1/deeplink"}
			]}`))
	}))
	defer server.Close()

	cfg := Config{ServerKey: "core-key", CoreBase: server.URL}
	res, err := CreateQRISCharge(context.Background(), cfg, "PAY-1-abc", 32000)
	if err != nil {
		t.Fatalf("CreateQRISCharge: %v", err)
	}
	if sawPath != "/" {
		t.Fatalf("path = %q", sawPath)
	}
	if !strings.HasPrefix(sawAuth, "Basic ") {
		t.Fatalf("missing basic auth: %q", sawAuth)
	}
	if res.Token != "" {
		t.Fatalf("Token = %q, want empty (no snap token)", res.Token)
	}
	if res.PaymentURL != "https://api.test/qris/txn-1/qr-code" {
		t.Fatalf("PaymentURL = %q", res.PaymentURL)
	}
	if res.QRString != "000201PAY-1-abc-EMVCO" {
		t.Fatalf("QRString = %q, want the raw EMVCo payload", res.QRString)
	}
	if res.PaymentMethod != gateway.MethodQRIS {
		t.Fatalf("PaymentMethod = %q", res.PaymentMethod)
	}
	if res.ExpiresAt != "2026-09-25T10:30:00+07:00" {
		t.Fatalf("ExpiresAt = %q, want WIB-normalized RFC3339", res.ExpiresAt)
	}
}

// A non-2xx Core API answer is a typed provider error so callers can branch
// with errors.Is/As instead of parsing the message.
func TestCreateQRISChargeProviderErrorTyped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_messages":["bad"]}`))
	}))
	defer server.Close()

	_, err := CreateQRISCharge(context.Background(), Config{ServerKey: "k", CoreBase: server.URL}, "PAY-3-abc", 32000)
	if !errors.Is(err, gateway.ErrProviderStatus) {
		t.Fatalf("expected errors.Is(_, ErrProviderStatus), got: %v", err)
	}
	var pe *gateway.ProviderError
	if !errors.As(err, &pe) || pe.Status != http.StatusBadRequest || pe.Provider != "midtrans" {
		t.Fatalf("expected typed ProviderError 400/midtrans, got: %+v", pe)
	}
}

// A Core API failure falls back to the Snap checkout so the payer can still
// pay; the fallback must yield a Snap token, not an error.
func TestCreateTransactionQRISFallsBackToSnap(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_messages":["boom"]}`))
	}))
	defer core.Close()
	snap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"snap-fallback","redirect_url":"https://snap.test/fb"}`))
	}))
	defer snap.Close()

	t.Setenv("MIDTRANS_SERVER_KEY", "fallback-key")
	t.Setenv("MIDTRANS_CORE_BASE_URL", core.URL)
	t.Setenv("MIDTRANS_SNAP_BASE_URL", snap.URL)

	out, err := (Gateway{}).CreateTransaction(context.Background(), &gateway.CreateTxRequest{
		OrderID: "PAY-2-abc", AmountMinor: 32000, PaymentMethod: gateway.MethodQRIS, DirectQRIS: true,
	})
	if err != nil {
		t.Fatalf("fallback: %v", err)
	}
	if out.Token != "snap-fallback" {
		t.Fatalf("Token = %q, want snap fallback", out.Token)
	}
}
