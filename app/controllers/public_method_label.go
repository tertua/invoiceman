package controllers

import (
	"strings"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
)

// publicPayMethodLabel hides the settlement provider from public payloads.
// The payments row keeps its provider name in the table (owner view); the
// public page only ever sees the neutral method ("QRIS"), the crypto asset
// ("Crypto · USDT (BEP20)"), or the owner's own manual entry ("Cash").
// Rows settle through gateway_transactions (method id + asset), so the
// label prefers the transaction; rows written before that carry the
// provider display name and fall back to a static map (Midtrans is
// QRIS-only, see midtransMethodAllowlist). The label logic is pure so
// tests need no database.
func publicPayMethodLabel(db database.Queries, p models.Payment) string {
	if p.GatewayOrderID != nil && *p.GatewayOrderID != "" {
		if txn, err := db.GetTransaction(*p.GatewayOrderID); err == nil {
			return payMethodLabel(p.Method, &txn)
		}
	}
	return payMethodLabel(p.Method, nil)
}

func payMethodLabel(method string, txn *models.GatewayTransaction) string {
	if txn != nil && txn.PaymentMethod != "" {
		if txn.PaymentMethod == gateway.MethodCrypto {
			return cryptoAssetLabel(txn.PayCurrency)
		}
		return gateway.MethodName(txn.PaymentMethod)
	}
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "midtrans":
		return "QRIS"
	case "nowpayments":
		return "Crypto"
	}
	return method
}

// cryptoAssetLabels mirrors web/src/lib/cryptoAssets.js (display-only
// labels for the platform/nowpayments allowlist codes).
var cryptoAssetLabels = map[string]string{
	"usdttrc20": "USDT (TRC20)",
	"usdterc20": "USDT (ERC20)",
	"usdtbsc":   "USDT (BEP20)",
	"trx":       "TRX",
	"doge":      "DOGE",
	"ltc":       "LTC",
}

func cryptoAssetLabel(code string) string {
	normalized := strings.ToLower(strings.TrimSpace(code))
	if normalized == "" {
		return "Crypto"
	}
	if label, ok := cryptoAssetLabels[normalized]; ok {
		return "Crypto · " + label
	}
	return "Crypto · " + strings.ToUpper(normalized)
}
