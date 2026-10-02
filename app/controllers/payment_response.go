package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
)

// paymentListResponse maps a list-row to the API shape (with joined invoice
// fields and the computed can_void flag).
func paymentListResponse(row models.PaymentListRow) paymentListResponseRow {
	return paymentListResponseRow{
		ID:              row.PaymentID,
		InvoiceID:       row.InvoiceID,
		InvoiceNumber:   row.InvoiceNumber,
		ClientName:      row.ClientName,
		InvoiceCurrency: row.InvoiceCurrency,
		Amount:          row.Amount,
		Method:          row.Method,
		PaidOn:          utils.FormatDate(row.PaidOn),
		TxnID:           row.TxnID,
		Notes:           row.Notes,
		CanVoid:         row.CanVoid(),
	}
}

// paymentListResponseRow mirrors the historical hand-built map; field order =
// the literal order of the original fiber.Map.
type paymentListResponseRow struct {
	ID              uuid.UUID    `json:"id"`
	InvoiceID       uuid.UUID    `json:"invoice_id"`
	InvoiceNumber   string       `json:"invoice_number"`
	ClientName      string       `json:"client_name"`
	InvoiceCurrency string       `json:"invoice_currency"`
	Amount          models.Money `json:"amount"`
	Method          string       `json:"method"`
	PaidOn          string       `json:"paid_on"`
	TxnID           string       `json:"txn_id"`
	Notes           string       `json:"notes"`
	CanVoid         bool         `json:"can_void"`
}

// paymentListPayload is the ListPayments envelope: the rows plus totals and
// paging meta; totals stays a two-key map shared with other list views.
type paymentListPayload struct {
	Payments []paymentListResponseRow `json:"payments"`
	Totals   fiber.Map                `json:"totals"`
	Meta     fiber.Map                `json:"meta"`
}

// paymentItemResponse maps a single payment row to the API shape used by
// CreatePayment / detail views.
func paymentItemResponse(p models.Payment) paymentItemResponseRow {
	return paymentItemResponseRow{
		ID:        p.ID,
		InvoiceID: p.InvoiceID,
		Amount:    p.Amount,
		Method:    p.Method,
		PaidOn:    utils.FormatDate(p.PaidOn),
		TxnID:     p.TxnID,
		Notes:     p.Notes,
	}
}

type paymentItemResponseRow struct {
	ID        uuid.UUID    `json:"id"`
	InvoiceID uuid.UUID    `json:"invoice_id"`
	Amount    models.Money `json:"amount"`
	Method    string       `json:"method"`
	PaidOn    string       `json:"paid_on"`
	TxnID     string       `json:"txn_id"`
	Notes     string       `json:"notes"`
}

// paymentVoidResponse is the minimal shape returned after a void.
func paymentVoidResponse(p models.Payment) paymentVoidResponseRow {
	return paymentVoidResponseRow{
		ID:        p.ID,
		InvoiceID: p.InvoiceID,
		Voided:    true,
	}
}

type paymentVoidResponseRow struct {
	ID        uuid.UUID `json:"id"`
	InvoiceID uuid.UUID `json:"invoice_id"`
	Voided    bool      `json:"voided"`
}

// paymentAuditMeta is the audit-record metadata for a void, marshaled into the
// audit log detail column. Field order = the original fiber.Map literal.
type paymentAuditMeta struct {
	InvoiceID string       `json:"invoice_id"`
	Amount    models.Money `json:"amount"`
	Reason    string       `json:"reason"`
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
