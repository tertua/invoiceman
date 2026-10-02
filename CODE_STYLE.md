# Code Style Guide

Panduan style code untuk TuPay berdasarkan observasi codebase existing. Dokumentasi ini menjelaskan konvensi yang **sudah digunakan**, bukan yang seharusnya.

## Naming Conventions

### Go Backend

**Files**
- Controllers: `{entity}_{type}.go` (e.g., `invoice_controller.go`, `invoice_build.go`)
- Queries: `{entity}_query.go` (e.g., `invoice_query.go`, `client_query.go`)
- Models: `{entity}_model.go` (e.g., `invoice_model.go`); DTO: `{entity}_input.go`
- Tests: `{subject}_test.go` atau `flow_{feature}_test.go`
- Helpers: `{domain}_helper.go` / `{domain}_rules.go` (e.g., `audit_helper.go`, `invoice_rules.go`)

**Functions & Methods**
- Exported (public): `PascalCase` (e.g., `GetInvoice`, `CreateInvoice`)
- Unexported (internal): `camelCase` (e.g., `invoiceDetail`, `buildInvoice`)
- Method receivers: `func (q *InvoiceQueries) ListInvoices(...)`
- Test helpers: `func newInvoice() invoiceSpec`, `func createInvoice(t *testing.T, ...)`

**Variables**
- Local variables: `camelCase` (e.g., `userID`, `invoiceID`, `paidOn`)
- Struct fields (exported): `PascalCase` dengan JSON/GORM tags:
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

**Provider packages** (`platform/<provider>`)
- One package per payment provider: `platform/midtrans`, `platform/nowpayments`, future `platform/xendit`. Each exports `const GatewayName` plus a `Gateway` type implementing `platform/gateway.Gateway`.
- Behavior is declared through the optional capability interfaces in `platform/gateway/contracts.go` (`Methods`, `Configured`, `Sandbox`, `ChargeCurrency`, `RequiresDecimalAmount`, `BrowserSDK`, `PayerConfig`, `DefaultMethods`, `MinAmountChecker`) — implement only what the provider needs; callers type-assert.
- Config is read from `configs.Get().Provider("<name>")` using the `<PROVIDER>_<KEY>` env convention; never add a provider struct to `pkg/configs`.
- Controllers, models, and queries must **never** import a provider package — they talk to the `platform/gateway` registry only. CI: `scripts/check-no-provider-imports.mjs` (`make check.imports`).

### JavaScript Frontend

**Files**
- Hooks: `use{Entity}.js` (e.g., `useInvoices.js`, `usePayments.js`)
- API modules: `{entity}.js` (e.g., `invoices.js`, `adminGateway.js`)
- Components: `{Entity}{Type}.jsx` (e.g., `InvoicePaymentCard.jsx`)
- Pages: `{Entity}.jsx` (e.g., `Dashboard.jsx`, `ClientDetail.jsx`)
- Libraries: `{purpose}.js` (e.g., `chartData.js`, `utils.js`)
- Non-component entry files: `camelCase.jsx` (`main.jsx`, `routes.jsx`, `adminRoutes.jsx`)

**Functions & Components**
- Components: `PascalCase`; pages `export default function`, domain components named export:
  ```javascript
  export default function Invoices() { ... }
  export function InvoicePaymentCard({ invoice }) { ... }
  ```
- Hooks & utilities: `camelCase`, named export (`export function useInvoices()`, `export const chartNumbers = ...`)

**Variables & Constants**
- Variables: `camelCase`; props `camelCase`; handlers `handle{Action}`; booleans `is{State}`/`has{State}`
- Config constants: `SCREAMING_SNAKE_CASE` atau `camelCase`
  ```javascript
  export const PAYMENT_METHODS = ["Cash", "Bank transfer", "Online"];
  ```
- Query keys: camelCase factory functions
  ```javascript
  export const invoicesKey = (params) => ["invoices", params || {}];
  ```

## File Organization

### Go Project Structure

**Controllers** (`app/controllers/`)
- One file per responsibility, split sebelum melewati baseline (≤400 lines, CI `check:size`)
- `{domain}_controller.go` untuk main CRUD; split: `{domain}_build.go`, `{domain}_rules.go`, `{domain}_helper.go`
- Order dalam file: imports → helper (unexported) → HTTP handlers (exported) → supporting types

**Queries** (`app/queries/`)
- One query file per domain; struct dengan embedded `*gorm.DB`:
  ```go
  type InvoiceQueries struct{ *gorm.DB }

  // GetInvoice retrieves one invoice of an org by ID.
  func (q *InvoiceQueries) GetInvoice(orgID, id uuid.UUID) (models.Invoice, error) {
      var inv models.Invoice
      err := q.Where("org_id = ? AND id = ?", orgID, id).First(&inv).Error
      return inv, notFound(err)
  }
  ```
- Error handling: `notFound(err)` translate GORM → `sql.ErrNoRows`; retry transient: `DoRetry(...)`
- Limits: queries ≤300 lines, models ≤200 lines

**Models** (`app/models/`)
- Struct definitions dengan GORM + validation tags; status constants; helper methods (e.g., `EffectiveStatus()`)

**Tests** (`pkg/routes/*_test.go`)
- Flow tests wajib pakai fixtures dari `flow_fixtures_test.go`
- Pattern: `TestFeatureFlow`, setup: register → login → createClient → createInvoice → action → assert
- Helpers: `newTestApp()`, `doRequest()`, `decodeBody()`

### Frontend Structure

**Pages** (`webui/src/pages/`) — one file per route, max 250 lines; extract ke `components/*` bila terlalu besar

**Components** (`webui/src/components/`)
```
components/
├── admin/            # Admin console cards
├── auth/             # Auth UI
├── clients/          # Client domain
├── dashboard/        # Dashboard widgets
├── gateway/          # Gateway relay domain
├── invoice/          # Invoice domain (incl. InvoiceDocument.jsx, PDF)
├── layout/           # App shells, sidebar, topbar
├── payments/         # Payment domain
├── publicpay/        # Public pay page
├── reports/          # Reports domain
├── settings/         # Settings domain
├── settlement/       # Settlement domain
└── ui/               # Reusable primitives (Button, Card, Input)
```

**Hooks** (`webui/src/hooks/`) — one file per domain, export query key + query/mutation hooks:
```javascript
// useInvoices.js
export const invoicesKey = (params) => ["invoices", params];
export function useInvoices(params) { ... }
export function useCreateInvoice() { ... }
export function useDeleteInvoice() { ... }
```

**API** (`webui/src/api/`) — one file per domain, export API object:
```javascript
export const invoicesApi = {
  list: (params) => apiClient.get("/invoices", { params }),
  get: (id) => apiClient.get(`/invoices/${id}`),
  create: (payload) => apiClient.post("/invoices", payload),
};
```

**i18n** (`webui/src/lib/`) — `i18n.js` hanya re-export; sumber: `i18n.en.js`/`i18n.id.js` + split `i18n.*.settings.js`/`i18n.*.gateway.js`/`i18n.*.ui.js`

### Typography (frontend)

Tiga font, tiga peran — jangan dicampur:

- **Geist (`font-display`)** — display/UI app: judul halaman, judul kartu, angka besar (StatCard, gauge), tabel, navigasi. Ini default untuk seluruh app shell.
- **Inter (body, `--font-sans`)** — body text, label, tombol, input. Default tanpa kelas.
- **Cormorant Garamond (`font-serif`)** — editorial voice, **hanya** di: (a) Landing (hero + section heading), (b) halaman auth (`Login`/`Register`/`ForgotPassword`/`ResetPassword`, lewat `AuthShell`), (c) heading dokumen invoice di `InvoicePreview.jsx` (HTML preview saja — **bukan** PDF `@react-pdf/renderer`). Angka, tabel, dan StatCard tetap Geist.

Aturan: `font-serif` dilarang di shell app, tabel, atau angka. `font-display` wajib untuk heading app agar konsisten.

## Import Style

### Go

Import order: standard library → third-party → local; no dot imports, blank identifier hanya untuk side effects:
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

### JavaScript

Import order: external → internal (`@/` alias) → relative:
```javascript
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { invoicesApi } from "@/api/invoices";
import { Button } from "@/components/ui/Button";

import { formatCurrency } from "./utils";
```
- `@` alias → `webui/src` (configured di `jsconfig.json`)
- **Axios imports**: hanya di `api/http.js`

## Code Patterns

### Controller Pattern (Go)

Standard flow — auth → parse/validate → DB → business logic → mutation → side effects → response:
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
	// 3. DB (database.OpenDBConnection) + 4. Business logic (errors sudah Fail()-wrapped)
	if err := validateInvoice(input.Status, input.ClientID); err != nil {
		return err
	}
	// 5. Mutation + 6. Side effects (best-effort)
	if err := db.CreateInvoice(&invoice); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create invoice", nil)
	}
	invalidateAggregates(c, userID)
	// 7. Response — data fields langsung
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"invoice": invoiceDetail(db, userID, invoice.ID)})
}
```

### Query Pattern (Go)

CRUD standard dengan user/org isolation + notFound translation + pagination:
```go
func (q *ClientQueries) ListClients(orgID uuid.UUID, limit, offset int) ([]models.Client, error) {
	var clients []models.Client
	err := q.Where("org_id = ?", orgID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&clients).Error
	return clients, err
}
```

Transactional dengan retry:
```go
func (q *InvoiceQueries) CreateInvoice(invoice *models.Invoice) error {
	return DoRetry(func() error {
		return q.Transaction(func(tx *gorm.DB) error {
			return tx.Create(invoice).Error
		})
	})
}
```

### React Hook Pattern

```javascript
export function useInvoices(params = {}) {
  return useQuery({
    queryKey: invoicesKey(params),
    queryFn: () => invoicesApi.list(params),
  });
}

export function useCreateInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload) => invoicesApi.create(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["invoices"] }),
  });
}
```

Usage di component — loading/error guard sebelum render:
```javascript
export function Invoices() {
  const { data, isLoading, error } = useInvoices();
  const { mutate: deleteInvoice } = useDeleteInvoice();

  if (isLoading) return <PageLoading />;
  if (error) return <QueryError error={error} />;

  return data.invoices.map((inv) => (
    <InvoiceRow key={inv.id} invoice={inv} onDelete={() => deleteInvoice(inv.id)} />
  ));
}
```

### Component Pattern

```javascript
export function InvoicePaymentCard({ invoice }) {
  const { t } = useLang();
  const { mutate: recordPayment } = useRecordPayment();

  const handleRecord = (amount) => recordPayment({ invoiceId: invoice.id, amount });

  return (
    <Card>
      <CardHeader><CardTitle>{t("invoice.payments")}</CardTitle></CardHeader>
      <CardContent>{/* content */}</CardContent>
    </Card>
  );
}
```

UI primitives pakai `forwardRef` + `displayName` (lihat `components/ui/Button.jsx`).

## Error Handling

### Go Backend

**Error creation**: `fmt.Errorf("payment: %w", err)` untuk wrap; sentinel `var ErrNotConfigured = errors.New(...)`; custom type `ProviderError{Provider, Status, Body}`.

**Error response** (controller):
```go
// Generic
return utils.Fail(c, fiber.StatusInternalServerError, "internal server error", nil)
// Dengan details
return utils.Fail(c, fiber.StatusBadRequest, "validation failed", fiber.Map{
	"field": "email", "reason": "invalid format",
})
// Not found (setelah notFound() translation)
if errors.Is(err, sql.ErrNoRows) {
	return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
}
```

**Error checking**:
```go
if errors.Is(err, gateway.ErrNotConfigured) {
	return utils.Fail(c, fiber.StatusBadGateway, "gateway not configured", nil)
}
var providerErr *gateway.ProviderError
if errors.As(err, &providerErr) {
	return utils.Fail(c, fiber.StatusBadGateway, "provider error", fiber.Map{
		"provider": providerErr.Provider, "status": providerErr.Status,
	})
}
```

### JavaScript Frontend

- 401 ditangani interceptor (global logout) — jangan tampilkan ulang di catch
- User-facing errors via toast + i18n: `toast.error(t("invoice.deleteFailed"))`
- Route-level error boundary: `{ path: "/invoices", element: <Invoices />, errorElement: <RouteError /> }`
- Context guard: `if (!ctx) throw new Error("useAuth must be used inside AuthProvider")`

## Logging

### Go

Structured logging via `logger.L()` (text di dev, JSON di prod); context (request ID, user ID, path) ikut otomatis dari logger middleware:
```go
logger.L().Info("starting server", "version", appVersion(), "stage", cfg.Stage, "addr", cfg.ListenAddr())
logger.L().Error("failed to migrate database", "err", err)
```

### JavaScript

- `console.log` hanya di dev (`if (import.meta.env.DEV)`), jangan di production
- User-facing: toast (`toast.success/info/error`), bukan console

## Testing

### Go

**Table-driven** (contoh 1 kasus; pakai `t.Run` per baris):
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
**Flow test dengan fixtures** (wajib — `check:fixtures` melarang raw `POST /api/invoices`):
```go
func TestCreateInvoiceFlow(t *testing.T) {
	app := newTestApp()
	cookies := registerAndLogin(t, app, "test@example.com", "password")
	client := createClient(t, app, cookies, clientSpec{Name: "ACME Corp"})
	invoice := newInvoice()
	invoice.ClientID = client["id"].(string)
	created := createInvoice(t, app, cookies, invoice)

	require.Equal(t, models.InvoiceStatusSent, created["status"])
}
```

**Mock external services**: `httptest.NewServer` + `t.Setenv("MIDTRANS_SNAP_BASE_URL", server.URL)`. Library: `testify/assert` + `testify/require`; close response body / pakai `decodeBody()`.

### JavaScript

Node.js native runner (bukan jest/vitest), test co-located `*.test.js`:
```javascript
import { test } from "node:test";
import assert from "node:assert/strict";
import { chartNumbers } from "./chartData.js";

test("chartNumbers coerces decimal-string to numbers", () => {
  const result = chartNumbers([{ key: "draft", value: "390720" }], ["value"]);
  assert.equal(result[0].value, 390720);
});
```
Test naming: describe behavior, bukan implementasi.

## Comments

### Go

**Single-line comments only** (kecuali Swagger annotations yang boleh multi-line):
```go
// GetInvoice retrieves an invoice by ID with user isolation.
func GetInvoice(c fiber.Ctx) error {
	// Parse invoice ID from path parameter.
	id, err := uuid.Parse(c.Params("id"))
	...
}
```

Swagger annotation multi-line (ubah → jalankan `swag init` / `make swag`, `docs/` di-commit):
```go
// GetInvoice godoc
// @Summary Get invoice by ID
// @Tags invoices
// @Param id path string true "Invoice ID"
// @Success 200 {object} map[string]any
// @Router /invoices/{id} [get]
```

### JavaScript

- JSDoc untuk exported functions bila non-obvious
- Inline comments untuk non-obvious logic: `// Replay is idempotent: second request returns cached response`

### Integration contract marker

An integration point created before its caller exists (future provider capability, planned route, optional hook) is unreferenced **on purpose**. Mark it so it is never mistaken for dead code:

```go
// INTEGRATION CONTRACT — do not delete. See docs/adr/0001-integration-contract.md
func VerifyWebhookSignature(payload []byte, sig string) error { ... }
```

- Own comment line, immediately above the declaration: `//` in Go/JS, `--` in SQL, `#` in shell/YAML, `<!-- -->` in markup.
- The path must point to an existing `docs/adr/*.md` that explains why the door exists — no ADR, no marker.
- Never delete, "clean up", or de-reference marked code because it has no call site. To retire the door: deprecate the ADR first, then remove marker + code in the same change (full rules: `docs/adr/0001-integration-contract.md`).

## Do's and Don'ts

### Backend (Go)

**✅ Do**
- `utils.OK()` / `utils.Fail()` untuk semua response; validasi dengan `utils.NewValidator().Struct()`
- Isolasi data dengan `WHERE org_id = ?` / `user_id = ?` di semua query
- Retry transient errors dengan `DoRetry()`; translate GORM → `sql.ErrNoRows` dengan `notFound()`
- Split file sebelum melewati baseline (controllers ≤400, queries ≤300, models ≤200)
- Flow tests pakai fixtures (`newInvoice()`, `createInvoice()`)

**❌ Don't**
- Jangan raw SQL di controllers/queries (hanya `platform/database`)
- Jangan inline JSON di flow tests; jangan float untuk money (`models.Money` / decimal)
- Jangan skip CSRF untuk session-cookie mutations; jangan hardcode secrets
- Jangan expose pprof ke public (localhost only)

### Frontend (JavaScript)

**✅ Do**
- Import axios hanya dari `api/http.js`; selalu pakai alias `@/`
- Invalidate queries setelah mutations; handle 401 via interceptor (skip di catch)
- Toast user-facing errors dengan i18n; lazy load routes dengan `React.lazy()`
- Split components bila >250 lines; `chartNumbers()` untuk pie chart data

**❌ Don't**
- Jangan import `@react-pdf/renderer` di luar `InvoiceDocument.jsx` + `InvoicePdfDownloadContent.jsx`
- Jangan import `recharts` di luar Dashboard/Client/Reports chart pages
- Jangan pass string ke recharts pie; jangan hardcode currency symbol (pakai `formatCurrency()`)
- Jangan localStorage untuk sensitive data (httpOnly cookies)
- Jangan hardcode user strings — taruh di `i18n.en.js`/`i18n.id.js` (fix `en` dulu, `id` fallback)
- Jangan terjemahan literal EN→ID untuk istilah yang memang dipakai apa adanya — `Scan struk`, bukan `Pemindaian struk`; `Dashboard`, bukan `Dasbor`; `Gateway`, bukan `Gerbang`. Patokannya: apa yang dibaca user Indonesia sehari-hari, bukan padanan KBBI yang jarang dipakai. `id` ditulis ulang seperti penutur asli, bukan hasil translate tool per-kata. CI: `webui/src/lib/i18nLiteral.test.js`.

### General

**✅ Do**
- Follow file size baselines (`scripts/file-size-baseline.json`); turunkan entri baseline setelah split, jangan dinaikkan
- Read/Edit/Grep via tool dedicated, jangan `cat`/`sed`/`grep` di bash

**❌ Don't**
- Jangan hapus kode ber-marker `INTEGRATION CONTRACT` sebagai dead code — baca ADR-nya dulu (`docs/adr/0001-integration-contract.md`)
- Jangan commit ke `main`/`master` (pakai `make promote`); jangan ubah `VERSION` di feat/fix commits
- Jangan emoji di file kecuali diminta; jangan buat markdown docs tanpa diminta
- Jangan mixed line endings (ikut `.editorconfig`: Go tab, JS 2 space)

## CI Enforcement

Checks yang **wajib pass**:
- `make test` (gocritic, gosec, golangci-lint, coverage)
- `bun run --cwd=webui lint` and `bun run --cwd=webui test`
  - Frontend tests **must** run via `bun run --cwd=webui test` (= `node --test`), **never** `bun test`: the suite uses `node:test` mock APIs (`stub.reset()`) that bun's test runner lacks, so it would fail.
- `bun run --cwd=webui check:charts` (recharts data validation)
- `bun run --cwd=webui check:bundles:strict` (bundle size)
- `bun run --cwd=webui check:size` (file size baseline)
- `node scripts/check-fixtures.mjs` (no raw JSON inline)
- `node scripts/check-no-provider-imports.mjs` (no `app/` file imports a provider package — controllers use the `platform/gateway` registry)
- `node scripts/sync-version.mjs --check` (VERSION sync)
- `make check-flow` (branching flow validation)

Tolak commit bila ada violations.
