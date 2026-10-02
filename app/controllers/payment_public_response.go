package controllers

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/gateway"
)

// publicPaymentData assembles the payload for the hosted pay page: invoice
// (with payer-invisible fields stripped), branding, gateway config, the
// charge-method list and the can_pay flag.
func publicPaymentData(ctx context.Context, db database.Queries, link models.PaymentLink, payCurrency string) (publicPaymentPageResponse, error) {
	invoice, err := db.GetInvoiceUnscoped(link.InvoiceID)
	if err != nil {
		return publicPaymentPageResponse{}, err
	}
	orgID := invoice.OrgID
	detail, err := invoiceDetail(db, orgID, link.InvoiceID)
	if err != nil {
		return publicPaymentPageResponse{}, err
	}
	settings, err := db.GetSettings(orgID)
	if err != nil {
		return publicPaymentPageResponse{}, err
	}
	paid, err := db.PaidAmount(invoice.ID)
	if err != nil {
		return publicPaymentPageResponse{}, err
	}
	balance := invoice.Total.Sub(paid)
	if balance.IsNegative() {
		balance = decimal.Zero
	}
	// The public list is rebuilt from the payment rows so each method can be
	// relabeled without the provider name — the payer never needs to know it.
	payments, err := db.GetInvoicePayments(invoice.ID)
	if err != nil {
		return publicPaymentPageResponse{}, err
	}
	clean := make([]publicPaymentResponse, 0, len(payments))
	for _, p := range payments {
		clean = append(clean, publicPaymentResponse{
			ID:     p.ID,
			Amount: p.Amount,
			PaidOn: utils.FormatDate(p.PaidOn),
			Method: publicPayMethodLabel(db, p),
		})
	}
	return publicPaymentPageResponse{
		Invoice: publicInvoiceDetailResponse{
			invoiceDetailResponse: detail,
			Payments:              clean,
		},
		Branding: fiber.Map{
			"company_name": settings.CompanyName,
			"logo_url":     settings.LogoURL,
		},
		Gateway: payerConfigFor(gateway.DefaultProvider()),
		Methods: availableChargeMethods(ctx, invoice.Currency, balance, settings.UsdToIdr, settings, payCurrency),
		CanPay:  detail.EffectiveStatus != models.InvoiceStatusPaid,
	}, nil
}

// publicInvoiceDetailResponse strips the payer-invisible fields off the internal
// invoice detail and swaps in relabeled payment rows. The shadowing fields sit
// shallower than the embedded invoiceDetailResponse, so encoding/json keeps only
// theirs: client_id/client_email/payment_link vanish (omitempty, zero value) and
// payments carries the four-key public row — same key set the old hand-built map
// produced after its deletes.
type publicInvoiceDetailResponse struct {
	invoiceDetailResponse
	ClientID    *uuid.UUID              `json:"client_id,omitempty"`
	ClientEmail string                  `json:"client_email,omitempty"`
	PaymentLink *paymentLinkResponse    `json:"payment_link,omitempty"`
	Payments    []publicPaymentResponse `json:"payments"`
}

type publicPaymentResponse struct {
	ID     uuid.UUID    `json:"id"`
	Amount models.Money `json:"amount"`
	PaidOn string       `json:"paid_on"`
	Method string       `json:"method"`
}

type publicInvoiceStatusResponse struct {
	Status  string       `json:"status"`
	Paid    models.Money `json:"paid"`
	Balance models.Money `json:"balance"`
}

// publicPaymentPageResponse is the outer hosted-pay-page payload. Field order
// mirrors the historical hand-built map; branding and gateway stay fiber.Map
// (<= 2-key nested maps).
type publicPaymentPageResponse struct {
	Invoice  publicInvoiceDetailResponse `json:"invoice"`
	Branding fiber.Map                   `json:"branding"`
	Gateway  fiber.Map                   `json:"gateway"`
	Methods  []publicChargeMethod        `json:"methods"`
	CanPay   bool                        `json:"can_pay"`
}

// publicIntentResponse is the relay payload returned by the public pay endpoint.
// provider_token/snap_token are exposed only when the provider declares the
// token is a browser widget token; ExposesBrowserToken guarantees a non-empty
// value, so omitempty is byte-exact.
func publicIntentResponse(t models.GatewayTransaction) publicIntentResponseRow {
	out := publicIntentResponseRow{
		OrderID:     t.OrderID,
		RedirectURL: t.RedirectURL,
		PaymentURL:  t.PaymentURL,
		QRString:    t.QRString,
		Address:     t.Address,
		PayAmount:   t.PayAmount,
		PayCurrency: t.PayCurrency,
		ExpiresAt:   t.ExpiresAt,
	}
	// Expose the token as a widget token only when the provider declares it
	// is one (BrowserSDKProvider): hosted payment-page ids stored in the same
	// column would be handed to the wrong browser SDK otherwise. Both the
	// provider-neutral key (provider_token) and the legacy key (snap_token)
	// carry the same value; the legacy key stays until pay-page clients
	// migrate.
	if gw, err := gateway.Get(t.Gateway); err == nil && gateway.ExposesBrowserToken(gw, t.ProviderToken) {
		out.ProviderToken = t.ProviderToken
		out.SnapToken = t.ProviderToken
	}
	return out
}

// publicIntentResponseRow mirrors the historical hand-built map. Field order =
// the literal order of the original fiber.Map.
type publicIntentResponseRow struct {
	OrderID       string `json:"order_id"`
	RedirectURL   string `json:"redirect_url"`
	PaymentURL    string `json:"payment_url"`
	QRString      string `json:"qr_string"`
	Address       string `json:"address"`
	PayAmount     string `json:"pay_amount"`
	PayCurrency   string `json:"pay_currency"`
	ExpiresAt     string `json:"expires_at"`
	ProviderToken string `json:"provider_token,omitempty"`
	SnapToken     string `json:"snap_token,omitempty"`
}
