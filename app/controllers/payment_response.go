package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
)

// paymentListResponse maps a list-row to the API shape (with joined invoice
// fields and the computed can_void flag).
func paymentListResponse(row models.PaymentListRow) fiber.Map {
	return fiber.Map{
		"id":               row.PaymentID,
		"invoice_id":       row.InvoiceID,
		"invoice_number":   row.InvoiceNumber,
		"client_name":      row.ClientName,
		"invoice_currency": row.InvoiceCurrency,
		"amount":           row.Amount,
		"method":           row.Method,
		"paid_on":          utils.FormatDate(row.PaidOn),
		"txn_id":           row.TxnID,
		"notes":            row.Notes,
		"can_void":         row.CanVoid(),
	}
}

// paymentItemResponse maps a single payment row to the API shape used by
// CreatePayment / detail views.
func paymentItemResponse(p models.Payment) fiber.Map {
	return fiber.Map{
		"id":         p.ID,
		"invoice_id": p.InvoiceID,
		"amount":     p.Amount,
		"method":     p.Method,
		"paid_on":    utils.FormatDate(p.PaidOn),
		"txn_id":     p.TxnID,
		"notes":      p.Notes,
	}
}

// paymentVoidResponse is the minimal shape returned after a void.
func paymentVoidResponse(p models.Payment) fiber.Map {
	return fiber.Map{
		"id":         p.ID,
		"invoice_id": p.InvoiceID,
		"voided":     true,
	}
}

// resolveVoidReason reads the void reason from the query string first, falling
// back to a JSON body ({reason}) when the query is absent. An empty result
// means "not provided" (callers turn that into 400); a non-nil error means the
// body was present but unparseable.
func resolveVoidReason(c fiber.Ctx) (string, error) {
	if reason := strings.TrimSpace(c.Query("reason")); reason != "" {
		return reason, nil
	}
	if len(c.Body()) == 0 {
		return "", nil
	}
	input := &models.PaymentVoidInput{}
	if err := c.Bind().Body(input); err != nil {
		return "", err
	}
	return strings.TrimSpace(input.Reason), nil
}
