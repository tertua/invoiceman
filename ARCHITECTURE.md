# Architecture

## Overview

TuPay adalah aplikasi manajemen invoice berbasis web dengan integrasi payment gateway (QRIS/Midtrans, crypto/NOWPayments), relay untuk integrasi pihak ketiga, ekspenses, dashboard/reports, dan AI text generation (Gemini, English/Indonesian).

## Tech Stack

**Backend**
- Go 1.27.1 (pinned)
- Fiber v3 (HTTP framework)
- GORM (ORM: SQLite dev/test, PostgreSQL production)
- Redis (optional: sessions/cache, fallback in-memory)
- JWT (session auth dengan transparent refresh)

**Frontend**
- React 19.3 + React Router v7
- Vite 8 (MPA: 2 entry points)
- TanStack Query v5 (data fetching)
- Tailwind CSS 4 + CVA
- Recharts (charts), @react-pdf/renderer (PDF)

**External Services**
- Midtrans (QRIS, bank transfer, e-wallet)
- NOWPayments (crypto payments: BTC, ETH, USDT, dll)
- Gemini API (AI text generation)
- Cloudflare Turnstile (CAPTCHA, optional)
- MinIO/S3 (file storage: logo & receipt, fallback local)

## Directory Structure

```
tupay/
├── main.go                    # Entry point: startup, gateway registry, worker
├── bootstrap.go               # Startup helpers (healthcheck, pprof, version)
├── app/
│   ├── controllers/           # HTTP handlers (65 file, split per domain)
│   ├── models/                # Domain entities (38 file)
│   └── queries/               # Database access layer (32 file, zero raw SQL)
├── pkg/
│   ├── routes/                # Route definitions + 48 test files (flow/gateway/security)
│   ├── middleware/            # Auth, CSRF, rate limit, idempotency, timeout
│   ├── configs/               # Environment config loader & validator
│   ├── utils/                 # JWT, pagination, response envelope, validator
│   ├── logger/                # Structured logging (text dev, JSON prod)
│   ├── metrics/               # Prometheus-compatible /metrics endpoint
│   └── constants/             # App constants & version
├── platform/
│   ├── database/              # GORM setup, migrations, version guard
│   ├── gateway/               # Payment provider abstraction + registry + capabilities
│   ├── midtrans/              # Midtrans Snap + Core API implementation
│   ├── nowpayments/           # NOWPayments crypto implementation
│   ├── cache/                 # Redis/in-memory (sessions, aggregates)
│   ├── storage/               # File upload (local/MinIO/S3)
│   ├── mail/                  # Email templates + mailer
│   ├── outbox/                # Worker: mail/webhook delivery, reconcile
│   ├── relay/                 # Webhook signing + forward
│   ├── ai/                    # Gemini integration
│   ├── captcha/               # Turnstile verification
│   └── events/                # SSE per-user event broker
├── webui/
│   ├── index.html             # Product app entry
│   ├── admin.html             # Admin console entry (/admin)
│   └── src/
│       ├── main.jsx           # Product app bootstrap
│       ├── admin.jsx          # Admin app bootstrap
│       ├── routes.jsx         # User routes (protected + public)
│       ├── adminRoutes.jsx    # Admin routes (role guard)
│       ├── pages/             # Route pages (Dashboard, Invoices, dll)
│       ├── components/        # UI components (domain + reusable)
│       ├── api/               # Axios client per domain (23 files)
│       ├── hooks/             # React Query hooks (20 files)
│       ├── context/           # React Context (Auth, Theme, Lang)
│       ├── routes/            # publicRoutes.jsx (product public routes split)
│       └── lib/               # Utils + i18n (i18n.en/id.js, chart, payment, format)
├── docs/
│   ├── swagger.json/.yaml     # OpenAPI spec (generated)
│   ├── docs.go                # Swagger embed (generated)
│   ├── MODULE_MAP.md          # Domain ownership map
│   └── API_DOCS.md            # API conventions
├── scripts/                   # CI checks (file size, fixtures, version)
├── data/                      # Local state (gitignored: SQLite, uploads)
├── .github/workflows/         # CI (lint, test, dialect-check, version)
├── Makefile                   # Build targets
├── Dockerfile                 # API-only image (split deploy)
├── Dockerfile.dev             # API+FE embed (single container)
├── .env.example               # Config template
├── VERSION                    # Canonical version (2.0.0)
└── AGENTS.md                  # Repository guide & conventions
```

## Core Components

### Backend Layers

**Controllers** (`app/controllers/`)
- HTTP handler per domain (invoice, client, payment, gateway, dashboard, AI)
- Pattern: auth → parse input → validate → query DB → business logic → side effects → response
- Split per domain + feature: `invoice_controller.go`, `invoice_list_controller.go`, `invoice_build.go`, `invoice_rules.go`
- File size limit enforced: ≤400 lines (CI `check:size`)

**Queries** (`app/queries/`)
- Database access layer terpisah dari controller
- GORM only, zero raw SQL (raw SQL hanya di `platform/database`)
- Retry mechanism untuk transient errors (3x, 50ms doubling backoff)
- Pattern: `{Domain}Queries` struct embedding `*gorm.DB`

**Models** (`app/models/`)
- Domain entities dengan GORM tags
- Money type: fixed-point decimal (`shopspring/decimal`)
- Status constants per domain

**Middleware** (`pkg/middleware/`)
- `AuthRequired`: JWT cookie/Bearer dengan transparent refresh
- `RequireCSRF`: Double-submit token untuk session mutations
- `Idempotency`: Dedup via `Idempotency-Key` header (payment mutations); scope is per user **and** active org, so a key replayed after an org switch executes instead of returning the other tenant's response
- `GatewayAuth`: API key untuk service relay
- Rate limiting per IP (auth/public) atau API key (gateway)
- Timeout per request (AI/gateway intent routes)

**Platform** (`platform/`)
- Gateway abstraction: `gateway.Gateway` interface + registry + routing
- Gateway capabilities: optional interfaces in `platform/gateway/contracts.go` (see the capability table below)
- Providers: Midtrans (Snap + Core QRIS), NOWPayments (crypto) — each one package under `platform/<provider>`
- Cache: Redis/memory sessions, dashboard aggregates dengan singleflight
- Storage: local/S3 untuk logo (public) & receipt (private)
- Outbox worker: async mail, webhook delivery, stale intent reconcile
- Events: in-process SSE broker (per-user notification bell)

### Frontend Architecture

**Entry Points** (Vite MPA)
- Product app: `index.html` → `main.jsx` → `routes.jsx`
- Admin console: `admin.html` → `admin.jsx` → `adminRoutes.jsx` (basename `/admin`)

**Routing**
- User routes: protected shell (`AuthRequired` via `useAuth()`) + public routes
- Admin routes: role guard (redirect via `window.location` karena crossing entries)
- Lazy loading semua page components via `React.lazy()`

**State Management**
- React Query: data fetching, caching, mutations
- Context: Auth (user session), Theme (dark/light), Lang (en/id)
- No global state store (Redux/Zustand), domain state via hooks

**API Layer** (`api/`)
- Axios client per domain (invoices, clients, payments, gateway, dll)
- Centralized instance di `http.js` dengan interceptors:
  - Request: inject CSRF token dari cookie + locale header
  - Response: localize error, broadcast auth expired (401 → global logout)

**Hooks Layer** (`hooks/`)
- Wrap `@tanstack/react-query` untuk setiap domain
- Pattern: `useInvoices()`, `useCreateInvoice()`, `useDeleteInvoice()`
- Query keys via factory functions (`invoicesKey`, `invoiceKey`)

**Component Organization**
- Domain-specific: `components/invoice/*`, `components/publicpay/*`, `components/gateway/*`
- Layout: `components/layout/*` (AppShell, AdminShell, sidebar, topbar)
- Reusable UI: `components/ui/*` (Button, Card, Input, Badge, dll)

**Bundle Constraints** (CI enforced)
- `@react-pdf/renderer`: hanya `InvoiceDocument.jsx` + `InvoicePdfDownloadContent.jsx`
- `recharts`: hanya Dashboard/Client/Reports chart pages
- `axios`: hanya `api/http.js`

## Data Flow

### Request Flow (User Session)

```
User → Browser
  ↓ HTTP + cookies
FE Router (React Router)
  ↓ useQuery/useMutation
API Hook (TanStack Query)
  ↓ axios
API Client (http.js) + interceptors (CSRF, locale, error)
  ↓ /api/v1/*
Fiber Middleware Chain
  ├─ Rate Limit (per IP)
  ├─ AuthRequired (JWT cookie/Bearer → transparent refresh)
  ├─ CSRF (double-submit token)
  └─ Idempotency (payment routes)
  ↓
Controller Handler
  ├─ Parse & validate input
  ├─ DB query via app/queries
  ├─ Business logic (invoice rules, payment link ensure)
  ├─ Side effects (audit log, cache invalidate, enqueue notification)
  └─ Response (utils.OK / utils.Fail)
  ↓
React Query Cache
  ↓ re-render
Component
```

### Request Flow (Gateway Relay)

```
Third-party Service (via TuPay gateway)
  ↓ Authorization: <API-key>
API /gateway/* routes
  ↓
GatewayAuth middleware (verify API key → project identity)
  ↓
Gateway Controller
  ├─ Idempotency (required Idempotency-Key)
  ├─ Intent create: claim slot → route provider → charge
  ├─ Provider routing: preferred → deterministic registry search via capability
  └─ Response (provider token / crypto address / QRIS string)
  ↓
Downstream service (webhook forward via relay signing)
```

### Payment Flow (Public Pay Page)

```
Invoice with PaymentMethod=Online
  ↓ auto-generate payment link (bestEffort)
Public page /pay/:token
  ↓ GET /public/pay/:token (preflight: invoice + methods)
User picks method (QRIS / crypto asset)
  ↓ POST /public/intents (idempotency via asset)
Gateway charge
  ├─ Midtrans: hosted checkout token / Core QRIS
  └─ NOWPayments: crypto address + ExpiresAt
  ↓
User pays via provider
  ↓ webhook /api/webhooks/:gateway
Webhook Handler
  ├─ Verify signature
  ├─ Parse & normalize status
  ├─ Atomic settlement: lock invoice → create payment → mark paid
  └─ Forward to user notification endpoint (relay signing)
  ↓
SSE event → bell badge refresh
```

### Background Worker Flow

```
Outbox Worker (10s poll, batch 20)
  ├─ Mail queue: fetch due → claim → send SMTP → mark sent/failed
  ├─ Webhook queue: fetch due → claim → forward → mark delivered/failed
  ├─ Reconcile: poll stale pending intents → provider status → settle
  └─ Idempotency purge: delete expired keys (24h TTL)
```

## External Integrations

### Payment Gateways

**Midtrans**
- Snap API: hosted checkout (redirect URL + token)
- Core API: QRIS on-page (`qris.acquirer=gopay`, returns raw EMVCo string)
- Webhook: `/api/webhooks/midtrans` (signature verify)
- Status mapping: `settlement` → `StatusSuccess`, `expire` → `StatusExpired`, dll

**NOWPayments**
- Create payment: `POST /v1/payment` (price_amount, pay_currency)
- IPN webhook: `/api/webhooks/nowpayments` (HMAC verify)
- Minimum amount check: `GET /v1/min-amount?currency_from=usd&currency_to=btc`
- Sandbox mode: `NOWPAYMENTS_SANDBOX=true` → `https://api-sandbox.nowpayments.io`

**Gateway Abstraction**
- Interface: `gateway.Gateway` (`Name`, `CreateTransaction`, `ParseAndVerify`) — implemented once per provider
- Registry: `gateway.Register` / `gateway.Get` / `gateway.Names`; providers register in `main.go` (one line each)
- Default provider: `gateway.DefaultProvider()` returns `GATEWAY_DEFAULT_PROVIDER` when set, else `gateway.DefaultProviderName` (`"midtrans"`), so no controller hardcodes a provider
- Routing: preferred provider → deterministic search over the registry in name order (first configured + supporting the method); `supports` via `PaymentMethodProvider`
- Status normalization: provider-specific → unified `Status*` constants
- Controllers reach providers only through the registry — never by importing a provider package (enforced by `scripts/check-no-provider-imports.mjs`)

**Gateway capabilities** (optional interfaces in `platform/gateway/contracts.go`; a provider implements only what it needs, callers type-assert):

| Capability | Method | Meaning |
|---|---|---|
| `PaymentMethodProvider` | `Methods() []string` | the neutral method ids this provider offers |
| `ConfiguredProvider` | `Configured() bool` | credentials present; unusable providers are skipped in routing/status |
| `SandboxProvider` | `Sandbox() bool` | running against a test environment |
| `MinAmountChecker` | `MinAmount(ctx, currencyFrom, payCurrency)` | live per-currency minimum, so a doomed charge is hidden up front |
| `ChargeCurrencyProvider` | `ChargeCurrency() string` | the single fiat the provider charges (empty = invoice currency) |
| `DecimalAmountProvider` | `RequiresDecimalAmount() bool` | the provider bills a decimal amount (crypto/sub-unit fiat) |
| `BrowserSDKProvider` | `BrowserSDK() bool` | the stored token is consumable by an embedded browser checkout SDK |
| `PayerConfigProvider` | `PayerConfig() map[string]any` | public browser config (client key, env flag) — never server secrets |
| `DefaultMethodsProvider` | `DefaultMethods() []string` | the provider-side method narrowing (counterpart of the owner allowlist) |

**Provider configuration & browser config**
- Env convention: `<PROVIDER>_<KEY>` (upper-snake of the registry name), e.g. `MIDTRANS_SERVER_KEY`, `NOWPAYMENTS_IPN_SECRET`. Read via `configs.Provider("<name>")` + `ProviderString`/`ProviderBool`/`ProviderInt` — adding a provider needs no `pkg/configs` edit.
- Browser config: `GET /gateway/config?gateway=<name>` returns the gateway name, `configured`, and the provider's `PayerConfig` map; an unknown `?gateway=` answers 400, an empty one defaults to `gateway.DefaultProvider()`.

### How to add a provider

Adding provider X touches only these files (nothing under `app/controllers/*`, `app/models/*`, `pkg/configs/config.go`, `pkg/routes/*`, the frontend, or the database schema):

1. `platform/xendit/gateway.go` — `const GatewayName = "xendit"`; `type Gateway struct{}`; implement `Name()`, `CreateTransaction()`, `ParseAndVerify()`.
2. `platform/xendit/methods.go` — the optional capabilities X actually needs (e.g. `Methods()`, `Configured()`, `Sandbox()`, `ChargeCurrency()`, `PayerConfig()`, `DefaultMethods()`, `RequiresDecimalAmount()`).
3. `platform/xendit/config.go` — read `configs.Get().Provider("xendit")` (`XENDIT_API_KEY`, `XENDIT_CALLBACK_TOKEN`, …).
4. Split provider logic per the ≤400-line budget (`create.go`, `verify.go`, `status.go`) with `*_test.go` beside each.
5. `main.go` — one line: `gateway.Register(xendit.Gateway{})` (headroom exists after the `bootstrap.go` split).
6. `webui/src/lib/<x>.js` — **only if** X needs a browser SDK; otherwise nothing (`payerCheckout.js` falls back to `redirect_url`/`payment_url`).
7. `docs/MODULE_MAP.md` (gateway row) + this file's provider list + `.env.example`.

### AI (Gemini)

**Endpoints** (`ai_controller.go`)
- `POST /ai/invoice-description`: generate description dari invoice items
- `POST /ai/reminder-text`: payment reminder template
- `POST /ai/expense-description`: expense description dari kategori
- Language via `X-Locale` header (en/id), currency dari settings/invoice

**Rate Limit**
- Gemini free tier: 20 req/min
- 429 dari provider → HTTP 429 (`ai.rateLimited`)
- Cache di localStorage per user (regenerate overwrite)

### Email (SMTP)

**Queue Pattern** (outbox)
- Enqueue: render HTML + store di `mail_outbox` table (status=pending)
- Worker: poll due → claim → send → mark sent/failed
- Retry: exponential backoff, max 5 attempts

**Templates** (`platform/mail/templates/*.html`)
- Password reset link
- Payment link (invoice)
- Payment received notification

### File Storage

**Backends**
- Local: files di `STORAGE_DIR`, served `/uploads/logos/*` (public only)
- S3: MinIO/R2/B2/AWS dengan public-read untuk logo

**Domains**
- Logo: public, served via CDN/proxy
- Receipt: private, auth proxy `/expenses/:id/receipt`

## Configuration

**Environment** (`.env`, auto-loaded via godotenv)
- `SQL_DSN`: empty → SQLite (`SQLITE_PATH`), `postgres://...` → PostgreSQL
- `REDIS_HOST`: empty → in-memory sessions/cache
- `STAGE_STATUS`: `dev` (no graceful shutdown) / `prod` (graceful)
- Gateway keys: `<PROVIDER>_<KEY>` convention (`MIDTRANS_*`, `NOWPAYMENTS_*`), read generically via `configs.Provider()`; `GATEWAY_DEFAULT_PROVIDER` overrides the built-in default provider
- Optional: `GEMINI_API_KEY`, `TURNSTILE_SECRET`, `S3_*`

**Runtime Config** (`pkg/configs/`)
- Load & validate di startup (fail fast pada bad env)
- Config struct dengan defaults & validation rules
- Public config advertised via `GET /config` (branding, registration toggle)

**Database Migrations**
- Auto migrate via GORM `AutoMigrate` di startup
- Reversible migrations di `platform/database/migrations.go` (backend-agnostic)
- Version guard: newer DB schema → binary refuses to start

## Build & Deploy

### Development

**Local (no external services)**
```bash
make dev-be  # Build + run API di /tmp/opencode/tupay:5000
make dev-fe  # Vite dev :5173, proxy /api → :5000
```

**Docker Compose** (Postgres + Redis + API)
```bash
make docker.run   # Start stack
make docker.stop  # Stop stack
```

### Testing

**Backend**
```bash
make test              # Clean + gocritic + gosec + golangci-lint + coverage
go test ./...          # Quick run (in-memory SQLite)
go test ./pkg/routes -run TestName -v  # Focus test
```

**Frontend**
```bash
bun run --cwd=webui lint
bun run --cwd=webui test          # Node.js native test runner (via bun run)
bun run --cwd=webui check:charts      # Recharts data validation
bun run --cwd=webui check:bundles     # Bundle size advisory
bun run --cwd=webui check:bundles:strict  # CI enforcement
```

**CI Checks** (GitHub Actions)
- Lint: golangci-lint, eslint
- Test: unit + flow tests (48 file test di `pkg/routes/`)
- Dialect check: SQLite + PostgreSQL compatibility
- Version check: `VERSION` sync, `CHANGELOG.md` no stub sections
- File size: baseline enforcement (`scripts/file-size-baseline.json`)
- Fixtures: no raw JSON inline (`check:fixtures`)
- Provider imports: no `app/` file imports a provider package (`scripts/check-no-provider-imports.mjs`, `make check.imports`)
- Module map: `docs/MODULE_MAP.md` owner files exist both directions (`scripts/check-module-map.mjs`)

### Backups

```bash
make db.backup                      # snapshot DB → data/backups/ (SQLite: VACUUM INTO; PostgreSQL: pg_dump when SQL_DSN is set)
make db.restore FILE=<path|latest>  # replace DB from a snapshot (stop the backend first)
```

`db.restore` confirms interactively; `CONFIRM=yes` skips the prompt and `FORCE=yes` overrides the PostgreSQL connection guard.

### Production Deploy

**Split Mode** (default, `Dockerfile`)
- API-only image (scratch base)
- FE built separately (`make webui.check`) → host di nginx/Cloudflare
- FE proxy `/api` + `/uploads` ke BE
- `CORS_ORIGINS` set untuk cross-origin cookie sessions

**Single Container** (`Dockerfile.dev`)
- FE embedded di image (`SERVE_WEBUI=/spa`)
- Same-origin, no CORS
- Build: `docker build -f Dockerfile.dev --build-arg VITE_TURNSTILE_SITE_KEY=... -t tupay:dev .`

**Branching** (dev → main → master)
- `dev`: daily work
- `main`: stable (promoted via `make promote`, ff-only)
- `master`: production (promoted via `make promote-prod`, idempotent tag `v$(VERSION)`)
- Never force-push, never merge commits, verify dengan `make check-flow`

## Security & Hardening

**Authentication**
- JWT access token (15 min) + refresh token (720 hours)
- Transparent refresh: expired access → auto-refresh via refresh cookie
- Single session: login mints new SID, previous tokens invalidated
- Bearer fallback untuk non-cookie clients

**CSRF Protection**
- Double-submit token: readable `csrf_token` cookie → echo as `X-CSRF-Token` header
- Required untuk session-cookie POST/PATCH/DELETE
- Public GET dan API-key routes exempt

**Rate Limiting** (per IP atau API key)
- General: 100 req/min
- Auth: 10 req/min
- Public: 30 req/min
- Webhook: 60 req/min
- Gateway: 120 req/min

**CAPTCHA** (Cloudflare Turnstile, optional)
- Login/register/forgot-password
- Server: `TURNSTILE_SECRET` (runtime env)
- Client: `VITE_TURNSTILE_SITE_KEY` (build-time)

**Idempotency**
- Payment mutations require `Idempotency-Key` header
- 24h TTL, dedup via snapshot response
- Replay returns original result

**Webhook Security**
- Signature verification per provider (HMAC)
- Amount/currency guard: reject mismatch settlement
- Replay idempotency via `gateway_order_id` unique constraint

**Money Handling**
- Decimal strings crossing API (never float)
- Fixed-point arithmetic via `shopspring/decimal`
- Overpayment rejected (400)

**Audit Trail**
- CSRF middleware logs actions (user, entity, meta)
- Best-effort (tidak fail request)

**Secrets Management**
- Never baked in image
- Injected via `.env` atau env vars di runtime
- Localhost-only pprof (`DEBUG_PORT`)

## Performance & Scalability

**Caching**
- Dashboard/report aggregates: 60s TTL (Redis/memory)
- Explicit invalidation on mutations
- Singleflight: collapse concurrent cache misses

**Database**
- SQLite: dev/test, zero config
- PostgreSQL: production, connection pool (max 100)
- Retry transient errors: 3x, 50ms doubling backoff
- Row locking: `FOR UPDATE` di settlement atomicity

**Background Jobs**
- Worker poll: 10s interval, batch 20
- Async: mail, webhook delivery, reconcile
- Graceful drain: worker stopped after server shutdown

**Bundle Optimization**
- Lazy route loading via `React.lazy()`
- Code splitting: `@react-pdf/renderer` dan `recharts` di separate chunks
- Bundle budget enforcement (`check:bundles:strict`)

**Static Assets**
- Logo: public, CDN-friendly
- Receipt: auth proxy, tidak cached
- Vite dev: hot reload, no restart

## Observability

**Logging**
- Structured: text (dev), JSON (prod)
- Levels: debug, info, warn, error
- Context: request ID, user ID, action

**Metrics** (`GET /metrics`)
- Prometheus-compatible
- DB pool stats: open, idle, in-use
- Build version label

**Health Checks**
- `/healthz`: liveness probe
- `/readyz`: readiness probe (DB connectivity)
- Docker HEALTHCHECK: `./apiserver -healthcheck`

**Debugging**
- Localhost pprof: `DEBUG_PORT` (off by default)
- Never exposed publicly

## Key Design Decisions

1. **Dual-database support**: SQLite (dev simplicity) + PostgreSQL (production scale)
2. **Optional Redis**: in-memory fallback untuk single-instance dev
3. **MPA over SPA**: 2 entry points (product + admin), independent deploys
4. **Payment gateway abstraction**: add provider = new package under `platform/` implementing `gateway.Gateway` + optional capabilities, one `gateway.Register` in `main.go`, and `<PROVIDER>_<KEY>` env vars via `configs.Provider()`. No controller, model, config-struct, or frontend branch changes — enforced by `scripts/check-no-provider-imports.mjs` (`make check.imports` + CI).
5. **Transparent session refresh**: frontend tidak perlu token handling
6. **Outbox pattern**: reliable async delivery dengan retry
7. **File size limits**: enforced baseline, prevent runaway files
8. **Flow test fixtures**: centralized builders, no raw JSON inline
9. **Branching flow**: dev → main → master, ff-only, no merge commits
10. **Split/embed deploy**: API-only image OR single container, runtime choice
