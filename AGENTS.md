# Repository guide

For domain ownership and request flow, use `docs/MODULE_MAP.md`; update it when owner files or domain wiring change (`npm --prefix web run check:map` is enforced in CI).

## File size limits (enforced, not advisory)

One file = one responsibility. `npm --prefix web run check:size` fails CI when a file grows past its recorded size.

- New files must fit the budget: controllers ≤400, queries ≤300, models ≤200, route tests ≤400, pages ≤250, components ≤250, hooks/api ≤150, per-language i18n ≤900 lines.
- Never append to a file near its limit — split it first, the way the repo already does: controllers by route area (`gateway_*_controller.go`), pages by extracting `components/*` cards (`InvoiceDetail.jsx` split), flow tests by domain (`flow_*_test.go`), i18n by language (`i18n.en.js`/`i18n.id.js`).
- After an intentional split, lower that file's entry in `scripts/file-size-baseline.json` (never raise it).
- `docs/docs.go` is generated and exempt.

## Architecture and contracts

- `main.go` wires the Fiber API, migrations, gateways, background worker, and optional SPA; `web/src/main.jsx` is the React/Vite entrypoint. Follow the request path and domain rows in `docs/MODULE_MAP.md` before changing a feature.
- Keep SQL/GORM queries in `app/queries`; raw SQL is restricted to `platform/database`. Queries and migrations must work with both SQLite and PostgreSQL (`.github/workflows/dialect-check.yml`).
- Startup runs GORM `AutoMigrate`; rollback steps live in `platform/database/migrations.go`. New rollback steps must use backend-agnostic GORM migrator calls and undo only their own additions.
- Route order is significant: register `/api/v1` before legacy `/api`, and register public, gateway, then private routes within each prefix (`pkg/routes/versioning.go`). Gateway routes authenticate by API key; session routes use middleware-provided identity (`utils.CurrentUserID` / `utils.CurrentServiceProject`). Session-cookie mutations require `X-CSRF-Token`; money-moving mutations require an `Idempotency-Key`.
- Preserve the API envelope: successes use `utils.OK` with direct data fields; errors use `utils.Fail`'s `error.message`/`details` shape. Client-facing strings are stable English and translated in the SPA. Dates use `YYYY-MM-DD` via `pkg/utils/date.go`.
- Add payment providers by implementing `platform/gateway.Gateway` and registering them in `main.go`.
- AI text endpoints take language from the `X-Locale` header (`en`/`id`, default `en`) and currency from user settings or the invoice; build prompts with `aiLocale`/`languageDirective`/`currencyDirective` (`app/controllers/ai_controller.go`) so answers match UI language and money formatting (e.g. Bahasa Indonesia with `Rp160.000`, never default bare numbers to `$`). Cache generated AI text in `localStorage` scoped per user (+invoice/tone/language where relevant); regenerating overwrites the cache instead of calling Gemini again.
- Gemini 429 surfaces as `ErrRateLimited` → HTTP 429 (`ai.rateLimited`). The free-tier quota is tiny (limit 20 requests), so verify AI with single calls, never probe loops. `GEMINI_MODEL` default lives in `.env.example`.
- `.env` is loaded automatically for local runs; empty `SQL_DSN` selects auto-created SQLite and empty `REDIS_HOST` selects in-memory stores. Local settings live in gitignored `.env.build`, symlinked as `.env` (godotenv autoload only reads that exact name). Sessions are in-memory when `REDIS_HOST` is empty, so every BE restart invalidates all logins (expect 401s until re-login).

## Branching (main stable, dev active)

- `dev` is daily work. `main` is stable only — never commit directly to `main`.
- Promote `dev` → `main` only when stable (tests/lint pass) via fast-forward, never merge-commit or force-push:
  `make promote` (= `git fetch` + `git checkout main` + `git merge --ff-only origin/dev` + `git push origin main`).
- Keep history linear: `git pull --ff-only` / `git pull --rebase`; no `git merge --no-ff`, no `git push --force` on `main`/`dev`.
- Verify with `make check-flow` (`main` must be ancestor of `dev`, no merge commits in `main..dev`); CI enforces this in `branch-flow.yml`.

## Development and verification

- Go toolchain is pinned to 1.27.1 (`go.mod`). `go test ./...` runs the default suite without external services; focus a route flow with `go test ./pkg/routes -run TestName -v`. Flow tests use in-memory SQLite. PostgreSQL and Redis integration tests are opt-in via `INVOICEMAN_TEST_PG_DSN` (empty scratch DB) and `INVOICEMAN_TEST_REDIS_ADDR`.
- `make test` runs clean, gocritic, gosec, golangci-lint, then coverage tests. After any `go.mod`/`go.sum` change, run `govulncheck ./...` before committing (CI also runs it on push/PR plus weekly on schedule). `make build` depends on `make test`; `make run` runs `swag init`, builds, then starts the API. Lint scope is intentional: `.golangci.yml` excludes vendored/test noise and the `critic` target lists packages explicitly with `hugeParam,rangeValCopy` disabled — don't revert to bare `./...`.
- Swagger annotations changed: run `swag init`; generated `docs/` files are committed.
- Frontend CI uses Node 22: `npm ci`, then `npm --prefix web run lint`, `npm --prefix web run build`, and `npm --prefix web run check:bundles:strict`. `make web.check` runs these frontend checks.
- `VERSION` is canonical; after changing it run `npm --prefix web run sync:version`, which syncs `web/package.json` and ensures `web/CHANGELOG.md` has a `## [vX.Y.Z]` section (auto-inserts a stub, then fill in the notes). CI (`version-check.yml`) runs the same check and fails on a missing or still-stubbed section.

## Local dev run (this machine)

- BE: `make dev-be` (builds to `/tmp/opencode/invoiceman`, serves `:5000`). Health: `curl localhost:5000/healthz`. Restart after BE code changes: rebuild, `pkill -f /tmp/opencode/invoiceman`, start again.
- FE: `make dev-fe` (vite dev on `:5173`, proxies `/api`+`/uploads` to BE). Hot-reloads; no restart needed after FE changes.
- Scratch binaries/logs go ONLY to `/tmp/opencode/` (pre-approved); never the repo root or bare `/tmp`.

## Frontend constraints

- Axios imports belong only in `web/src/api/http.js`. Vite aliases `@` to `web/src` and proxies `/api` and `/uploads` to `localhost:5000` on port 5173.
- Keep `@react-pdf/renderer` imports in `InvoiceDocument.jsx` and `InvoicePdfDownloadContent.jsx`; keep `recharts` in the Dashboard, Client, and Reports chart components. Bundle limits are checked by `check:bundles:strict`.
- User-visible strings belong in `web/src/lib/i18n.en.js` (`en`) and `web/src/lib/i18n.id.js` (`id`) — `web/src/lib/i18n.js` is only the re-export entrypoint — not hardcoded in components.
- Money crosses the API as decimal **strings** (`models.Money`); Recharts `Pie` silently draws nothing for strings. Route pie data through `chartNumbers` (`web/src/lib/chartData.js`); `check:charts` enforces it and `npm test` covers the helper.
