# Testing Patterns

## Rules
- Default test DB: in-memory SQLite (fast, no external services)
- Opt-in integration: INVOICEMAN_TEST_PG_DSN and INVOICEMAN_TEST_REDIS_ADDR env vars
- Flow tests MUST use flow_fixtures_test.go builders (check:fixtures CI enforces)
- Test fixtures: newInvoice(), createInvoice(), createClient() (never raw JSON POST)
- Each test gets newTestApp() with fresh in-memory DB
- go test ./... runs full suite with coverage (no external deps needed)
- Focus tests: go test ./pkg/routes -run TestName -v
- Makefile test target: clean + gocritic + gosec + golangci-lint + coverage

## Examples

### Test Fixture Builder Pattern
```go
// pkg/routes/flow_fixtures_test.go:14-48
type invoiceLine struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Rate        string  `json:"rate"`
}

type invoiceSpec struct {
	Status        string        `json:"status"`
	Currency      string        `json:"currency"`
	ClientID      string        `json:"client_id,omitempty"`
	Issue         string        `json:"issue_date"`
	Due           string        `json:"due_date"`
	Items         []invoiceLine `json:"items"`
}

func newInvoice() invoiceSpec {
	return invoiceSpec{
		Status:   models.InvoiceStatusSent,
		Currency: "IDR",
		Issue:    "2026-09-01",
		Due:      "2026-09-30",
		Items:    []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100000"}},
	}
}

func createInvoice(t *testing.T, app *fiber.App, cookies []*http.Cookie, spec invoiceSpec) map[string]interface{} {
	t.Helper()
	resp := doRequest(t, app, "POST", "/api/invoices", spec.body(t), cookies)
	require.Equal(t, 201, resp.StatusCode, "createInvoice: %s", spec.body(t))
	return decodeBody(t, resp)["invoice"].(map[string]interface{})
}
```

### Create Client Helper
```go
// pkg/routes/flow_fixtures_test.go:71-77
func createClient(t *testing.T, app *fiber.App, cookies []*http.Cookie, name string) string {
	t.Helper()
	resp := doRequest(t, app, "POST", "/api/clients", `{"name":"`+name+`"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	return decodeBody(t, resp)["client"].(map[string]interface{})["id"].(string)
}
```

### AI Locale Test (Unit Test Pattern)
```go
// app/controllers/ai_locale_test.go (example structure)
func TestResolveLocale(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"id", "id"},
		{"ID", "id"},
		{"en", "en"},
		{"", "en"},
		{"unknown", "en"},
	}
	for _, tt := range tests {
		got := resolveLocale(tt.header)
		if got != tt.want {
			t.Errorf("resolveLocale(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}
```

## Anti-patterns

### ❌ Raw JSON POST in Tests
```go
// BAD: manual JSON construction (CI fails check:fixtures)
resp := doRequest(t, app, "POST", "/api/invoices", 
	`{"status":"sent","currency":"IDR","issue_date":"2026-09-01"}`, cookies)

// GOOD: fixture builder
inv := createInvoice(t, app, cookies, newInvoice())
```

### ❌ Hardcoded Test Data
```go
// BAD: magic values scattered across tests
clientID := "550e8400-e29b-41d4-a716-446655440000"

// GOOD: fixture creates and returns ID
clientID := createClient(t, app, cookies, "Test Client")
```
