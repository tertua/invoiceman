# Repository instructions

`opencode.json` loads this file. For domain ownership, read `docs/MODULE_MAP.md` first; its CI check requires owner files and references to stay mapped.

## Structure and contracts

- Go 1.27 Fiber API entrypoint: `main.go`; React/Vite SPA entrypoint: `web/src/main.jsx`. Request flow and domain owners are mapped in `docs/MODULE_MAP.md`.
- Keep SQL/GORM domain queries in `app/queries`, not controllers. Raw SQL is permitted only in `platform/database` (enforced by `.github/workflows/dialect-check.yml`); support both SQLite and PostgreSQL.
- Schema upgrades use GORM `AutoMigrate` at startup; rollback steps are registered in `platform/database/migrations.go`. New rollback steps must use backend-agnostic GORM migrator calls and only undo their own version's additions.
- Authentication identity is set by middleware: use `utils.CurrentUserID` / `utils.CurrentServiceProject`, not custom token parsing. Session-cookie mutations require the CSRF double-submit header; money-moving mutations require an `Idempotency-Key`.
- Route registration order is significant: register `/api/v1` before legacy `/api`, and within each prefix register public, gateway, then private routes. Gateway routes use API-key auth, not session cookies.
- Add payment providers by implementing `platform/gateway.Gateway` and registering it in `main.go`.
- Keep API success fields direct via `utils.OK`; errors use `utils.Fail`'s `error.message`/`details` shape. Backend messages are stable English for client-side translation; DELETE/logout return 204. Dates use `YYYY-MM-DD` and `pkg/utils/date.go`.
- Empty `SQL_DSN` selects auto-created SQLite (`SQLITE_PATH`, default `./data/invoiceman.db`); only PostgreSQL DSNs are supported. Empty `REDIS_HOST` selects in-memory stores; configured Redis selects shared stores. Default tests need neither service; optional integration tests use `INVOICEMAN_TEST_PG_DSN` and `INVOICEMAN_TEST_REDIS_ADDR`.
- `.env` is autoloaded for local runs. Docker images contain the binary (the embedded-SPA variant is `Dockerfile.dev`); inject secrets at runtime, never into the image.

## Commands and generated files

- Backend: `go test ./...`; focus a test with `go test ./pkg/routes/ -run TestName -v`. `make test` runs clean, gocritic, gosec, golangci-lint, and coverage tests; `make build` depends on `make test`. `make run` runs `swag init`, builds, then serves on port 5000.
- Swagger annotations/controllers changed: run `swag init`; generated `docs/` files are committed.
- Frontend (Node 22 in CI): `npm ci` then `npm --prefix web run lint` and `npm --prefix web run build`. `make web.check` additionally runs strict bundle-budget checks (after build).
- When adding/renaming domain owner files, update `docs/MODULE_MAP.md` and run `npm --prefix web run check:map` (CI enforces it).
- `VERSION` is canonical; after changing it run `npm --prefix web run sync:version` to update `web/package.json`.

## Frontend boundaries

- Axios imports belong only in `web/src/api/http.js`; Vite aliases `@` to `web/src` and proxies `/api` and `/uploads` to `localhost:5000` on port 5173.
- Keep `@react-pdf/renderer` imports in `InvoiceDocument.jsx` and `InvoicePdfDownloadContent.jsx`; `recharts` imports in `DashboardCharts.jsx`, `ClientCharts.jsx`, and `ReportsCharts.jsx`. Bundle budgets are in `web/bundle-baseline.json`.
- User-visible strings belong in `web/src/lib/i18n.js` (`en` and `id`), not hardcoded in components.
