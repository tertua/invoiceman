# AGENTS.md

## Start Here

- Read `docs/MODULE_MAP.md` before searching for an owner file. It maps each domain's request path and CI checks that all owner files and references stay current.
- The repo is a Go 1.27 Fiber API at the root and a Vite/React SPA in `web/`; entrypoints are `main.go` and `web/src/main.jsx`.
- `opencode.json` loads this file as the repository instruction source.

## Architecture And Contracts

- Request flow is `web/src/pages` -> `web/src/hooks` -> `web/src/api` -> `pkg/routes`/middleware -> `app/controllers` -> `app/queries` -> `platform`/`pkg`. Keep GORM/SQL detail in `app/queries`, not controllers.
- Controllers obtain the DB through `database.OpenDBConnection()` and request identity through `utils.CurrentUserID` / `utils.CurrentServiceProject`; do not re-parse JWTs or API keys there.
- New payment/gateway providers implement `platform/gateway.Gateway` and are registered in `main.go`.
- Register API prefixes in this order: `/api/v1` before legacy `/api`; within each prefix, public routes, gateway routes, then private routes. Gateway routes use API-key `GatewayAuth`, not cookie sessions.
- Success responses expose data keys directly through `utils.OK`; failures use `{"error":{"message","details"}}` through `utils.Fail`. DELETE and logout return `204` with no body. Backend messages are stable English; the SPA translates API keys.
- Auth mutations using session cookies require the CSRF double-submit header. Money-moving mutations require an `Idempotency-Key`.
- Empty `SQL_DSN` uses auto-created SQLite at `SQLITE_PATH` (default `./data/invoiceman.db`); only `postgres://`/`postgresql://` DSNs are supported. Schema is GORM `AutoMigrate` at startup; do not add migration files or raw SQL outside `platform/database`.
- Empty `REDIS_HOST` selects in-memory sessions, rate limits, and aggregate cache; setting it selects Redis. Features must work with either backend. Optional PG/Redis integration tests use `INVOICEMAN_TEST_PG_DSN` and `INVOICEMAN_TEST_REDIS_ADDR`; the default suite needs neither service.
- `.env` is autoloaded for local runs. The Docker image contains only the binary, so inject secrets at runtime; never bake `.env` into an image.
- Dates use `YYYY-MM-DD` and the helpers in `pkg/utils/date.go`. AI routes return `501` without `GEMINI_API_KEY`; online payment links are IDR-only and drafts cannot create them.

## Commands

- `go test ./...` runs the default backend suite with in-memory SQLite; focus route behavior with `go test ./pkg/routes/ -run TestName -v`.
- `make test` runs clean, `gocritic`, `gosec`, `golangci-lint`, and covered tests. `make build` depends on it. `make run` runs `swag init`, builds, and serves on port 5000.
- After changing Swagger annotations/controllers, run `swag init`; generated files under `docs/` are committed.
- Frontend: from `web/`, run `npm ci`, then `npm run lint`, `npm run build`; `make web.check` runs lint, build, and strict bundle checks.
- Run `npm --prefix web run check:map` after adding or renaming mapped owner files. Run `npm --prefix web run check:bundles` for advisory checks; strict mode requires a build first.
- `VERSION` is the version source of truth. Change it, then run `npm --prefix web run sync:version`; never edit `web/package.json`'s version directly.

## Frontend Constraints

- Axios is imported only by `web/src/api/http.js`; it normalizes failures to `{status, message, details}`. `@` aliases `web/src`; Vite dev server runs on 5173 and proxies `/api` and `/uploads` to `localhost:5000`.
- Keep `@react-pdf/renderer` imports only in `InvoiceDocument.jsx` and `InvoicePdfDownloadContent.jsx`; keep `recharts` imports only in `DashboardCharts.jsx`, `ClientCharts.jsx`, and `ReportsCharts.jsx`. Bundle budgets live in `web/bundle-baseline.json`.
- Shared format helpers in `web/src/lib/utils.js` must not throw. User-visible strings belong in `web/src/lib/i18n.js` (`en` and `id`), not hardcoded in components.
- Use two-space indentation for frontend/config and tabs for Go. Behavior changes should extend the matching flow test in `pkg/routes/*_test.go`, especially for public, webhook, auth, and payment endpoints.
