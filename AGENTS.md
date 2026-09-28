# Repository guide

For domain ownership and request flow, use `docs/MODULE_MAP.md`; update it when owner files or domain wiring change (`npm --prefix web run check:map` is enforced in CI).

## Chat output (assistant)

- Always respond in Bahasa Indonesia, regardless of the language used in code, commits, or docs.
- Keep answers short and direct (CLI-friendly); no unnecessary preamble or postamble.
- When a request is ambiguous (scope, phrasing, defaults, or direction), resolve it with the `question` tool instead of assuming — offer concrete options and a recommended first choice.
- Non-API-docs comments must stay on one line; only Swagger doc annotations may span multiple lines.

## File size limits (enforced, not advisory)

One file = one responsibility. `npm --prefix web run check:size` fails CI; run it locally after any Go/JS file change.

- Two hard rules: a file already listed in `scripts/file-size-baseline.json` may **never grow past its recorded entry** — even when it sits far below its category budget (e.g. `admin_model.go` is capped at its entry of 33, not at 200). A file *not* in the baseline (new or post-split) must fit its budget: controllers ≤400, queries ≤300, models ≤200, route tests ≤400, pages ≤250, components ≤250, hooks/api ≤150, per-language i18n ≤900 lines.
- So appending to almost any existing file fails CI. Split first, the way the repo already does: controllers by route area (`gateway_*_controller.go`), pages by extracting `components/*` cards (`InvoiceDocument.jsx`/`PublicPay.jsx` splits), flow tests by domain (`flow_*_test.go`), i18n by language (`i18n.en.js`/`i18n.id.js` + `i18n.*.settings.js`).
- After an intentional split, lower that file's entry in `scripts/file-size-baseline.json` (never raise it). `docs/docs.go` is generated and exempt.

## Architecture and contracts

- `main.go` wires the Fiber API, migrations, gateways, background worker, and optional SPA (Vite MPA: `web/src/main.jsx` is the product entry, `web/src/admin.jsx` the admin console under `/admin`); follow the request path and domain rows in `docs/MODULE_MAP.md` before changing a feature.
- Keep SQL/GORM queries in `app/queries`; raw SQL is restricted to `platform/database`. Queries and migrations must work with both SQLite and PostgreSQL (`.github/workflows/dialect-check.yml`).
- Startup runs GORM `AutoMigrate`; rollback steps live in `platform/database/migrations.go`. New rollback steps must use backend-agnostic GORM migrator calls and undo only their own additions.
- Route order is significant: register `/api/v1` before legacy `/api`, and register public, gateway, then private routes within each prefix (`pkg/routes/versioning.go`). Gateway routes authenticate by API key; session routes use middleware-provided identity (`utils.CurrentUserID` / `utils.CurrentServiceProject`). Session-cookie mutations require `X-CSRF-Token`; money-moving mutations require an `Idempotency-Key`.
- Preserve the API envelope: successes use `utils.OK` with direct data fields; errors use `utils.Fail`'s `error.message`/`details` shape. Client-facing strings are stable English and translated in the SPA. Dates use `YYYY-MM-DD` via `pkg/utils/date.go`.
- Add payment providers by implementing `platform/gateway.Gateway` and registering them in `main.go`.
- AI text endpoints take language from the `X-Locale` header (`en`/`id`, default `en`) and currency from user settings or the invoice; build prompts with `aiLocale`/`languageDirective`/`currencyDirective` (`app/controllers/ai_controller.go`) so answers match UI language and money formatting (e.g. Bahasa Indonesia with `Rp160.000`, never default bare numbers to `$`). Cache generated AI text in `localStorage` scoped per user (+invoice/tone/language where relevant); regenerating overwrites the cache instead of calling Gemini again.
- Gemini 429 surfaces as `ErrRateLimited` → HTTP 429 (`ai.rateLimited`). The free-tier quota is tiny (limit 20 requests), so verify AI with single calls, never probe loops. Code default model is `gemini-2.0-flash` (`pkg/configs`); `.env.example` overrides it.
- `.env` is loaded automatically for local runs; empty `SQL_DSN` selects auto-created SQLite and empty `REDIS_HOST` selects in-memory stores. Local settings live in gitignored `.env.build`, symlinked as `.env` (godotenv autoload only reads that exact name). Sessions are in-memory when `REDIS_HOST` is empty, so every BE restart invalidates all logins (expect 401s until re-login).

## Branching (dev active, main stable, master production)

- `dev` is daily work. `main` is stable only, `master` is production only — never commit directly to `main` or `master`.
- Promote `dev` → `main` only when stable (tests/lint pass) via fast-forward, never merge-commit or force-push:
  `make promote` (= `git fetch` + `git checkout main` + `git merge --ff-only origin/dev` + `git push origin main`).
- Release `main` → `master` with `make promote-prod` (ff-only push of `origin/main` to `master` + annotated tag `v$(VERSION)` when that tag does not exist yet; re-running is a no-op). Release order: `make promote && make promote-prod`.
- `VERSION` changes only as part of a release, never inside feat/fix commits. The agent picks the number from the change (`fix` → patch, `feat` → minor, breaking → major), then `npm --prefix web run sync:version` + real notes in `web/CHANGELOG.md` before promoting. `sync-version.mjs --check` (CI `version-check.yml`) rejects any bump that is not a single-component +1 with lower components reset (`0.6.1→0.6.2`, `0.6.2→0.7.0`, `0.7.0→1.0.0`) — jumps and downgrades fail the build.
- Keep history linear: `git pull --ff-only` / `git pull --rebase`; no `git merge --no-ff`, no `git push --force` on `dev`/`main`/`master`.
- Verify with `make check-flow` (`master` ancestor of `main`, `main` ancestor of `dev`, no merge commits in `master..dev`); CI enforces this in `branch-flow.yml`.
- Dependabot: PRs target `dev` with minor+patch grouped per ecosystem (`.github/dependabot.yml`); `.github/workflows/dependabot-auto-merge.yml` squash-merges them once every reported check is green (majors stay manual). The repo has merge commits disabled (squash/rebase only) so PR merges can never break `check-flow`.
- The repo is public and `dev`/`main`/`master` carry branch protection: no force-push, no deletion, linear history only, enforced for admins too. Fast-forward pushes (`make promote`, `make promote-prod`) stay allowed — anything else is rejected by GitHub.

## Development and verification

- Go toolchain is pinned to 1.27.1 (`go.mod`). `go test ./...` runs the default suite without external services; focus a route flow with `go test ./pkg/routes -run TestName -v`. Flow tests use in-memory SQLite. PostgreSQL and Redis integration tests are opt-in via `INVOICEMAN_TEST_PG_DSN` (empty scratch DB) and `INVOICEMAN_TEST_REDIS_ADDR`.
- Flow tests build fixtures through `pkg/routes/flow_fixtures_test.go` (`newInvoice`/`createInvoice`/`createClient`) — never hand-write invoice JSON in a test; `check:fixtures` fails on a raw `POST /api/invoices` literal. Cross-field invoice rules live once in `app/controllers/invoice_rules.go` (`validateInvoice`); add new invariants there, not inline at each call site.
- `make test` runs clean, gocritic, gosec, golangci-lint, then coverage tests — and `make build` depends on `make test`, so never use it for a quick binary (use `make dev-be`). After any `go.mod`/`go.sum` change, run `govulncheck ./...` before committing (CI also runs it on push/PR plus weekly on schedule). `make run` runs `swag init`, builds, then starts the API. Lint scope is intentional: `.golangci.yml` excludes vendored/test noise and the `critic` target lists packages explicitly with `hugeParam,rangeValCopy` disabled — don't revert to bare `./...`.
- Swagger annotations changed: run `swag init`; generated `docs/` files are committed.
- Cheap pre-push guards are plain Node scripts that need **no `npm ci`** (CI runs them bare): `npm --prefix web run check:map`, `check:size`, `check:fixtures`, and `node scripts/sync-version.mjs --check`.
- Frontend CI uses Node 22 (local may differ): `npm ci`, then in `web/` `npm run lint`, `npm test`, `npm run check:charts`, `npm run build`, `npm run check:bundles:strict`. `make web.check` runs all of those plus `check:fixtures`.
- `VERSION` is canonical; after changing it run `npm --prefix web run sync:version`, which syncs `web/package.json` and ensures `web/CHANGELOG.md` has a `## [vX.Y.Z]` section (auto-inserts a stub, then fill in the notes). CI (`version-check.yml`) runs the same check and fails on a missing or still-stubbed section.

## Local dev run (this machine)

- BE: `make dev-be` (builds to `/tmp/opencode/invoiceman`, serves `:5000`). Health: `curl localhost:5000/healthz`. Restart after BE code changes: rebuild, `pkill -f /tmp/opencode/invoiceman`, start again.
- FE: `make dev-fe` (vite dev on `:5173`, proxies `/api`+`/uploads` to BE). Hot-reloads; no restart needed after FE changes.
- Local state lives in gitignored `data/` (`SQLITE_PATH=./data/invoiceman.db`, `STORAGE_DIR=./data/uploads`) — survives BE restarts; test runs use in-memory SQLite instead.
- README's Quick start is Docker (`make docker.run` = Postgres + Redis + API) — for this machine use `make dev-be` / `make dev-fe`; README also documents the two deploy modes and the Turnstile build-time vs runtime key split.
- Scratch binaries/logs go ONLY to `/tmp/opencode/` (pre-approved); never the repo root or bare `/tmp`.

## Frontend constraints

- Axios imports belong only in `web/src/api/http.js`. Vite aliases `@` to `web/src` and proxies `/api` and `/uploads` to `localhost:5000` on port 5173.
- Keep `@react-pdf/renderer` imports in `InvoiceDocument.jsx` and `InvoicePdfDownloadContent.jsx`; keep `recharts` in the Dashboard, Client, and Reports chart components. Bundle limits are checked by `check:bundles:strict`.
- User-visible strings belong in `web/src/lib/i18n.en.js` (`en`) and `web/src/lib/i18n.id.js` (`id`) — `web/src/lib/i18n.js` is only the re-export entrypoint — not hardcoded in components.
- Money crosses the API as decimal **strings** (`models.Money`); Recharts `Pie` silently draws nothing for strings. Route pie data through `chartNumbers` (`web/src/lib/chartData.js`); `check:charts` enforces it and `npm test` covers the helper.
