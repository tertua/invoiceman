# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.5.4] - 2026-09-26

### Fixed
- The first QRIS tap on a public pay link no longer fails with "payment
  could not be started": concurrent submits now collapse onto a single QR
  instead of racing each other at the gateway.

### Changed
- `POST /gateway/intents` now requires an `Idempotency-Key` header (same as
  `POST /gateway/invoices`), so a retried relay submit replays the original
  intent instead of opening a second payable charge. Send one UUID per
  payment intent.

## [v0.5.3] - 2026-09-25

### Changed
- "Sent" invoice status now uses sky blue instead of teal, so it no longer
  looks almost identical to "Paid" green — in badges, dashboard/Reports
  charts, and notification pills.
- USD → IDR rate hint in company settings shortened to one line; the input
  placeholder already shows the 18000 example.

## [v0.5.2] - 2026-09-25

### Added
- Public pay page embeds a crypto widget: pay with USDT (TRC20) without
  leaving the page — QR code, deposit address with copy button, exact amount
  to send, and a countdown, instead of redirecting to the provider.

### Changed
- Payment start failures now explain themselves: a rate-limited provider
  says it is busy and to retry in a few seconds, and invoices below the
  crypto minimum are told to pick another method. Both translate to English
  and Indonesian.
- Crypto payments are confirmed by the background reconcile poll as well as
  webhooks, so invoices mark themselves paid even where the provider cannot
  reach the server (local sandbox, missed IPN).

### Fixed
- NOWPayments direct payments failed with "price_amount must be a safe
  number" because the amount was sent as a string; the widget could not
  open a transaction at all.
- Rate-limited NOWPayments calls now retry briefly instead of failing the
  first attempt with a generic gateway error.

## [v0.5.1] - 2026-09-25

### Added
- AI invoice notes/terms now forbid thank-you and cooperation closings
  ("thank you", "terima kasih", "kerja sama", "kerjasama", "cooperation").

### Changed
- Gateway methods trimmed to `bank_transfer`, `qris`, `gopay`, `credit_card`;
  the Midtrans setting is a single-select dropdown defaulting to `gopay`.
- NOWPayments invoices are always charged in USD; IDR balances convert with
  the manual `usd_to_idr` rate instead of being sent as-is.
- Auth forms drop stiff example placeholders; login disables browser email
  history and uses `new-password` for the password field; address hints are
  unified to "Alamat Lengkap".
- Indonesian copy shortened: pending status "Menunggu", payment link title
  "Tautan Publik", invoice lines "Qty"/"Harga"; the share-link copy is
  icon-only.
- Settings merges the Account and Password tabs; the sidebar logout moves
  into the user card (avatar is display-only, only the logout icon acts).
- Dashboard dates honor the saved language from first paint, so ID renders
  "24 Sep 2026" instead of "Sep 24, 2026".
- Public pay secured-by note reads "Pembayaran diproses aman oleh mitra kami".

## [v0.5.0] - 2026-09-24

### Added
- Choose which Midtrans payment methods your account offers. The setting lists
  every method Midtrans supports; unchecking one hides it from payment links
  and blocks it if requested directly. Leaving all checked keeps the previous
  "every method" behaviour.

### Changed
- The public pay page now shows each method as a name-only button. The invoice
  total is already shown above, so the per-button converted amount that
  overlapped on every method is gone.

## [v0.4.0] - 2026-09-24

### Added
- Multi-provider payments: a `payment_method` on an intent now routes to any
  configured provider that supports it, regardless of the project default. The
  optional `gateway` field still pins a provider for legacy callers.
- Manual USD/IDR conversion: a new `usd_to_idr` setting (IDR per 1 USD) lets an
  IDR-only provider charge a USD invoice and vice versa. No realtime FX feed is
  used; when the rate is unset the intent is rejected instead of guessing.
- The public pay page now lets the payer choose a payment method first and only
  then opens the gateway paylink (Snap for Midtrans, hosted checkout for
  crypto), showing the converted amount per method.

### Changed
- Gateway transactions record the charged currency plus the source
  `invoice_currency`/`invoice_amount` and the `usd_to_idr` rate used.
- Settings gain the manual rate field; the public "secured by" note is now
  provider-neutral.

## [v0.3.1] - 2026-09-24

### Fixed
- The local Snap intent for an invoice works again. It moved out of the
  API-key `/gateway` namespace to `POST /api/v1/invoices/{id}/intents`, so a
  signed-in session is no longer rejected with "missing api key".
- Provider webhooks no longer register a duplicate route: `/webhooks/midtrans`
  and `/webhooks/nowpayments` are both served by the single generic
  `/webhooks/{gateway}` handler.

### Changed
- `payment_method` is validated against the known method ids and
  `GET /gateway/methods` now returns a typed `{ id, name }` list. The optional
  `gateway` field is documented as a legacy override; new integrations should
  send only `payment_method`.

## [v0.3.0] - 2026-09-24

### Added
- Provider-neutral payment methods: payment intents accept an optional
  `payment_method` (`qris`, `bank_transfer`, `gopay`, `crypto`, ...), routed to
  a provider that supports it, and a new `GET /api/v1/gateway/methods` lists
  the available methods without exposing provider names.
- Gateway transactions persist the chosen `payment_method`, expose it on intent
  responses, and forward it in the relay webhook payload.

### Changed
- Relay webhook payload keeps the provider-specific `payment_type` and now also
  carries the neutral `payment_method`; `gross_amount_idr` is an integer number
  of fiat minor units.

### Fixed
- Money no longer round-trips through `float64` on the gateway path: NOWPayments
  prices and verifies amounts as exact decimals and local settlement builds the
  amount from integer minor units, so fractional amounts stay precise.

## [v0.2.3] - 2026-09-24

### Added
- External service projects can create invoices through the API key-protected
  `/api/v1/gateway/invoices` endpoint with an inline customer payload.
- Integration invoice creation supports project-scoped external IDs and
  idempotent retries through the `Idempotency-Key` header.
- Swagger documentation and integration flow tests cover customer creation,
  invoice creation, retries, and duplicate external IDs.

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
