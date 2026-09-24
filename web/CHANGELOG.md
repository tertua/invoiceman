# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Sending an invoice now requires a client: the backend returns `422 client is
  required to send an invoice`, the detail page hides the "Sent" action, and the
  editor blocks the save. Drafts may still be saved without a client.
- Frontend unit tests (`npm test`, Node's built-in runner) and a `check:charts`
  guard that fails when a `<Pie>` renders un-coerced money.
- Flow tests build invoices through the shared `invoiceSpec` fixture
  (`flow_fixtures_test.go`); `check:fixtures` fails on hand-written invoice JSON.

### Fixed
- Status donuts on the dashboard and reports render again: money arrives as
  decimal strings, which Recharts' `Pie` ignores when summing slices.

## [v0.2.2] - 2026-09-24

### Added
- Awaiting-payment (`pending`) is a first-class display status: a live gateway
  transaction overlays the stored status on the dashboard, reports, client
  list/detail and the public pay page, and gets its own slice in the status
  donut.
- Public pay page shows a distinct "Awaiting payment" badge while a Snap intent
  is live, and keeps its Pay button so the in-flight intent can be resumed.

### Changed
- Money in flight locks the invoice: content edits, status flips, hard delete,
  manual payment recording, voiding and creating a new payment link all return
  `422 invoice has a pending payment`. An existing public link stays usable and
  visible.
- Pending invoices count as receivables (outstanding / totalBilled) in
  dashboard, reports and client aggregates; a stale draft with a live intent
  reads as payable instead of "still a draft".
- `payment_controller.go` split into `payment_public_controller.go`; public
  draft gates go through a shared `effectiveInvoiceStatus` helper.

### Fixed
- Dashboard/report/client outstanding no longer drops invoices whose stored
  status is draft while a gateway payment is pending.
- The status donut no longer hides the pending slice.
- Public pay and online-link endpoints no longer reject a pending invoice as
  "still a draft".

## [v0.2.1] - 2026-09-24

### Added
- Awaiting-payment status with gateway reconciliation: live transactions
  surface as pending and stale ones are settled by the outbox worker.
- Invoices get empty default terms; AI notes read as money wisdom.

### Changed
- Money uses the `models.Money` decimal alias consistently, avoiding frontend
  precision loss.

### Fixed
- Parse Midtrans `gross_amount` as decimal instead of float64.

## [v0.2.0] - 2026-09-24

### Added
- Per-user language persists in the database; session caches warm during the
  login delay.
- `dev-be` / `dev-fe` make targets for the local dev run.

### Changed
- Money is stored and returned as fixed-point decimals.
- Unified design tokens, radius, and focus states.
- Project license switched to GPLv3.

## [v0.1.7] - 2026-09-23

### Changed
- Oversized files split; file-size budgets enforced in CI.

### Fixed
- Auto-mark invoices paid once payments cover the total, syncing client
  aggregates.
- Lock paid invoices and persist the payment link display.
- Strip sensitive fields from public payment data.
- Use the `PAY-` prefix for local order ids.
- Force AI terms output to a numbered list only.

## [v0.1.6] - 2026-09-23

### Added
- Public gateway status endpoint; gateway availability banner and gateway
  select in the SPA.

### Changed
- Payments are voided instead of hard-deleted; gateway-settled payments hide
  the void action.
- Hide the AI reminder once an invoice is paid.

### Fixed
- NOWPayments decimal guard.

## [v0.1.5] - 2026-09-23

### Added
- User webhook targets for invoice and payment events.
- AI text follows the UI language and user currency; Gemini rate limits map to
  HTTP 429; business summary and payment-reminder drafts persist in
  localStorage.
- Browser language detection for first-time visitors.

### Changed
- React 19.3 and Vite 8.3.

### Fixed
- Settle local invoice payments atomically.
- Correct the inverted custom UUID validator rule; fix the aging chart measure.
- Preserve line breaks in invoice notes and terms.
- Count only sent invoices as outstanding.

## [v0.1.4] - 2026-09-22

### Added
- Registration kill switch via `ALLOW_REGISTRATION`.
- Back-to-home link on the login page.
- Optional embedded SPA via `SERVE_SPA_DIR` and `Dockerfile.dev`.

## [v0.1.3] - 2026-09-22

### Added
- Auto-redirect to login on an expired session; query error states with retry.

### Security
- Stop baking `.env` into the Docker image.
- Serve only public logos under `/uploads`; receipts stay behind the auth proxy.
- Store password-reset tokens hashed.
- Reject SVG uploads; serve legacy SVG downloads as attachment.
- Strict single-session, atomic first-admin, CSRF rotation, proxy trust and
  login audit; secure cookie clear and HS256 pin.

## [v0.1.2] - 2026-09-22

### Added
- Per-domain module map with `check:map` and CI enforcement.
- Self-contained Swagger (session cookie scheme, auth flows, envelopes).
- Dual-backend parity: cache, retry, rollback and guards.

### Changed
- Docker build context excludes `web`, `data`, and local artifacts.

## [v0.1.1] - 2026-09-22

### Added
- Database-backed cookie sessions, RBAC, and an admin users UI.
- Clients, invoices, dashboard, catalog items, expenses, payments and reports.
- Public payment links and a public pay page with language toggle and `?lang=`.
- Central Midtrans relay, generalized into a multi-gateway framework (Midtrans
  Snap, NOWPayments).
- Generic SMTP email delivery.
- Platform hardening: recover/requestid/helmet/rate-limit/health probes and
  SIGTERM shutdown, typed config + slog + Prometheus metrics, pagination +
  idempotency keys + outbox worker, API versioning + CSRF/audit + Turnstile +
  version guard, S3-compatible storage and HTML mail templates.
- Route-level error pages, bundle splitting and budgets, EN/ID localization of
  backend errors, currency-aware dashboard totals.

### Changed
- Brand renamed to Invoiceman with `APP_NAME` override.
- Feature modules split; the API client was renamed to `http`.

### Fixed
- Fail fast on an unsupported `SQL_DSN` instead of silently falling back to
  SQLite.
- Accessibility: pointer cursor on buttons, keyboard-accessible cards and rows,
  Escape closes modals, buttons default to `type="button"`.
- Localize AR aging buckets and monthly chart labels via stable backend keys.

## [v0.1.0] - 2026-09-20

Initial commit. Baseline scaffold: a Vite + React 19 + Tailwind v4 SPA over a
Go (Fiber + GORM) API with SQLite/PostgreSQL support, cookie-session auth,
invoice CRUD, clients, dashboard, reports, and PDF export.
