package models

type PaymentReminderInput struct {
	InvoiceID string `json:"invoiceId" validate:"required,uuid"`
	Tone      string `json:"tone" validate:"required,oneof=friendly firm final"`
}

type WriteNoteInput struct {
	Kind   string                   `json:"kind" validate:"required,oneof=description terms"`
	Prompt string                   `json:"prompt" validate:"lte=2000"`
	Items  []map[string]interface{} `json:"items"`
	Client map[string]interface{}   `json:"client"`
}
