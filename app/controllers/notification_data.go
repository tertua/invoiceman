package controllers

import (
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// invoiceNotifData builds the data block shared by invoice.* events; orgID scopes both lookups.
func invoiceNotifData(db *database.Queries, orgID, invoiceID uuid.UUID) invoiceNotifDataResponse {
	out := invoiceNotifDataResponse{InvoiceID: invoiceID.String()}
	invoice, err := db.GetInvoice(orgID, invoiceID)
	if err != nil {
		return out
	}
	out.InvoiceNumber = &invoice.InvoiceNumber
	out.Status = &invoice.Status
	out.IssueDate = datePtr(utils.FormatDate(invoice.IssueDate))
	out.DueDate = datePtr(utils.FormatDate(invoice.DueDate))
	out.Currency = &invoice.Currency
	out.Total = &invoice.Total
	if invoice.ClientID != nil {
		if client, err := db.GetClient(orgID, *invoice.ClientID); err == nil {
			out.ClientName = &client.Name
			out.ClientCompany = &client.Company
			out.ClientEmail = &client.Email
			out.ClientPhone = &client.Phone
		}
	}
	return out
}

// invoiceNotifDataResponse is the wire shape of an invoice.* event's data block.
// invoice_id always rides along; the six loaded fields and the four client
// fields are omitted entirely when the invoice load fails, while issue_date and
// due_date may legitimately be present-and-empty ("") — hence the pointer types
// distinguish "absent" from "present as empty".
type invoiceNotifDataResponse struct {
	InvoiceID     string        `json:"invoice_id"`
	InvoiceNumber *string       `json:"invoice_number,omitempty"`
	Status        *string       `json:"status,omitempty"`
	IssueDate     *string       `json:"issue_date,omitempty"`
	DueDate       *string       `json:"due_date,omitempty"`
	Currency      *string       `json:"currency,omitempty"`
	Total         *models.Money `json:"total,omitempty"`
	ClientName    *string       `json:"client_name,omitempty"`
	ClientCompany *string       `json:"client_company,omitempty"`
	ClientEmail   *string       `json:"client_email,omitempty"`
	ClientPhone   *string       `json:"client_phone,omitempty"`
}

// paymentNotifData builds the data block for payment.created events. The five
// base keys are always present; the embedded invoice fields add the remaining
// invoice.* keys — the outer InvoiceID shadows the embedded one (same value) so
// the merged key set matches the historical hand-built map.
func paymentNotifData(db *database.Queries, payment models.Payment) paymentNotifDataResponse {
	return paymentNotifDataResponse{
		PaymentID:                payment.ID.String(),
		InvoiceID:                payment.InvoiceID.String(),
		Amount:                   payment.Amount,
		Method:                   payment.Method,
		PaidOn:                   utils.FormatDate(payment.PaidOn),
		invoiceNotifDataResponse: invoiceNotifData(db, payment.OrgID, payment.InvoiceID),
	}
}

type paymentNotifDataResponse struct {
	PaymentID string       `json:"payment_id"`
	InvoiceID string       `json:"invoice_id"`
	Amount    models.Money `json:"amount"`
	Method    string       `json:"method"`
	PaidOn    string       `json:"paid_on"`
	invoiceNotifDataResponse
	VoidReason string `json:"void_reason,omitempty"`
}

// datePtr returns a pointer to s so an always-loaded-but-possibly-empty date
// survives omitempty (a non-nil pointer is never omitted).
func datePtr(s string) *string {
	return &s
}
