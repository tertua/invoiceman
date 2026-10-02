package controllers

import (
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
)

// clientInvoiceRow is one invoice row on the client detail page.
type clientInvoiceRow struct {
	ID              uuid.UUID    `json:"id"`
	InvoiceNumber   string       `json:"invoice_number"`
	IssueDate       string       `json:"issue_date"`
	DueDate         string       `json:"due_date"`
	Total           models.Money `json:"total"`
	Currency        string       `json:"currency"`
	Status          string       `json:"status"`
	EffectiveStatus string       `json:"effective_status"`
	PaidAmount      models.Money `json:"paid_amount"`
	Balance         models.Money `json:"balance"`
}

// clientDetailResponse is the client detail payload with its invoice rows and stats.
type clientDetailResponse struct {
	Client   models.Client      `json:"client"`
	Invoices []clientInvoiceRow `json:"invoices"`
	Stats    models.ClientStats `json:"stats"`
}
