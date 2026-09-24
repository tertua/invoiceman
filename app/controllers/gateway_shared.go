package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

func sanitizeExternal(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 48 {
		out = out[:48]
	}
	if out == "" {
		out = "order"
	}
	return out
}

func randHex(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func relayOrderID(slug, external string) (string, error) {
	suffix, err := randHex(4)
	if err != nil {
		return "", err
	}
	return "EXT-" + slug + "-" + sanitizeExternal(external) + "-" + suffix, nil
}

// localOrderID builds the gateway order id for a local invoice. The PAY-
// prefix keeps it distinct from the user-configurable invoice number prefix
// (default INV-), avoiding stutter like INV-INV-000042.
func localOrderID(invoiceNumber, suffix string) string {
	return "PAY-" + strings.ReplaceAll(invoiceNumber, " ", "") + "-" + suffix
}

// legacyLocalOrderID is the pre-rename order id format (INV- prefix). It is
// only used to reuse intents created before the rename instead of
// duplicating them at the gateway.
func legacyLocalOrderID(invoiceNumber, suffix string) string {
	return "INV-" + strings.ReplaceAll(invoiceNumber, " ", "") + "-" + suffix
}

func intentResponse(t models.GatewayTransaction) fiber.Map {
	return fiber.Map{
		"order_id":          t.OrderID,
		"external_order_id": t.ExternalOrderID,
		"project_slug":      t.ProjectSlug,
		"gateway":           t.Gateway,
		"payment_method":    t.PaymentMethod,
		"amount_idr":        t.AmountIDR,
		"amount_decimal":    t.AmountDecimal,
		"currency":          t.Currency,
		"status":            t.Status,
		"snap_token":        t.SnapToken,
		"redirect_url":      t.RedirectURL,
		"payment_url":       t.PaymentURL,
		"address":           t.Address,
	}
}

// resolveGateway picks the provider: explicit request > project default > midtrans.
func resolveGateway(requested, projectDefault string) string {
	if name := strings.ToLower(strings.TrimSpace(requested)); name != "" {
		return name
	}
	if name := strings.ToLower(strings.TrimSpace(projectDefault)); name != "" {
		return name
	}
	return "midtrans"
}
