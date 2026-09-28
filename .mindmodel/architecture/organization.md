# Code Organization

## Rules
- File size limits enforced by scripts/file-size-baseline.json (CI check:size)
- Controllers ≤400, queries ≤300, models ≤200, components ≤250, hooks ≤150 lines
- Split strategy: gateway_*_controller.go by route area, flow_*_test.go by domain
- Frontend splits: extract components/* cards from pages, i18n.*.settings.js per language
- Route registration order critical: /api/v1 before /api, public→gateway→private within prefix
- One responsibility per file; growing a capped file triggers split, never baseline raise
- Comments: one-line only except Swagger annotations
- Tests use flow_fixtures_test.go builders (check:fixtures CI enforces)

## Examples

### Route Registration Order
```go
// pkg/routes/versioning.go:21-34
func RegisterAPI(a *fiber.App, prefix string) {
	PublicRoutesAt(a, prefix)
	GatewayRoutesAt(a, prefix)
	PrivateRoutesAt(a, prefix)
}

func DeprecationHeaders() fiber.Handler {
	return func(c fiber.Ctx) error {
		p := c.Path()
		if strings.HasPrefix(p, APILegacyPrefix+"/") && !strings.HasPrefix(p, APIV1Prefix+"/") {
			c.Set("Deprecation", "true")
			c.Set("Sunset", SunsetDate)
		}
		return c.Next()
	}
}
```

### Test Fixture Builder Pattern
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

func (s invoiceSpec) body(t *testing.T) string {
	t.Helper()
```

### File Size Baseline Entry
```json
// scripts/file-size-baseline.json:2-5
{
 "app/controllers/admin_controller.go": 185,
 "app/controllers/ai_controller.go": 249,
 "app/controllers/ai_locale_test.go": 64,
```

## Anti-patterns

### ❌ Growing Past Baseline Cap
```go
// BAD: adding 50 lines to gateway_intent_controller.go (already at 239/400)
// without checking if it should split to gateway_intent_create.go

// GOOD: split when nearing cap, lower baseline entry
```

### ❌ Raw Invoice POST in Tests
```go
// BAD: test builds JSON manually (CI fails check:fixtures)
resp := doRequest(t, app, "POST", "/api/invoices", `{"status":"sent",...}`, cookies)

// GOOD: use fixture builder
inv := createInvoice(t, app, cookies, newInvoice())
```

### ❌ Multi-line Comment (Non-Swagger)
```go
// BAD: multi-line regular comment
// This function does X
// by doing Y
// and then Z
func doSomething() {}

// GOOD: one-liner or Swagger annotation
// doSomething does X by doing Y and then Z.
func doSomething() {}
```
