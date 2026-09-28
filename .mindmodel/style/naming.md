# Naming Conventions

## Rules
- Go: camelCase for unexported, PascalCase for exported
- File names: snake_case.go (invoice_controller.go, flow_fixtures_test.go)
- Frontend: PascalCase for components (InvoiceDocument.jsx), camelCase for hooks (useInvoices.js)
- Database: snake_case for tables/columns (invoice_number, created_at)
- Constants: PascalCase with prefix (InvoiceStatusDraft, APILegacyPrefix)
- Test helpers: lowercase function names with descriptive verbs (newInvoice, createClient)
- Gateway names: lowercase strings ("midtrans", "nowpayments")
- Route params: lowercase with hyphens (/:invoice-id becomes c.Params("invoice-id"))

## Examples

### Controller Function Names
```go
// app/controllers/invoice_controller.go:26
func GetInvoice(c fiber.Ctx) error {
```

### Model Constants
```go
// app/models/invoice_model.go:14-23
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

### Test Fixture Functions
```go
// pkg/routes/flow_fixtures_test.go:37-48
func newInvoice() invoiceSpec {
	return invoiceSpec{
		Status:   models.InvoiceStatusSent,
		Currency: "IDR",
		Issue:    "2026-09-01",
		Due:      "2026-09-30",
		Items:    []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100000"}},
	}
}

func createClient(t *testing.T, app *fiber.App, cookies []*http.Cookie, name string) string {
```

## Anti-patterns

### ❌ Inconsistent File Naming
```go
// BAD: mixed styles
InvoiceController.go
invoice-controller.go

// GOOD: snake_case
invoice_controller.go
```

### ❌ Uppercase Gateway Names
```go
// BAD: case-sensitive mismatch
gateway.Register("Midtrans")

// GOOD: lowercase
gateway.Register("midtrans")
```
