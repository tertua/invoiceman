package models

// GatewayInvoiceInput is the public service-to-service invoice payload.
// Customer and invoice identifiers belong to the calling project, not the
// internal UUIDs used by Invoiceman.
type GatewayInvoiceInput struct {
	ExternalID string               `json:"external_id" validate:"required,lte=128"`
	Status     string               `json:"status" validate:"required,oneof=draft sent"`
	IssueDate  string               `json:"issue_date"`
	DueDate    string               `json:"due_date"`
	Currency   string               `json:"currency" validate:"required,lte=3"`
	TaxRate    float64              `json:"tax_rate" validate:"gte=0"`
	Discount   Money                `json:"discount"`
	Notes      string               `json:"notes"`
	Terms      string               `json:"terms"`
	Customer   GatewayCustomerInput `json:"customer" validate:"required"`
	Items      []InvoiceItemInput   `json:"items" validate:"required,min=1,dive"`
}

type GatewayCustomerInput struct {
	ExternalID string `json:"external_id" validate:"required,lte=128"`
	Name       string `json:"name" validate:"required,lte=255"`
	Email      string `json:"email" validate:"omitempty,email,lte=255"`
	Company    string `json:"company" validate:"lte=255"`
	Phone      string `json:"phone" validate:"lte=100"`
	Address    string `json:"address"`
}
