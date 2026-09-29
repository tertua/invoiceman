# Repository guide

Read first: this file, then `docs/MODULE_MAP.md` (domain owners + request path), then `main.go` → `pkg/routes/versioning.go`. Don't re-derive ownership by grepping.

## Chat output (assistant)

- Always respond chat in Bahasa Indonesia, regardless of the language used in code, commits, spawn_agent, comments or docs.
- Keep answers short and direct (CLI-friendly); no unnecessary preamble or postamble.
- When a request is ambiguous (scope, phrasing, defaults, or direction), resolve it with the `question` tool instead of assuming — offer concrete options and a recommended first choice.
- Non-API-docs comments must stay on one line; only Swagger doc annotations may span multiple lines.

## File size limits (enforced, not advisory)

One file = one responsibility. `npm --prefix webui run check:size` fails CI; run it after any Go/JS change.

- A file listed in `scripts/file-size-baseline.json` may **never grow past its recorded entry** (even far below budget, e.g. `admin_model.go` capped at 33, not 200). New files must fit budgets: controllers ≤400, queries ≤300, models ≤200, route tests ≤400, pages/components ≤250, hooks/api ≤150, per-language i18n ≤900.
- So split before appending: controllers by route area (`gateway_*_controller.go`), pages by extracting `components/*` cards, flow tests by domain (`flow_*_test.go`), i18n by language (`i18n.en/id.js` + `i18n.*.settings.js`).
- After a split, lower that file's baseline entry (never raise). `docs/docs.go` is generated and exempt.

## Architecture and contracts

- Queries live in `app/queries` (never inline in controllers); raw SQL only in `platform/database`. Both SQLite and PostgreSQL must work (CI `dialect-check.yml`). Startup runs GORM `AutoMigrate`; rollback steps in `platform/database/migrations.go` must be backend-agnostic and undo only their own additions.
- Route order matters: `/api/v1` before legacy `/api`; public → gateway → private within each prefix (`pkg/routes/versioning.go`). Gateway = API key; session routes use `utils.CurrentUserID` / `utils.CurrentServiceProject`. Session-cookie mutations need `X-CSRF-Token`; money-moving mutations need `Idempotency-Key`.
- Envelope: success via `utils.OK` (direct data fields), errors via `utils.Fail` (`error.message`/`details`, stable English, translated in MPA). Dates `YYYY-MM-DD` (`pkg/utils/date.go`). Money crosses the API as decimal **strings** (`models.Money`).
- New payment providers implement `platform/gateway.Gateway`, registered in `main.go`. Swagger annotations changed → run `swag init`; generated `docs/` is committed.
- AI endpoints: language from `X-Locale` (`en`/`id`, default `en`), currency from settings/invoice — build prompts with `aiLocale`/`languageDirective`/`currencyDirective` (`ai_controller.go`), never bare `$` defaults. Cache AI text in `localStorage` per user (+invoice/tone/language); regenerate overwrites instead of re-calling. Gemini 429 → HTTP 429 (`ai.rateLimited`); quota is tiny (20), so verify with single calls. Default model `gemini-2.0-flash` (`pkg/configs`); `.env.example` overrides.
- `.env` autoloads (godotenv reads only that exact name; local settings in gitignored `.env.build` symlinked as `.env`). Empty `SQL_DSN` = auto SQLite, empty `REDIS_HOST` = in-memory stores. In-memory sessions die on every BE restart (expect 401s until re-login).

## Branching (dev active, main stable, master production)

- Never commit to `main`/`master`. Promote with `make promote` (dev→main, ff-only) then `make promote-prod` (main→master + idempotent `v$(VERSION)` tag). Verify with `make check-flow`; no merge commits, no force-push on `dev`/`main`/`master` (`git pull --ff-only` / `--rebase`).
- `VERSION` changes only in releases, never in feat/fix commits (`fix`→patch, `feat`→minor, breaking→major), then `npm --prefix webui run sync:version` + real notes in `webui/CHANGELOG.md` (stub sections fail CI `version-check.yml`). Version bumps are pre-approved: commit and push directly.

## Development and verification

- Go 1.27.1 pinned. `go test ./...` needs no external services; focus with `go test ./pkg/routes -run TestName -v` (in-memory SQLite). PG/Redis integration is opt-in via `INVOICEMAN_TEST_PG_DSN` / `INVOICEMAN_TEST_REDIS_ADDR`.
- Flow tests must build invoices via `pkg/routes/flow_fixtures_test.go` (`newInvoice`/`createInvoice`/`createClient`); `check:fixtures` fails on raw `POST /api/invoices`. Cross-field invoice rules live once in `app/controllers/invoice_rules.go` (`validateInvoice`).
- `make test` = clean + gocritic + gosec + golangci-lint + coverage; `make build` depends on it, so use `make dev-be` for a quick binary. Lint scope is explicit in `Makefile` (`critic` lists packages, `hugeParam,rangeValCopy` off) — don't revert to bare `./...`. After `go.mod`/`go.sum` changes run `govulncheck ./...`.
- Cheap pre-push guards need no `npm ci`: `check:map`, `check:size`, `check:fixtures`, `node scripts/sync-version.mjs --check`. FE full check: `make webui.check` (lint, test, charts, fixtures, build, strict bundles).
- Local: `make dev-be` (binary to `/tmp/opencode/tupay`, `:5000`, `curl localhost:5000/healthz`) + `make dev-fe` (Vite `:5173`, proxies `/api`+`/uploads`, hot-reload). Rebuild + restart BE after BE changes. Scratch files only under `/tmp/opencode/`; local state in gitignored `data/` (tests use in-memory SQLite). Never `pkill -f` with a literal pattern — disguise one char (`[v]ite`), or you kill your own session.

## Frontend constraints (all CI-enforced)

- Axios imports only in `webui/src/api/http.js`; `@` aliases `webui/src`. User strings go in `i18n.en.js`/`i18n.id.js` (+ `i18n.*.settings.js`) — `i18n.js` is only the re-export.
- `@react-pdf/renderer` only in `InvoiceDocument.jsx` + `InvoicePdfDownloadContent.jsx`; `recharts` only in Dashboard/Client/Reports chart components (`check:bundles:strict`). Pie data must go through `chartNumbers` (`lib/chartData.js`) — strings render nothing (`check:charts`).
