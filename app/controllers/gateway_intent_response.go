package controllers

import (
	"time"

	"github.com/tertua/tupay/app/models"
)

// intentResponse is the relay / dashboard intent payload. The provider-neutral
// keys (provider_token, provider_txn_id) are the canonical names; the legacy
// keys (snap_token, midtrans_txn_id) carry the same values and stay until
// relay clients migrate. These JSON keys are the only remaining provider-named
// surface and they are kept for backward compatibility, not because a provider
// is special. Deprecation is documented in the route's @Description
// (gateway_intent_controller.go).
func intentResponse(t models.GatewayTransaction) intentResponseRow {
	return intentResponseRow{
		OrderID:         t.OrderID,
		ExternalOrderID: t.ExternalOrderID,
		ProjectSlug:     t.ProjectSlug,
		Gateway:         t.Gateway,
		PaymentMethod:   t.PaymentMethod,
		AmountIDR:       t.AmountIDR,
		AmountDecimal:   t.AmountDecimal,
		Currency:        t.Currency,
		InvoiceCurrency: t.InvoiceCurrency,
		InvoiceAmount:   t.InvoiceAmount,
		UsdToIdr:        t.UsdToIdr,
		Status:          t.Status,
		ProviderToken:   t.ProviderToken,
		SnapToken:       t.ProviderToken,
		RedirectURL:     t.RedirectURL,
		PaymentURL:      t.PaymentURL,
		Address:         t.Address,
		ProviderTxnID:   t.ProviderTxnID,
		MidtransTxnID:   t.ProviderTxnID,
		CreatedAt:       t.CreatedAt,
	}
}

// intentResponseRow mirrors the historical hand-built map. Field order = the
// literal order of the original fiber.Map. snap_token and midtrans_txn_id are
// the deprecated legacy aliases documented on intentResponse.
type intentResponseRow struct {
	OrderID         string       `json:"order_id"`
	ExternalOrderID string       `json:"external_order_id"`
	ProjectSlug     string       `json:"project_slug"`
	Gateway         string       `json:"gateway"`
	PaymentMethod   string       `json:"payment_method"`
	AmountIDR       int64        `json:"amount_idr"`
	AmountDecimal   string       `json:"amount_decimal"`
	Currency        string       `json:"currency"`
	InvoiceCurrency string       `json:"invoice_currency"`
	InvoiceAmount   models.Money `json:"invoice_amount"`
	UsdToIdr        models.Money `json:"usd_to_idr"`
	Status          string       `json:"status"`
	ProviderToken   string       `json:"provider_token"`
	SnapToken       string       `json:"snap_token"`
	RedirectURL     string       `json:"redirect_url"`
	PaymentURL      string       `json:"payment_url"`
	Address         string       `json:"address"`
	ProviderTxnID   string       `json:"provider_txn_id"`
	MidtransTxnID   string       `json:"midtrans_txn_id"`
	CreatedAt       time.Time    `json:"created_at"`
}
