# AGENTS.md

File owners per domain live in `docs/MODULE_MAP.md` — read it before grepping the codebase. CI `module-map-check` fails on drift (run `npm --prefix web run check:map` after adding/renaming owner files).

## Layout

- Go/Fiber API at root (`main.go`); Vite/React SPA in `web/` (`web/src/main.jsx`, routes in `web/src/routes.jsx`).
- Backend: `app/` (controllers, models, queries) → `platform/` (database, cache, gateway, mail, ai) + `pkg/` (routes, middleware, utils, repository). Controllers call `database.OpenDBConnection()` and read identity via `utils.CurrentUserID` / `utils.CurrentServiceProject` — never re-parse JWT/API keys in controllers.
- New gateway providers implement `platform/gateway.Gateway` and register in `main.go`.

## Backend

- Go 1.27. API under `/api`; Swagger at `/swagger/index.html`. After changing annotations/controllers run `swag init` (`docs/` is committed).
- Auth: cookie (or Bearer) JWT via `AuthRequired`; expired access tokens refresh transparently from the refresh cookie. Empty `REDIS_HOST` = in-memory sessions; set = Redis. Service relay (`/api/gateway/*`) uses `GatewayAuth` (API key), never sessions.
- Route registration order in `main.go` matters: `GatewayRoutes` before `PrivateRoutes`, or service routes get forced through cookie sessions.
- Responses: success = data keys directly via `utils.OK`; errors = `{"error":{"message","details"}}` via `utils.Fail`. Backend messages are stable English; DELETE/logout return `204` with no body.
- Dates are `YYYY-MM-DD` strings: use `pkg/utils/date.go` (`DateLayout`, `ParseDate`/`ParseRequiredDate`/`FormatDate`/`FormatTime`).
- DB: empty `SQL_DSN` = auto-created SQLite at `SQLITE_PATH`; `postgres://` = PostgreSQL; anything else is a startup error (no MySQL). Default path is `./data/invoiceman.db` — a root-level `invoiceman.db` is NOT used (never create one; `sqlite3.connect` probes create empty files). Schema is GORM `AutoMigrate` at startup — no migration files/CLI.
- Dual-backend contract (SQLite/PostgreSQL, memory/Redis): no feature may require Redis or PostgreSQL unconditionally. Redis-backed stores (sessions, limiters, aggregate cache) fall back to in-memory with identical behaviour when `REDIS_HOST` is empty. No raw SQL outside `platform/database` (CI `dialect-check` fails the build) — queries stay in `app/queries` via GORM. Schema rollbacks live in the `platform/database` registry (`POST /admin/migrate/down`); forward upgrades re-apply on startup. Env-gated integration tests cover the full-resource path and skip otherwise (`INVOICEMAN_TEST_PG_DSN` = scratch PG database, `INVOICEMAN_TEST_REDIS_ADDR` = host:port) — the default suite never needs services.
- Live dev processes: BE runs as `/tmp/opencode/invoiceman` (CWD = repo root, env `SERVER_HOST=0.0.0.0 SERVER_PORT=5000`, log `/tmp/opencode/be.log`); FE vite runs from `web/` (log `/tmp/opencode/fe.log`). After backend changes, rebuild + restart BE (SIGTERM is graceful) — a stale binary only serves `/api/*` while current FE speaks `/api/v1`, which surfaces as 401 "sesi berakhir" on every login. Any restart wipes in-memory sessions (users must log in again) and is also required after a `VERSION` bump (version is read once at startup; no rebuild needed for that).
- `.env` autoloads; local dev needs no services. Docker build (`Dockerfile` copies root `.env` into scratch image) requires a root `.env` — start from `.env.example`.
- Roles are `admin`/`user`/`moderator`; `RequireRoles("admin", "user")` (e.g. `PATCH /settings`) intentionally excludes moderators.
- AI endpoints return `501` without `GEMINI_API_KEY`. Online payment links are IDR-only and blocked for draft invoices.

## Verify

- Backend: `go test ./...` (in-memory SQLite + sessions via `pkg/routes/routes_test.go` `TestMain` — no Postgres/Redis needed). Focused: `go test ./pkg/routes/ -run TestClientInvoiceFlow -v` (also `TestAuthFlow`, `TestGatewayRelayFlow`, `TestDraftOnlinePaymentBlocked`, ...).
- `make test` = `clean` + `gocritic` + `gosec` + `golangci-lint` + coverage; those tools may be missing. `make build`/`make run` depend on it; `make run` also runs `swag init` + `build`, then serves on `5000`.
- Frontend from `web/`: `npm run lint`, `npm run build`. Full gate: `make web.check` (lint + build + `check:bundles:strict`). No frontend test script.
- Bundle boundaries (enforced by `check:bundles`): `@react-pdf/renderer` only in `components/invoice/InvoiceDocument.jsx` + `InvoicePdfDownloadContent.jsx`; `recharts` only in `DashboardCharts.jsx`, `ClientCharts.jsx`, `ReportsCharts.jsx`. Entry/chunk budgets in `web/bundle-baseline.json`.

## Conventions

- One domain per controller file; `Get*` for single-resource getters (`GetSettings`, `GetDashboard`).
- Frontend: `web/src/api/` + `web/src/hooks/` are one module per domain; axios is imported only in `web/src/api/http.js` (normalizes failures to `{status, message, details}`). `@` aliases `web/src`. Vite `5173` proxies `/api` to `localhost:5000`.
- Never throw in shared format helpers (`web/src/lib/utils.js`) — return `""`/`"—"`. User-visible strings go through `web/src/lib/i18n.js` `api.*` keys (en+id); never hardcode display text.
- `VERSION` is the single source of truth; sync with `npm --prefix web run sync:version` (CI `version-check` fails on drift). Never bump `web/package.json` by hand.
- 2-space indent in frontend/config, tabs in Go (`.editorconfig`). When behavior changes, add/extend a flow test in `pkg/routes/*_test.go` and give public/webhook endpoints a data-exposure review.
