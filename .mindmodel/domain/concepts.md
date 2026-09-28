# Domain Concepts

## Rules
- Invoice lifecycle: draft → sent → paid (lock points: pending tx or fully paid)
- Effective status: stored status + overlays (overdue if past due_date, pending if transaction exists)
- Payment flows: manual record, gateway relay (API key), public pay (PaymentLink with token)
- Money: models.Money decimal type, wire format as strings
- Gateway relay: downstream projects charge via Tupay, webhook forwarded with HMAC
- Idempotency: money-moving mutations, CLM- prefix for claim slots (concurrent collapse prevention)
- AI features: receipt scan, reminder generation, invoice summaries (locale from X-Locale)
- Multi-language: en/id, backend errors stable English → frontend maps via i18n

## Examples

### Invoice Status Constants
```go
// app/models/invoice_model.go:14-26
const (
	InvoiceStatusDraft = "draft"
	InvoiceStatusSent  = "sent"
	InvoiceStatusPaid  = "paid"
)

const (
	InvoiceEffectiveOverdue = "overdue"
	InvoiceEffectivePending = "pending"
)
```

### Business Rule: Client Required to Send
```go
// app/controllers/invoice_rules.go:11-40
var (
	ErrClientRequiredToSend = errors.New("client is required to send an invoice")
	ErrPendingPayment       = errors.New("invoice has a pending payment")
	ErrInvoicePaid          = errors.New("invoice is already paid")
)

func validateInvoice(status string, clientID *uuid.UUID) error {
	if status == models.InvoiceStatusSent && clientID == nil {
		return ErrClientRequiredToSend
	}
	return nil
}

func resolveClient(input *models.InvoiceInput) (*uuid.UUID, error) {
	var clientID *uuid.UUID
	if input.ClientID != nil && *input.ClientID != "" {
		parsed, err := uuid.Parse(*input.ClientID)
		if err != nil {
			return nil, errors.New("invalid client_id")
		}
		clientID = &parsed
	}
	if err := validateInvoice(input.Status, clientID); err != nil {
		return nil, err
	}
```

### Gateway Transaction Status
```go
// platform/gateway/gateway.go:31-39
const (
	StatusPending         = "pending"
	StatusSuccess         = "success"
	StatusFailed          = "failed"
	StatusExpired         = "expired"
	StatusRefunded        = "refunded"
	StatusPartialRefunded = "partially_refunded"
)
```

## Anti-patterns

### ❌ Updating Paid Invoice
```go
// BAD: allow mutation of paid invoice
if invoice.Status == models.InvoiceStatusPaid {
	// still allow update
	invoice.Total = newTotal
}

// GOOD: reject mutation
if invoice.Status == models.InvoiceStatusPaid {
	return ErrInvoicePaid
}
```

### ❌ Sending Invoice Without Client
```go
// BAD: sent invoice with nil client
invoice.Status = models.InvoiceStatusSent
invoice.ClientID = nil

// GOOD: validate rule
if err := validateInvoice(status, clientID); err != nil {
	return failInvoiceRule(c, err)
}
```
