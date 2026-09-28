package controllers

import (
	"testing"

	"github.com/tertua/tupay/app/models"
)

func TestPayMethodLabel(t *testing.T) {
	qrisTxn := models.GatewayTransaction{PaymentMethod: "qris"}
	cryptoTxn := models.GatewayTransaction{PaymentMethod: "crypto", PayCurrency: "usdtbsc"}
	unknownAssetTxn := models.GatewayTransaction{PaymentMethod: "crypto", PayCurrency: "xyzcoin"}
	emptyMethodTxn := models.GatewayTransaction{Gateway: "midtrans"}

	cases := []struct {
		name   string
		method string
		txn    *models.GatewayTransaction
		want   string
	}{
		{"transaction carries the method", "Midtrans", &qrisTxn, "QRIS"},
		{"crypto shows the asset", "NOWPayments", &cryptoTxn, "Crypto · USDT (BEP20)"},
		{"crypto asset outside the mirror list", "NOWPayments", &unknownAssetTxn, "Crypto · XYZCOIN"},
		{"transaction without a method falls back to the stored label", "Midtrans", &emptyMethodTxn, "QRIS"},
		{"legacy midtrans row", "Midtrans", nil, "QRIS"},
		{"legacy nowpayments row", "NOWPayments", nil, "Crypto"},
		{"provider name is matched case-insensitively", "midtrans", nil, "QRIS"},
		{"manual entries pass through", "Cash", nil, "Cash"},
		{"manual bank transfer passes through", "Bank transfer", nil, "Bank transfer"},
		{"empty method stays empty", "", nil, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := payMethodLabel(tc.method, tc.txn)
			if got != tc.want {
				t.Fatalf("payMethodLabel(%q, txn=%v) = %q, want %q", tc.method, tc.txn, got, tc.want)
			}
		})
	}
}

func TestCryptoAssetLabelEmptyCode(t *testing.T) {
	if got := cryptoAssetLabel("  "); got != "Crypto" {
		t.Fatalf("cryptoAssetLabel(blank) = %q, want %q", got, "Crypto")
	}
}
