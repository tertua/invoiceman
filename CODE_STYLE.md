# Code Style Guide

Panduan style code untuk TuPay berdasarkan observasi codebase existing. Dokumentasi ini menjelaskan konvensi yang **sudah digunakan**, bukan yang seharusnya.

## Naming Conventions

### Go Backend

**Files**
- Controllers: `{entity}_{type}.go` (e.g., `invoice_controller.go`, `invoice_build.go`)
- Queries: `{entity}_query.go` (e.g., `invoice_query.go`, `client_query.go`)
- Models: `{entity}_model.go` (e.g., `invoice_model.go`, `payment_model.go`)
- Tests: `{subject}_test.go` atau `flow_{feature}_test.go`
- Helpers: `{domain}_helper.go` (e.g., `audit_helper.go`, `cache_helper.go`)

**Functions & Methods**
- Exported (public): `PascalCase` (e.g., `GetInvoice`, `CreateInvoice`)
- Unexported (internal): `camelCase` (e.g., `invoiceDetail`, `buildInvoice`, `autoOnlinePaymentLink`)
- Method receivers: `func (q *InvoiceQueries) ListInvoices(...)`
- Test helpers: `func newInvoice() invoiceSpec`, `func createInvoice(t *testing.T, ...)`

**Variables**
- Local variables: `camelCase` (e.g., `userID`, `invoiceID`, `paidOn`)
- Struct fields (exported): `PascalCase` dengan JSON/GORM tags
  ```go
  type Invoice struct {
      ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
      CreatedAt time.Time `json:"created_at"`
      UserID    uuid.UUID `gorm:"type:uuid" json:"user_id"`
  }
  ```
- JSON tags: `snake_case` (e.g., `json:"invoice_id"`, `json:"created_at"`)

**Constants**
- PascalCase dengan namespace prefix:
  ```go
  const (
      InvoiceStatusDraft = "draft"
      InvoiceStatusSent  = "sent"
      InvoiceStatusPaid  = "paid"
  )
  
  const (
      DefaultPageSize = 20
      MaxPageSize     = 100
  )
  ```
- Gateway names: `const GatewayName = "nowpayments"`

**Packages**
- Lowercase, single word atau compound tanpa underscore
- `package middleware`, `package gateway`, `package nowpayments`

### JavaScript/TypeScript Frontend

**Files**
- Hooks: `use{Entity}.js` (e.g., `useInvoices.js`, `usePayments.js`)
- API modules: `{entity}.js` (e.g., `invoices.js`, `payments.js`, `adminGateway.js`)
- Components: `{Entity}{Type}.jsx` (e.g., `InvoicePaymentCard.jsx`, `RecordPaymentModal.jsx`)
- Pages: `{Entity}.jsx` (e.g., `Dashboard.jsx`, `Invoices.jsx`, `ClientDetail.jsx`)
- Libraries: `{purpose}.js` (e.g., `chartData.js`, `paymentLabels.js`, `utils.js`)

**Functions & Components**
- Components: `PascalCase`, named export
  ```javascript
  export function InvoicePaymentCard({ invoice }) { ... }
  export function RecordPaymentModal({ open, onClose }) { ... }
  ```
- Hooks: `camelCase`, named export
  ```javascript
  export function useInvoices() { ... }
  export function useCreateInvoice() { ... }
  ```
- Utility functions: `camelCase`
  ```javascript
  export const chartNumbers = (data, fields) => { ... }
  export function localizeApiError(err, t) { ... }
  ```

**Variables & Constants**
- Variables: `camelCase` (e.g., `userID`, `invoiceData`, `queryClient`)
- Constants (config): `SCREAMING_SNAKE_CASE` atau `camelCase`
  ```javascript
  export const PAYMENT_METHODS = ["Cash", "Bank transfer", "Online"];
  export const CRYPTO_ASSETS = [{ code: "btc", name: "Bitcoin" }];
  ```
- Query keys: `camelCase` functions
  ```javascript
  export const invoicesKey = (params) => ["invoices", params || {}];
  export const invoiceKey = (id) => ["invoice", id];
  ```

**React Specifics**
- Props: `camelCase` (e.g., `invoice`, `onClose`, `isLoading`)
- Event handlers: `handle{Action}` (e.g., `handleSubmit`, `handleDelete`)
- Boolean props: `is{State}` atau `has{State}` (e.g., `isLoading`, `hasError`)

## File Organization

### Go Project Structure

**Controllers** (`app/controllers/`)
- One controller per domain, split bila >400 lines
- Pattern: `{domain}_controller.go` untuk main CRUD
- Helpers split: `{domain}_build.go`, `{domain}_rules.go`, `{domain}_helper.go`
- Order dalam file:
  1. Package imports
  2. Helper functions (unexported)
  3. HTTP handlers (exported)
  4. Supporting types

**Queries** (`app/queries/`)
- One query file per domain model
- Pattern: struct with embedded `*gorm.DB`
  ```go
  type InvoiceQueries struct {
      *gorm.DB
  }
  
  func (q *InvoiceQueries) GetInvoice(userID, id uuid.UUID) (Invoice, error) {
      // implementation
  }
  ```
- Error handling: `notFound(err)` translate GORM → `sql.ErrNoRows`
- Retry transient errors: `DoRetry(func() error { ... })`

**Models** (`app/models/`)
- Struct definitions dengan GORM + validation tags
- Constants terkait model (status, types)
- Helper methods untuk business logic (e.g., `EffectiveStatus()`)

**Tests** (`pkg/routes/*_test.go`)
- Flow tests wajib pakai fixtures dari `flow_fixtures_test.go`
- Pattern: `TestFeatureFlow`, `TestEdgeCaseHandling`
- Setup order: register → login → createClient → createInvoice → action → assert
- Helpers: `newTestApp()`, `doRequest()`, `decodeBody()`

### Frontend Structure

**Pages** (`webui/src/pages/`)
- One file per route
- Max 250 lines (enforced via CI)
- Extract components ke `components/*` bila terlalu besar

**Components** (`webui/src/components/`)
```
components/
├── invoice/          # Domain-specific
├── publicpay/
├── clients/
├── dashboard/
├── layout/           # App shells, sidebar, topbar
├── auth/             # Auth UI
└── ui/               # Reusable primitives
```

**Hooks** (`webui/src/hooks/`)
- One hook file per domain
- Export multiple hooks terkait domain:
  ```javascript
  // useInvoices.js
  export const invoicesKey = (params) => ["invoices", params];
  export function useInvoices(params) { ... }
  export function useInvoice(id) { ... }
  export function useCreateInvoice() { ... }
  export function useDeleteInvoice() { ... }
  ```

**API** (`webui/src/api/`)
- One file per domain, export API object:
  ```javascript
  export const invoicesApi = {
    list: (params) => apiClient.get("/invoices", { params }),
    get: (id) => apiClient.get(`/invoices/${id}`),
    create: (payload) => apiClient.post("/invoices", payload),
  };
  ```

## Import Style

### Go

**Import order**:
1. Standard library
2. Third-party packages
3. Local packages

```go
import (
    "context"
    "time"
    
    "github.com/gofiber/fiber/v3"
    "github.com/google/uuid"
    
    "github.com/tertua/tupay/app/models"
    "github.com/tertua/tupay/pkg/utils"
    "github.com/tertua/tupay/platform/database"
)
```

**No dot imports**, no blank identifiers kecuali side effects (`_ "github.com/joho/godotenv/autoload"`)

### JavaScript

**Import order**:
1. External packages (React, libraries)
2. Internal modules (`@/` alias)
3. Relative imports
4. Styles (bila ada)

```javascript
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { invoicesApi } from "@/api/invoices";
import { Button } from "@/components/ui/Button";

import { formatCurrency } from "./utils";
```

**`@` alias** menunjuk ke `webui/src` (configured di `jsconfig.json`)

**Axios imports**: hanya di `api/http.js`, semua tempat lain import dari sana

## Code Patterns

### Controller Pattern (Go)

**Standard flow**:
```go
func CreateInvoice(c fiber.Ctx) error {
    // 1. Autentikasi
    userID, err := utils.CurrentUserID(c)
    if err != nil {
        return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized", nil)
    }
    
    // 2. Parse & validate input
    var input models.InvoiceInput
    if err := c.Bind().Body(&input); err != nil {
        return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
    }
    if err := utils.NewValidator().Struct(&input); err != nil {
        return utils.ValidationFailed(c, err)
    }
    
    // 3. Database connection
    db, err := database.OpenDBConnection()
    if err != nil {
        return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
    }
    
    // 4. Business logic
    invoice := buildInvoice(userID, &input)
    if err := validateInvoice(invoice.Status, invoice.ClientID); err != nil {
        return err // already Fail() wrapped
    }
    
    // 5. Database mutation
    if err := db.CreateInvoice(&invoice); err != nil {
        return utils.Fail(c, fiber.StatusInternalServerError, "failed to create invoice", nil)
    }
    
    // 6. Side effects (best-effort)
    autoOnlinePaymentLink(db, userID, invoice)
    invalidateAggregates(c, userID)
    enqueueNotification(c, userID, "invoice.created", invoice.ID)
    
    // 7. Response
    detail := invoiceDetail(db, userID, invoice.ID)
    return utils.OK(c, fiber.StatusCreated, fiber.Map{"invoice": detail})
}
```

**Error response**: selalu English, stable message (ditranslate di frontend)

### Query Pattern (Go)

**CRUD standard**:
```go
type ClientQueries struct {
    *gorm.DB
}

// Get with user isolation + notFound translation
func (q *ClientQueries) GetClient(userID, id uuid.UUID) (models.Client, error) {
    var client models.Client
    err := q.Where("user_id = ? AND id = ?", userID, id).First(&client).Error
    return client, notFound(err)
}

// List dengan pagination
func (q *ClientQueries) ListClients(userID uuid.UUID, limit, offset int) ([]models.Client, error) {
    var clients []models.Client
    err := q.Where("user_id = ?", userID).
        Order("created_at DESC").
        Limit(limit).Offset(offset).
        Find(&clients).Error
    return clients, err
}

// Create
func (q *ClientQueries) CreateClient(client *models.Client) error {
    return q.Create(client).Error
}

// Update
func (q *ClientQueries) UpdateClient(client *models.Client) error {
    return q.Model(client).Updates(map[string]any{
        "name":       client.Name,
        "company":    client.Company,
        "updated_at": time.Now(),
    }).Error
}

// Delete
func (q *ClientQueries) DeleteClient(userID, id uuid.UUID) error {
    return q.Where("user_id = ? AND id = ?", userID, id).
        Delete(&models.Client{}).Error
}
```

**Transactional dengan retry**:
```go
func (q *InvoiceQueries) CreateInvoice(invoice *models.Invoice) error {
    return DoRetry(func() error {
        return q.Transaction(func(tx *gorm.DB) error {
            // atomic operations
            if err := tx.Create(invoice).Error; err != nil {
                return err
            }
            // more operations...
            return nil
        })
    })
}
```

### React Hook Pattern

**Query hook**:
```javascript
export function useInvoices(params = {}) {
  return useQuery({
    queryKey: invoicesKey(params),
    queryFn: () => invoicesApi.list(params),
  });
}

export function useInvoice(id) {
  return useQuery({
    queryKey: invoiceKey(id),
    queryFn: () => invoicesApi.get(id),
    enabled: !!id,
  });
}
```

**Mutation hook dengan invalidation**:
```javascript
export function useCreateInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload) => invoicesApi.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["invoices"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    },
  });
}

export function useDeleteInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id) => invoicesApi.delete(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["invoices"] });
      qc.removeQueries({ queryKey: invoiceKey(id) });
    },
  });
}
```

**Usage di component**:
```javascript
export function Invoices() {
  const [params, setParams] = useState({ page: 1, per_page: 20 });
  const { data, isLoading, error } = useInvoices(params);
  const { mutate: deleteInvoice } = useDeleteInvoice();
  
  if (isLoading) return <PageLoading />;
  if (error) return <QueryError error={error} />;
  
  return (
    <div>
      {data.invoices.map(invoice => (
        <InvoiceRow
          key={invoice.id}
          invoice={invoice}
          onDelete={() => deleteInvoice(invoice.id)}
        />
      ))}
    </div>
  );
}
```

### Component Pattern

**Functional component dengan props destructuring**:
```javascript
export function InvoicePaymentCard({ invoice }) {
  const { t } = useLang();
  const { mutate: recordPayment } = useRecordPayment();
  
  const handleRecord = (amount) => {
    recordPayment({ invoiceId: invoice.id, amount });
  };
  
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("invoice.payments")}</CardTitle>
      </CardHeader>
      <CardContent>
        {/* content */}
      </CardContent>
    </Card>
  );
}
```

**forwardRef pattern** (untuk UI primitives):
```javascript
export const Button = forwardRef(
  ({ className, variant = "default", ...props }, ref) => {
    return (
      <button
        ref={ref}
        className={cn(buttonVariants({ variant }), className)}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";
```

## Error Handling

### Go Backend

**Error creation**:
```go
// Simple error
return fmt.Errorf("payment: invalid amount %q", input.Amount)

// Wrapped error
return fmt.Errorf("payment: %w", err)

// Sentinel error
var ErrNotConfigured = errors.New("provider not configured")

// Custom error type
type ProviderError struct {
    Provider string
    Status   int
    Body     string
}
```

**Error response** (controller):
```go
// Generic error
return utils.Fail(c, fiber.StatusInternalServerError, "internal server error", nil)

// With details
return utils.Fail(c, fiber.StatusBadRequest, "validation failed", fiber.Map{
    "field": "email",
    "reason": "invalid format",
})

// Validation helper
if err := utils.NewValidator().Struct(&input); err != nil {
    return utils.ValidationFailed(c, err) // returns 400
}

// Not found
invoice, err := db.GetInvoice(userID, id)
if errors.Is(err, sql.ErrNoRows) {
    return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
}
```

**Error checking**:
```go
// Sentinel errors
if errors.Is(err, gateway.ErrNotConfigured) {
    return utils.Fail(c, fiber.StatusBadGateway, "gateway not configured", nil)
}

// Type assertion
var providerErr *gateway.ProviderError
if errors.As(err, &providerErr) {
    return utils.Fail(c, fiber.StatusBadGateway, "provider error", fiber.Map{
        "provider": providerErr.Provider,
        "status":   providerErr.Status,
    })
}
```

### JavaScript Frontend

**Error handling di mutation**:
```javascript
const { mutate, error } = useCreateInvoice();

const handleSubmit = async (data) => {
  try {
    await mutate(data);
    toast.success(t("invoice.created"));
    navigate("/invoices");
  } catch (err) {
    // 401 handled by interceptor (global logout)
    if (err.status !== 401) {
      toast.error(err.message || t("invoice.createFailed"));
    }
  }
};
```

**Error boundary di route**:
```javascript
// routes.jsx
{
  path: "/invoices",
  element: <Invoices />,
  errorElement: <RouteError />,
}
```

**Context guard**:
```javascript
export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
```

## Logging

### Go

**Structured logging**:
```go
logger.L().Info("starting server",
    "version", appVersion(),
    "stage", cfg.Stage,
    "addr", cfg.ListenAddr(),
    "db", database.Backend(),
)

logger.L().Error("failed to migrate database", "err", err)

logger.L().Warn("debug listener stopped", "err", err)
```

**Context** (implicit via logger middleware):
- Request ID
- User ID (bila authenticated)
- Path & method

### JavaScript

**Console logging** (development only):
```javascript
// Avoid console.log in production
if (import.meta.env.DEV) {
  console.log("Debug info:", data);
}
```

**Toast notifications** (user-facing errors):
```javascript
toast.error(t("invoice.deleteFailed"));
toast.success(t("invoice.created"));
toast.info(t("settings.saved"));
```

## Testing

### Go Test Patterns

**Table-driven tests**:
```go
func TestValidateInvoice(t *testing.T) {
    tests := []struct {
        name      string
        status    string
        clientID  *uuid.UUID
        wantError bool
    }{
        {"draft without client ok", "draft", nil, false},
        {"sent without client fails", "sent", nil, true},
        {"sent with client ok", "sent", &uuid.Nil, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateInvoice(tt.status, tt.clientID)
            if (err != nil) != tt.wantError {
                t.Errorf("got error %v, wantError %v", err, tt.wantError)
            }
        })
    }
}
```

**Flow test dengan fixtures**:
```go
func TestCreateInvoiceFlow(t *testing.T) {
    app := newTestApp()
    
    // Register + login
    cookies := registerAndLogin(t, app, "test@example.com", "password")
    
    // Create client
    client := createClient(t, app, cookies, clientSpec{
        Name: "ACME Corp",
    })
    
    // Create invoice (menggunakan fixture)
    invoice := newInvoice()
    invoice.ClientID = client["id"].(string)
    invoice.Status = models.InvoiceStatusSent
    
    created := createInvoice(t, app, cookies, invoice)
    
    // Assertions
    require.Equal(t, models.InvoiceStatusSent, created["status"])
    assert.Equal(t, client["id"], created["client_id"])
}
```

**Mock external services**:
```go
func TestGatewayWebhook(t *testing.T) {
    // Mock provider endpoint
    snapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"token":"snap-tok-1"}`))
    }))
    defer snapServer.Close()
    
    t.Setenv("MIDTRANS_SNAP_BASE_URL", snapServer.URL)
    
    // Test with mocked endpoint
    // ...
}
```

### JavaScript Test Patterns

**Unit test** (Node.js native):
```javascript
import { test } from "node:test";
import assert from "node:assert/strict";
import { chartNumbers } from "./chartData.js";

test("chartNumbers coerces decimal-string to numbers", () => {
  const input = [
    { key: "draft", value: "390720" },
    { key: "paid", value: "111000" },
  ];
  
  const result = chartNumbers(input, ["value"]);
  
  assert.equal(result[0].value, 390720);
  assert.equal(result[1].value, 111000);
});

test("handles null/undefined gracefully", () => {
  const input = [{ key: "test", value: null }];
  const result = chartNumbers(input, ["value"]);
  assert.equal(result[0].value, 0);
});
```

**Test naming**: describe behavior, bukan implementasi

## Comments

### Go

**Single-line comments only** (kecuali Swagger annotations):
```go
// GetInvoice retrieves an invoice by ID with user isolation.
func GetInvoice(c fiber.Ctx) error {
    // Parse invoice ID from path parameter.
    id, err := uuid.Parse(c.Params("id"))
    if err != nil {
        return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
    }
    // ...
}
```

**Swagger annotations** (multi-line ok):
```go
// GetInvoice godoc
// @Summary Get invoice by ID
// @Description Retrieve a single invoice with full details
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Router /invoices/{id} [get]
// @Security SessionCookie
func GetInvoice(c fiber.Ctx) error {
    // ...
}
```

**Package comments**:
```go
// Package gateway defines the payment provider abstraction.
package gateway
```

### JavaScript

**JSDoc untuk exported functions**:
```javascript
/**
 * Transforms chart data by coercing decimal strings to numbers.
 * @param {Array} data - Chart data array
 * @param {Array<string>} fields - Fields to coerce
 * @returns {Array} Transformed data
 */
export const chartNumbers = (data, fields) => {
  // implementation
};
```

**Inline comments untuk non-obvious logic**:
```javascript
// Replay is idempotent: second request returns cached response
if (cachedResponse) {
  return cachedResponse;
}
```

## Do's and Don'ts

### Backend (Go)

**✅ Do**
- Gunakan `utils.OK()` dan `utils.Fail()` untuk response
- Validasi input dengan `utils.NewValidator().Struct()`
- Isolasi user dengan `WHERE user_id = ?` di semua query
- Retry transient errors dengan `DoRetry()`
- Translate GORM errors ke `sql.ErrNoRows` dengan `notFound()`
- Split file bila >400 lines (controllers), >300 lines (queries), >200 lines (models)
- Gunakan fixtures di flow tests (`newInvoice()`, `createInvoice()`)
- Close response body di tests atau gunakan `decodeBody()`

**❌ Don't**
- Jangan raw SQL di controllers/queries (hanya di `platform/database`)
- Jangan inline JSON di flow tests (gunakan fixtures)
- Jangan force-push atau merge commits di `dev`/`main`/`master`
- Jangan float untuk money (gunakan `models.Money` / `decimal.Decimal`)
- Jangan skip CSRF untuk session-cookie mutations
- Jangan hardcode secrets (gunakan env vars)
- Jangan expose pprof ke public (localhost only)

### Frontend (JavaScript)

**✅ Do**
- Import axios hanya dari `api/http.js`
- Gunakan `@` alias untuk internal imports
- Invalidate queries setelah mutations
- Handle 401 via interceptor (skip di catch block)
- Toast user-facing errors dengan i18n
- Lazy load routes dengan `React.lazy()`
- Split components bila >250 lines
- Use `chartNumbers()` untuk pie chart data

**❌ Don't**
- Jangan import axios langsung di component/hook
- Jangan import `@react-pdf/renderer` di luar `InvoiceDocument.jsx` + `InvoicePdfDownloadContent.jsx`
- Jangan import `recharts` di luar Dashboard/Client/Reports chart pages
- Jangan pass string ke recharts pie (gunakan `chartNumbers()`)
- Jangan hardcode currency symbol (gunakan `formatCurrency()`)
- Jangan localStorage untuk sensitive data (gunakan httpOnly cookies)

### General

**✅ Do**
- Follow file size baselines (`scripts/file-size-baseline.json`)
- Run `make test` sebelum commit (backend)
- Run `npm --prefix webui run check:size` setelah perubahan
- Read file dengan tool Read, jangan `cat` di bash
- Edit file dengan tool Edit, jangan `sed` di bash
- Search dengan tool Grep, jangan `grep` di bash

**❌ Don't**
- Jangan commit ke `main` atau `master` (gunakan `make promote`)
- Jangan ubah `VERSION` di feat/fix commits (hanya di release)
- Jangan emoji di file kecuali user eksplisit minta
- Jangan create markdown docs tanpa diminta user
- Jangan mixed line endings (gunakan `.editorconfig`)

## CI Enforcement

Checks yang **wajib pass**:
- `make test` (gocritic, gosec, golangci-lint, coverage)
- `npm --prefix webui run lint`
- `npm --prefix webui test`
- `npm --prefix webui run check:charts` (recharts data validation)
- `npm --prefix webui run check:bundles:strict` (bundle size)
- `npm --prefix webui run check:size` (file size baseline)
- `node scripts/check-fixtures.mjs` (no raw JSON inline)
- `node scripts/sync-version.mjs --check` (VERSION sync)
- `make check-flow` (branching flow validation)

Tolak commit bila ada violations.
