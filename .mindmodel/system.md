# TuPay System Overview

TuPay adalah invoice management system dengan payment gateway integration, dibangun dengan Go backend (Fiber v3) dan React frontend (Vite 8). Sistem mendukung multi-currency invoicing, AI-assisted features, dan gateway relay untuk downstream projects.

## Core Architecture

**Backend Stack**
- Go 1.27.1 dengan Fiber v3 framework
- GORM untuk dual-dialect database (SQLite + PostgreSQL)
- Redis optional (in-memory fallback)
- JWT session: access token (15m) + refresh cookie (7d), strict single-session
- Decimal arithmetic via shopspring/decimal untuk money (never float64)

**Frontend Stack**
- React 19.3 dengan Vite 8
- TanStack Query untuk server state management
- Tailwind CSS 4 untuk styling
- Axios centralized di http.js dengan CSRF + X-Locale interceptors
- Restricted imports: @react-pdf/renderer (2 files), recharts (3 components)

**Database**
- Dual-dialect: SQLite (default/test) + PostgreSQL (production)
- GORM AutoMigrate at startup
- Backend-agnostic migrations dengan Down rollback
- UUID primary keys, decimal.Decimal untuk money

## Key Patterns

**Layering**
- Controllers handle HTTP, validate input, call business logic
- Queries encapsulate semua database operations (never inline SQL)
- Models define structs + validation tags
- Platform layer: gateway, database, cache, mail, outbox abstractions
- Pattern: CurrentUserID → OpenDBConnection → logic → utils.OK/Fail

**API Design**
- Envelope: utils.OK untuk success (direct data fields), utils.Fail untuk errors (error.message + details)
- Versioning: /api/v1 (current) dan /api (legacy dengan Deprecation headers)
- Route order critical: /api/v1 before /api, public→gateway→private within prefix
- Dates: YYYY-MM-DD strings, Money: decimal strings on wire

**Security**
- Three auth schemes (never mixed): SessionCookie (most endpoints), ApiKeyAuth (gateway relay), public payment links
- CSRF double-submit: csrf_token cookie → X-CSRF-Token header (mutations only)
- Idempotency-Key: money-moving mutations dengan replay protection
- Webhook verification: signature/HMAC check in ParseAndVerify

**Testing**
- Default: in-memory SQLite (fast, no external services)
- Opt-in integration via INVOICEMAN_TEST_PG_DSN dan INVOICEMAN_TEST_REDIS_ADDR
- Flow tests MUST use flow_fixtures_test.go builders (check:fixtures CI)
- Pattern: newInvoice(), createInvoice(), createClient() (never raw JSON)

**Integration**
- Gateway interface: Name(), CreateTransaction(), ParseAndVerify()
- Register via gateway.Register(provider.New()) in main.go
- Outbox worker: durable job queue (mail, webhooks) dengan retry backoff 1m→2m→4m→...→2h (max 10 attempts)
- Gateway relay: downstream projects charge via Tupay, webhook forwarded dengan HMAC

**AI & Localization**
- X-Locale header (en/id) determines AI response language
- languageDirective + currencyDirective prevents $ defaults
- Gemini 2.0-flash default model
- AI text cached in localStorage per user+invoice+tone+language
- Errors logged server-side, generic message to client

## Domain Model

**Invoice Lifecycle**
- States: draft → sent → paid
- Lock points: pending transaction atau fully paid
- Effective status: stored + overlays (overdue jika past due_date, pending jika transaction exists)
- Business rule: sent invoice MUST have client_id (ErrClientRequiredToSend)

**Payment Flows**
- Manual record: user enters payment directly
- Gateway relay: downstream projects charge via API key
- Public pay: PaymentLink dengan token, scoped idempotency

**Money Handling**
- Type: decimal.Decimal (shopspring/decimal)
- Wire format: decimal strings ("100000.50")
- Never float64 (precision loss)
- Currency: 3-letter ISO codes (IDR, USD, EUR)

## File Organization

**Size Limits (CI enforced)**
- Controllers ≤400, queries ≤300, models ≤200
- Components ≤250, hooks ≤150
- Frontend i18n per language ≤900
- scripts/file-size-baseline.json: never grow past recorded entry
- Split strategy: gateway_*_controller.go, flow_*_test.go, components/* cards

**Naming Conventions**
- Backend files: snake_case.go (invoice_controller.go)
- Frontend: PascalCase components, camelCase hooks
- Database: snake_case (invoice_number, created_at)
- Constants: PascalCase dengan prefix (InvoiceStatusDraft)
- Gateway names: lowercase strings ("midtrans")

**Comments**
- One-line only except Swagger annotations
- Multi-line comments fail CI

## Build & Deployment

**Branch Flow**
- dev (active) → main (stable) → master (production)
- Promotion: `make promote` (ff-only), `make promote-prod` (+ v tag)
- No merge commits, no force-push on protected branches
- Version bumps only in release commits

**CI Checks**
- File size: check:size (enforces baseline)
- Test fixtures: check:fixtures (enforces builder usage)
- Import restrictions: check:bundles:strict (axios, recharts, react-pdf)
- Version sync: sync:version --check
- Module map: check:map

**Local Development**
- BE: `make dev-be` (binary → /tmp/opencode/tupay, :5000)
- FE: `make dev-fe` (Vite :5173, proxies /api + /uploads)
- Full test: `make test` (gocritic + gosec + golangci-lint + coverage)
- govulncheck ./... after go.mod/go.sum changes

## Critical Constraints

1. **File sizes are hard caps** - growing past baseline triggers split, never raise limit
2. **Route order matters** - /api/v1 before /api, public→gateway→private within prefix
3. **Query separation** - no inline SQL in controllers, raw SQL only in platform/database
4. **Money is decimal** - never float64, decimal strings on API wire
5. **Test fixtures mandatory** - check:fixtures CI enforces flow_fixtures_test.go usage
6. **Import restrictions** - axios/recharts/@react-pdf restricted (CI check:bundles:strict)
7. **One-line comments** - except Swagger annotations
8. **Dual-dialect DB** - migrations must work on SQLite and PostgreSQL
9. **Business rules return 422** - separated from malformed input (400)
10. **AI needs currency directive** - prevents $ defaults in localized responses
