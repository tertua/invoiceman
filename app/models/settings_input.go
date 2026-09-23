package models

// SettingsInput describes the user-editable settings payload.
type SettingsInput struct {
	CompanyName   string  `json:"company_name" validate:"lte=255"`
	Email         string  `json:"email" validate:"omitempty,email,lte=255"`
	Phone         string  `json:"phone" validate:"lte=100"`
	Address       string  `json:"address"`
	LogoURL       string  `json:"logo_url"`
	Currency      string  `json:"currency" validate:"required,lte=3"`
	TaxRate       float64 `json:"tax_rate" validate:"gte=0"`
	InvoicePrefix string  `json:"invoice_prefix" validate:"required,lte=20"`
	Language      string  `json:"language" validate:"omitempty,oneof=en id"`
}
