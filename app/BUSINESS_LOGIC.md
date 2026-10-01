# Business logic — source of truth for domain rules

This file states **what TuPay means**, not how it is wired. Read it together with:

- `ARCHITECTURE.md` — system design, data flow, security, deploy.
- `CODE_STYLE.md` — naming, file structure, CI gates.
- `docs/MODULE_MAP.md` — where each domain lives (files, routes, FE pages).

Rules below were reconstructed from the code and are pinned by flow tests in
`pkg/routes/` (`flow_*_test.go`, `gateway_*_test.go`). When code and this file
disagree, the code + failing test win — then fix this file in the same change.

Conventions used here:

- **422 / 403 / 400 / 404** = HTTP status a rule returns (error envelope per `ARCHITECTURE.md`).
- `file.go` refs point at the enforcing code, not at prose.

---

## 1. Domain overview

TuPay is an invoicing & payments platform for small teams, multi-tenant by
**organization**.

Core entities (`app/models/`):

| Entity | Meaning |
|---|---|
| `Organization` + `Membership` | Tenant + user's role inside it (`owner` / `staff`) |
| `Client` | Customer record, org-scoped |
| `Item` | Catalog rate card, org-scoped |
| `Invoice` + `InvoiceItem` | The bill; has computed `Subtotal/TaxAmount/Total` |
| `Payment` | One money record against an invoice; voidable, never hard-deleted |
| `PaymentLink` | Unguessable token → public pay page for one invoice (1:1) |
| `GatewayTransaction` | One payment intent/charge at a provider (local invoice or relay) |
| `GatewayProject` | Downstream API consumer (API-key auth) for the relay |
| `Expense` + receipt | Business spend, org-scoped, receipt stored privately |
| `Settings` | Per-org row (1:1): currency, tax, invoice prefix, provider methods |
| `NotificationEndpoint/Delivery` | User webhooks + delivery attempts |

Aggregate flows:

1. **User session flow** — `ARCHITECTURE.md` → Request Flow (User Session).
2. **Public pay flow** — payer opens `/pay/:token`, creates an intent, provider
   confirms via webhook/reconciler → settlement writes a `Payment`.
3. **Gateway relay flow** — third party calls `/gateway/*` with an API key,
   same intent/settlement machinery, project-scoped.

---

## 2. Money & numbers (invariants)

Enforced by `app/models/money.go`, `app/controllers/invoice_build.go`.

1. `Money = decimal.Decimal` (shopspring), stored `DECIMAL(19,4)`. **Money never
   round-trips through `float64`** — `MoneyFromMinor` for minor units,
   `DecimalFromFloat` only for non-money multipliers (quantity, tax rate).
2. API serializes money as **decimal strings** (JSON), dates as `YYYY-MM-DD`.
3. Invoice arithmetic (exact decimal):
   - `line amount = quantity × rate`
   - `subtotal = Σ line amounts`
   - `taxable = max(subtotal − discount, 0)` (discount is a **flat amount**, not %)
   - `tax = taxable × taxRate / 100`; `total = taxable + tax`
4. Negative money is rejected at create (item rate, expense amount, invoice
   rate/discount, gateway invoice discount). **Known gap:** `UpdateItem` /
   `UpdateExpense` do not repeat the negative guard.
5. Currency conversion is **manual only**: `UsdToIdr` owner setting (default
   18000), IDR↔USD, nothing else (`platform/gateway/currency.go`). No live rates.
6. `paid_on` / `expense_date` / invoice dates are **date-only UTC midnights** so
   SQLite (text) and PostgreSQL (instants) behave identically.

---

## 3. Tenancy, roles, permissions

Enforced by `pkg/middleware/{auth,role,org}_middleware.go`, `pkg/utils/org.go`,
`app/controllers/invoice_rules.go`.

### 3.1 Two role systems

- **Platform**: `user` | `admin` (`User.UserRole`) — gates the `/admin` console
  via `RequireRoles("admin")`.
- **Org**: `owner` | `staff` (`Membership.Role`) — gates tenant actions via
  `OrgContext` + `RequireOrgRole`.

The active org comes from a session hint, resolved/fallback by
`ResolveActiveOrgID` (`app/queries/org_query.go`); no membership → 403
`org.notMember`.

### 3.2 Scoping law

- Every tenant query filters `org_id = ?`. A foreign id is **404, never 403**
  (row not found — no existence leak).
- Public reads that authorize by token (`payment link`, gateway project key)
  load unscoped, then **re-scope everything downstream by the row's OrgID**.
- Invoice numbers are per-org sequences (`INV-` + `%06d`), reserved atomically
  inside the create transaction.

### 3.3 Org-role matrix (summary)

| Action | staff | owner |
|---|---|---|
| Create invoice as `draft` | ✅ | ✅ |
| Create invoice as `sent`/`paid` | ❌ 403 | ✅ |
| Edit / delete a `draft` | ✅ | ✅ |
| Edit / delete a non-draft | ❌ 403 | ✅ (if not locked) |
| Submit draft for approval | ✅ | ✅ |
| Approve / reject | ❌ 403 | ✅ |
| Send / pull back / reopen status | ❌ 403 | ✅ |
| Record/void payment, payment links, settings, logo, org invite/rename | ❌ 403 | ✅ |

Errors: ownership → 403 `org.ownerRequired`; not a member → 403 `org.notMember`.

### 3.4 Account lifecycle (short)

`app/controllers/auth_register.go`, `auth_verify.go`.

- Statuses: `0 blocked`, `1 active`, `2 pending verification`.
- New account is `pending` when `REQUIRE_EMAIL_VERIFICATION` (default on) **and**
  it is not the first-ever account; the first account is always active and gets
  platform role `admin`.
- Login gate runs **after** password check; message is always
  `wrong email or password` (no enumeration). Pending → 403, blocked → 403.
- Registration: closed unless `ALLOW_REGISTRATION`, first account, or an invite
  token is present; the closed-check runs **before** the duplicate-email check
  to prevent probing.
- Tokens (email verify 24h, password reset 1h, org invite 7 days) are stored
  **hash-only**, single-use; resets kill live sessions.
- Strict single session: a new login invalidates the previous sid instantly.

---

## 4. Invoice domain

Owners: `app/controllers/invoice_*.go`, `app/models/invoice_model.go`.

### 4.1 Status model — stored vs effective

- **Stored** `status` ∈ `draft | sent | paid` (validator `oneof`; `pending` is
  *not* storable).
- **Effective status** = `ResolveEffectiveStatus(status, dueDate, total, paid,
  pending)`, evaluated in this order:
  1. `paid` — stored paid **or** `total > 0 && paid >= total`
  2. `pending` — a live gateway transaction exists (overlay)
  3. `overdue` — stored `sent` + `dueDate` passed
  4. else the stored status
- The overlay never writes to the column. A stale `draft` with money in flight
  reads `pending` everywhere (detail, list, counts, public pay, reports).
- Effective status feeds: invoice detail/list, status-count tabs, public
  payability, client outstanding, dashboard/reports buckets.

### 4.2 State machine

Transition matrix in `invoice_rules.go` (`from>to` → owner-only flag):

| Transition | Endpoint | Who |
|---|---|---|
| `draft → pending` | `POST /invoices/:id/submit` | any member |
| `pending → sent` (approve) | `POST /invoices/:id/approve` | owner (+ client required) |
| `pending → draft` (reject) | `POST /invoices/:id/reject` | owner |
| `draft → sent` (send) | `PATCH /invoices/:id(/status)` | owner |
| `sent → draft` (pull back) | `PATCH /invoices/:id(/status)` | owner |
| `paid → sent` (reopen) | `PATCH /invoices/:id(/status)` | owner, subject to §4.4 |

Unmapped transition → **400** `invalid status transition`. Same-status →
idempotent no-op.

### 4.3 Cross-field rules

- **Send requires a client** → 422 `client is required to send an invoice`
  (enforced at create, status flip, full update, and again at approve).
- ≥ 1 line item; item description required; quantity ≥ 0; tax rate ≥ 0;
  currency ≤ 3; dates `YYYY-MM-DD`.
- Client must exist **inside the caller's org** → 404 `client not found`.
- `PaymentMethod` ∈ `Cash | Bank transfer | Online` — **informational only**;
  `Online` only triggers auto payment-link minting (§5.2), never routing.

### 4.4 Locks

- **Pending lock** (money in flight, `PendingInvoiceIDs`): blocks status change,
  content rewrite, delete, *new* payment link, record payment, void payment —
  all → 422 `invoice has a pending payment`. The existing link stays usable.
- **Paid lock** (`isPaidLocked`): paid invoices can't be edited, deleted,
  re-minted, or reopened → 422 `invoice is already paid`.
  - `guardPaidReopen` distinguishes **money-paid** (`paid >= total`) — reopen
    forbidden until payments are voided — from a **manual** `paid` with no
    money rows, which may reopen.
- Guard order matters: role guard (403) runs before locks (422) for delete.

### 4.5 Approval flow

- Submit: only from `draft`; notifies **all org members**; audit
  `invoice.submitted`.
- Approve: owner only; must be `pending`; re-checks client; → `sent`;
  notifies all members.
- Reject: owner only; → `draft`; notifies **submitter only**.
- Role gating in UI: `webui/src/components/invoice/InvoiceStatusActions.jsx`.

---

## 5. Payments & payment links

Owners: `app/controllers/payment_*.go`.

### 5.1 Recording & voiding (manual)

`CreatePayment` guard chain: org → validation → `amount > 0` → invoice owned
(404) → pending lock (422) → **no overpayment** (`paid + amount > total` → 400
`payment exceeds invoice balance`) → insert → **auto-mark paid** when
`paid >= total && total > 0`.

`VoidPayment`: reason required (≤500); already voided → 409; **gateway-settled
payments can never be voided** (`GatewayOrderID` set) → 422; pending lock → 422;
voiding below total flips invoice `paid → sent`.

Voided rows are kept for audit and excluded from every balance/list/aggregate.
Hard delete exists for invoices/clients/items/expenses — **payments are the only
soft-deleted money record.**

### 5.2 Payment links

- One link per invoice (`InvoiceID` unique), token = 24 random bytes hex;
  table has no `org_id` — scoping is via join to the invoice.
- Minting blocked unless stored status `sent` and `paid < total`; idempotent
  get-or-create (`payment_link_ensure.go`).
- **Auto-mint**: create with `PaymentMethod=Online` + `status=sent` → best-effort
  link; failure never fails invoice creation.
- Share endpoints (`POST /payments/online[/send]`, owner-only): effective-draft
  → 422 `invoice is still a draft`; pending → 422; mail unconfigured → 501.
- Existing link on a paid invoice stays resolvable as a receipt with
  `can_pay=false`.
- Links are **IDR-only** in practice (charge conversion happens server-side).

### 5.3 Public pay (`/pay/:token`)

Gates: token 404 → effective-draft 422 → already-paid 400. Public payload strips
PII (`client_id`, `client_email`, `payment_link`) and **never exposes provider
names** (neutral labels, `public_method_label.go`).

Intent lifecycle (`payment_public_intent.go`, `intent_claim.go`):

1. **Order id is the concurrency primitive**: the id is claimed in DB *before*
   the provider call, so concurrent clicks serialize on the PK.
2. Claim marker is pseudo-status `processing` — invisible to reuse logic and to
   the reconciler. Losers `waitForClaim` (poll ≤ 12s) and reuse the winner.
3. Dead claim (failed, never reached provider) → adopted in place; stale claim
   (> 60s) → adopted.
4. Reuse rules: fresh `pending` intent with matching balance and unexpired →
   reused. Expired/failed → new charge under `base-rN` suffix (advances per
   base; crypto assets counted separately so a deposit address is never reused
   across assets).
5. `reusableIntent` = status `pending` + not expired + invoice amount covers
   current balance.
6. `Idempotency-Key` replay (scope `pay:<token>`) → same charge,
   `Idempotent-Replayed: true`.
7. Crypto is **select-then-create**: address requested only after the payer
   picks an asset; live minimum preflighted with `?pay_currency=` (empty asset
   → no minimum gate).

### 5.4 Gateway relay (`/gateway/*`, API key)

`app/controllers/gateway_*.go`, `pkg/middleware/gateway_middleware.go`.

- Auth: API key by hash lookup; identity **never** taken from the body;
  inactive project → 403; cross-project read → 404.
- `Idempotency-Key` header **required** on intent/invoice creation (scope
  `project:<slug>`).
- Deterministic claim slot `CLM-<slug>-<hash16>` (content hash) → same content
  serializes, different content charges independently; on success the claim row
  is swapped for the real intent in one transaction. Claim rows are never
  payable.
- Safe retry: pending intent for same `(project, external)` + matching amount →
  returned instead of a new charge.
- Project delete is **unused-only**: any transaction → 409, disable instead.

### 5.5 Webhooks & settlement

`gateway_webhook_controller.go`, `gateway_webhook_validate.go`,
`app/queries/gateway_settlement_query.go`.

- Verify signature/API-fetch first → unknown order 404 → status map per provider
  (Midtrans: `settlement|capture→success`, `expire→expired`, `deny|cancel|
  failure→failed`, …; NOWPayments: `finished→success`, **`partially_paid` stays
  `pending`**).
- **Amount/currency guard** on `success` for local intents: currency mismatch or
  drift > 1 minor unit → refuse to settle (400).
- Settlement is atomic (invoice row `FOR UPDATE`): idempotent on
  `gateway_order_id` (unique index) → a replay never double-writes; credits
  `balance`, or `gross` when `0 < gross < balance` (partial); auto-marks invoice
  paid when covered; then cache invalidation + `invoice.status_updated`.
- Downstream forward to relay projects is signed (`WebhookSecret`), enqueued as
  `WebhookDelivery`, retried by the outbox worker.

### 5.6 Reconciliation

`platform/outbox/reconcile.go`: polls `pending` transactions older than the
stale threshold; **backoff = touching `updated_at`** (no extra state, replicas
converge). Expiry is always **provider-sourced**, never guessed locally.
Unknown order at provider → mark `failed` (stops polling; invoice untouched, the
next payer click recharges under a fresh id). Success flows through the same
idempotent settlement path as webhooks.

---

## 6. Clients, items, expenses, settings

- **Client**: org-scoped; detail stats count only non-draft invoices as billed
  (sent/overdue/paid/pending); `outstanding = billed − paid`. List aggregates
  include pending-overlay invoices.
- **Item**: org-scoped; rate ≥ 0 at create; feeds catalog only (no aggregates).
- **Expense**: org-scoped; amount ≥ 0 at create; currency defaults to settings;
  date-only `expense_date`; totals/categories always cover the whole filtered
  set (never just the page).
- **Receipts**: private storage (`receipts/<orgID>/<expenseID><ext>`), served
  only via the authed proxy; max 10 MiB; `image/*` or PDF; **SVG rejected**
  (stored-XSS) and legacy SVG is force-downloaded. Logos are public, also
  no SVG, max 400 KiB.
- **Settings**: per-org row created lazily on read (`FirstOrCreate`); defaults
  `IDR`, tax `11`, prefix `INV-`, `UsdToIdr 18000`; update is owner-only;
  response emits both canonical `provider_methods` and legacy
  `midtrans_methods`.
- **Provider method allowlist**: the default provider is pinned to **QRIS
  regardless of stored values** (`gateway_allowlist.go`); other providers are
  unrestricted. Methods are declared by capability contracts, never by provider
  name switches (`platform/gateway/contracts.go`).

---

## 7. Dashboard & reports

Owners: `app/queries/dashboard_query.go`, `report_query.go`; cache
`platform/cache/aggregates.go`.

- All aggregates are org-scoped, optionally `?currency=`-filtered (**client
  count is not currency-filtered**), cached under `agg:v1:<org>:<kind>:<cur>`
  with 60s TTL + singleflight, and **explicitly invalidated** after every
  client/expense/invoice/payment write (`cache_helper.go`). Cache failures fail
  open to DB.
- Dashboard: invoice/client counts, total revenue (Σ non-voided payments),
  paid this month (local-zone month), outstanding/overdue from effective
  status, 6-month revenue series, 5 recent invoices.
- Reports: `Revenue`, `Expenses`, `NetProfit = Revenue − Expenses`,
  `Outstanding` (open balance of sent/overdue/pending only — drafts and paid
  never count), 6-month buckets (revenue by `paid_on`, expenses by
  `expense_date`), aging buckets `current/d1_30/d31_60/d61_90/d90_plus`,
  status breakdown by **invoice total** (pending slice shows full total),
  top 5 clients by billed.
- Payment method for money in reports is always **non-voided** payments.

---

## 8. Notifications & live events

- Events: `invoice.created`, `invoice.status_updated`, `payment.created`,
  `payment.voided` (+ `notification.test`, which bypasses the subscription
  filter).
- Endpoints are **user-scoped** (not org); URL guard: http(s) + host, private
  hosts rejected **in prod only**; secret shown once, rotatable.
- Fan-out: submit/approve/status → **all org members** (one row per member);
  reject → **submitter only**.
- Delivery: enqueued as rows, claimed atomically by the outbox worker, retried
  with `next_retry_at`, terminal `dead` when exhausted; `event_id` is the
  downstream idempotency key.
- SSE (`GET /events`): per-user stream, 25s heartbeat, no timeout middleware —
  org fan-out repeats the publish, never widens a shared stream.

---

## 9. Cross-cutting checklist (read before changing anything)

1. **Never store `pending`** on an invoice — it is an overlay only.
2. **Never mutate money through float**; compare decimals exactly.
3. **Org scope every query**; foreign id → 404.
4. Respect the two locks: pending (422, 6 mutating paths) and paid (422).
5. Sending an invoice always requires a client — re-check at every entry point.
6. Idempotency scopes differ by actor: `user:<id>:org:<id>` (session),
   `project:<slug>` (relay), `pay:<token>` (public pay). Key reuse across
   tenants must stay impossible.
7. Settlement/replay safety rests on `gateway_order_id` uniqueness + row lock —
   don't add a second payment-writing path outside
   `SaveTransactionAndSettleInvoice`.
8. A claim row (`CLM-…` / `processing`) must never be treated as payable.
9. Void payments, never delete them; gateway-settled payments are immutable.
10. Provider-specific behavior belongs behind capability contracts in
    `platform/gateway`, not `if provider ==` in controllers.
11. New business rule → extend the matching flow test in `pkg/routes/` and
    update this file in the same change.
12. Error messages are stable English strings (API contract); UI translates.

---

## 10. Known gaps / sharp edges

Track these before adding more rules on top:

- `UpdateItem` / `UpdateExpense` lack the negative-amount guard that create has.
- `ensurePaymentLink` overloads one sentinel: a draft invoice hitting it reports
  "invoice is already paid" (400) — share controllers pre-empt with the correct
  422 draft message.
- `errLinkPaid` and `pending` counts: effective-`pending` invoices are tallied
  under `all` only, not in any status tab bucket.
- Two concurrent first-ever registrations can both become `admin`.
- Notification SSRF guard is prod-only (dev allows loopback targets).
- `provider_methods` stored value is intentionally ignored for the default
  provider (always QRIS) — editing it changes nothing.
