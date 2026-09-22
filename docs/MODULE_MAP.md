# Module map — where things live

Read first: `AGENTS.md`, then this file, then `main.go` →
`pkg/routes/versioning.go`. After that, jump straight to the domain row
below; you should not need to grep the codebase to find an owner file.

Request path: `web/src/pages/*.jsx` → `web/src/hooks/*` →
`web/src/api/*.js` (axios only in `http.js`) → `pkg/routes/*` →
`pkg/middleware/*` → `app/controllers/*` → `app/queries/*` (one file per
domain: SQL/GORM detail lives here, never inline in controllers) →
`platform/*` + `pkg/*`.

## Domains

| Domain | Backend controller | Model(s) | Routes | FE api / hook | FE pages | Platform / misc |
|---|---|---|---|---|---|---|
| Auth & session | `app/controllers/auth_controller.go` | `auth_model.go`, `user_model.go` | `public_routes.go` (register/login/forgot/reset), `private_routes.go` (me/logout) | `api/auth.js`, `context/AuthContext.jsx` | `Login.jsx`, `Register.jsx`, `ForgotPassword.jsx`, `ResetPassword.jsx` | `pkg/middleware/auth_middleware.go`, `platform/cache` (sessions), session issue/parse in `pkg/utils` |
| Users (admin) | `admin_controller.go` | `admin_model.go`, `user_model.go` | private | `api/admin.js`, `hooks/useAdminUsers.js` | `AdminUsers.jsx` | `pkg/middleware/role_middleware.go` |
| Clients | `client_controller.go` | `client_model.go` | private | `api/clients.js`, `hooks/useClients.js` | `Clients.jsx`, `ClientDetail.jsx`, `components/clients/ClientCharts.jsx` | — |
| Items | `item_controller.go` | `item_model.go` | private | `api/items.js`, `hooks/useItems.js` | `Items.jsx` | — |
| Invoices | `invoice_controller.go` | `invoice_model.go` | private | `api/invoices.js`, `hooks/useInvoices.js` | `Invoices.jsx`, `InvoiceDetail.jsx`, `InvoiceEditor.jsx`, `components/invoice/InvoiceDocument.jsx` + `InvoicePdfDownload*.jsx` | share-payment-link button on `InvoiceDetail.jsx` |
| Expenses + receipts | `expense_controller.go` | `expense_model.go` (`receipt_url`) | private (`/expenses/:id/receipt`) | `api/expenses.js`, `hooks/useExpenses.js` | `Expenses.jsx` | `platform/storage` (local/S3, `logos/` public, `receipts/` private) |
| Payments + links | `payment_controller.go` | `payment_model.go`, `payment_link_model.go` | private + public pay + webhooks | `api/payments.js`, `api/publicPay.js`, `hooks/usePayments.js` | `Payments.jsx`, `PublicPay.jsx` | `platform/mail` payment-link template, links are IDR-only, drafts blocked |
| Gateway relay | `gateway_controller.go`, `gateway_webhook_controller.go` | `gateway_project_model.go`, `gateway_transaction_model.go` | `gateway_routes.go` (`GatewayAuth`, API key) | `api/gateway.js`, `hooks/useGatewayAdmin.js` | `AdminGateway.jsx` | `platform/gateway` (provider iface), `platform/relay`, `platform/midtrans`, `platform/nowpayments` |
| Dashboard | `dashboard_controller.go` | `dashboard_model.go` | private | `api/dashboard.js`, `hooks/useDashboard.js` | `Dashboard.jsx`, `components/dashboard/DashboardCharts.jsx` | — |
| Reports | `report_controller.go` | `report_model.go` | private | `api/reports.js`, `hooks/useReports.js` | `Reports.jsx`, `components/reports/ReportsCharts.jsx` | — |
| Settings + logo | `settings_controller.go` | `settings_model.go` | private (`/settings/logo`) | `api/settings.js`, `hooks/useSettings.js` | `Settings.jsx` | logo via `platform/storage` (public) |
| AI | `ai_controller.go` | `ai_model.go` | private (501 without `GEMINI_API_KEY`) | `api/ai.js` (used directly by pages) | `Dashboard/InvoiceEditor/InvoiceDetail/Expenses.jsx` | `platform/ai` |
| App config (public) | `config_controller.go` | — | public (`/config`) | `api/config.js`, `hooks/useConfig.js` | `Landing.jsx` (public site), branding for SPA shell | — |
| Shell providers | — | — | — | — | — | `context/LangContext.jsx` (i18n), `context/ThemeContext.jsx`, `context/UIContext.jsx` (toasts/modals) |

## Cross-cutting

- **Mail**: `platform/mail` (`mailer.go`, `templates.go`, `templates/*.html`) → queued into `mail_outbox_model.go` → sent by `platform/outbox` worker. Render HTML at enqueue time; envelope stays `multipart/alternative`.
- **Audit**: `app/controllers/audit_helper.go` + `audit_log_model.go`; CSRF middleware writes the audit trail (see P3).
- **Request guards** (all in `pkg/middleware` + `pkg/routes/versioning.go`): `AuthRequired`, `GatewayAuth`, CSRF double-submit, version-guard, `RequireRoles`, idempotency keys (`idempotency_model.go`, `app/queries/idempotency_query.go`), pagination (`?page/?per_page` + `meta`).
- **Cache** (dual-backend: Redis when `REDIS_HOST` set, memory otherwise): `platform/cache` (`session_store.go` sessions, `aggregates.go` dashboard/reports with singleflight), invalidated from controllers via `cache_helper.go` (`invalidateAggregates`).
- **DB/schema**: `platform/database` (GORM `AutoMigrate`, backend health, `migrations.go` reversible registry + `POST /admin/migrate/down` rollback) with a forward-only version guard via `schema_migration_model.go` — a newer DB than the binary refuses to start.
- **Health/ops**: `app/controllers/health_controller.go`, `pkg/routes/health_routes.go` (`/healthz`, `/readyz`), `metrics_routes.go` (`/metrics`), `not_found_route.go` (JSON 404 for unknown `/api/*`), `uploads_route.go` (`MountUploads`: local `/uploads/logos/*` only — receipts stay behind the auth proxy), `swagger_route.go` (`/swagger`), `VERSION` file (read once at startup).
- **Captcha**: `platform/captcha` (Turnstile) — enforced on login/register.
- **Docs**: Swagger annotations → `swag init` → committed `docs/` (`swagger.json/yaml`, `docs/docs.go`); `docs/API_DOCS.md` is a stub.

## Flow tests (behavior contracts)

`pkg/routes/routes_test.go` (`TestMain`, in-memory DB) plus:
`flow_test.go`, `public_routes_test.go`, `private_routes_test.go`,
`gateway_test.go`, `security_test.go`, `hardening_test.go`,
`versioning_test.go`, `pagination_test.go`, `idempotency_test.go`,
`metrics_test.go`, `storage_test.go`. Test data fixtures live in
`pkg/routes/data/`. When behavior changes, extend the matching flow test.
