# Repository guide

For domain ownership and request flow, use `docs/MODULE_MAP.md`; update it when owner files or domain wiring change (`npm --prefix web run check:map` is enforced in CI).

## Architecture and contracts

- `main.go` wires the Fiber API, migrations, gateways, background worker, and optional SPA; `web/src/main.jsx` is the React/Vite entrypoint. Follow the request path and domain rows in `docs/MODULE_MAP.md` before changing a feature.
- Keep SQL/GORM queries in `app/queries`; raw SQL is restricted to `platform/database`. Queries and migrations must work with both SQLite and PostgreSQL (`.github/workflows/dialect-check.yml`).
- Startup runs GORM `AutoMigrate`; rollback steps live in `platform/database/migrations.go`. New rollback steps must use backend-agnostic GORM migrator calls and undo only their own additions.
- Route order is significant: register `/api/v1` before legacy `/api`, and register public, gateway, then private routes within each prefix (`pkg/routes/versioning.go`). Gateway routes authenticate by API key; session routes use middleware-provided identity (`utils.CurrentUserID` / `utils.CurrentServiceProject`). Session-cookie mutations require `X-CSRF-Token`; money-moving mutations require an `Idempotency-Key`.
- Preserve the API envelope: successes use `utils.OK` with direct data fields; errors use `utils.Fail`'s `error.message`/`details` shape. Client-facing strings are stable English and translated in the SPA. Dates use `YYYY-MM-DD` via `pkg/utils/date.go`.
- Add payment providers by implementing `platform/gateway.Gateway` and registering them in `main.go`.
- `.env` is loaded automatically for local runs; empty `SQL_DSN` selects auto-created SQLite and empty `REDIS_HOST` selects in-memory stores.

## Development and verification

- Go toolchain is pinned to 1.27.1 (`go.mod`). `go test ./...` runs the default suite without external services; focus a route flow with `go test ./pkg/routes -run TestName -v`. Flow tests use in-memory SQLite. PostgreSQL and Redis integration tests are opt-in via `INVOICEMAN_TEST_PG_DSN` (empty scratch DB) and `INVOICEMAN_TEST_REDIS_ADDR`.
- `make test` runs clean, gocritic, gosec, golangci-lint, then coverage tests. `make build` depends on `make test`; `make run` runs `swag init`, builds, then starts the API.
- Swagger annotations changed: run `swag init`; generated `docs/` files are committed.
- Frontend CI uses Node 22: `npm ci`, then `npm --prefix web run lint`, `npm --prefix web run build`, and `npm --prefix web run check:bundles:strict`. `make web.check` runs these frontend checks.
- `VERSION` is canonical; after changing it run `npm --prefix web run sync:version` to sync `web/package.json`.

## Frontend constraints

- Axios imports belong only in `web/src/api/http.js`. Vite aliases `@` to `web/src` and proxies `/api` and `/uploads` to `localhost:5000` on port 5173.
- Keep `@react-pdf/renderer` imports in `InvoiceDocument.jsx` and `InvoicePdfDownloadContent.jsx`; keep `recharts` in the Dashboard, Client, and Reports chart components. Bundle limits are checked by `check:bundles:strict`.
- User-visible strings belong in `web/src/lib/i18n.js` for both `en` and `id`, not hardcoded in components.
