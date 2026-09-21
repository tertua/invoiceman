# AGENTS.md

## Repository Shape

- The repository contains two independent apps: the Go/Fiber API at the root and the Vite/React SPA in `web/`.
- Backend entrypoint is `main.go`; frontend entrypoint is `web/src/main.jsx`, with routing in `web/src/routes.jsx`.
- Keep the application boundaries: `app/` contains business logic (`controllers`, `models`, `queries`) and must not import infrastructure; `pkg/` contains project wiring; `platform/` contains database/cache infrastructure. See the directory-level `BUSINESS_LOGIC.md`, `PROJECT_SPECIFIC.md`, and `PLATFORM_LEVEL.md` files before moving code.

## Backend

- Use Go 1.27. The API is rooted at `/api`; Swagger is at `/swagger/index.html`.
- Regenerate committed Swagger output in `docs/` with `swag init` after changing API annotations or controllers. `make run` and `make docker.run` also invoke it.
- `POST /api/auth/register` and `/login` establish HttpOnly JWT session cookies. Private routes use `AuthRequired`; Redis is used when `REDIS_HOST` is set, otherwise sessions are in memory. Bearer authentication is also accepted.
- API successes expose their data keys directly; errors use `{"error":{"message","details"}}`. Dates are exchanged as `YYYY-MM-DD` strings. Keep frontend parsing aligned with `pkg/utils/response.go` and `web/src/api/http.js`.
- With an empty `SQL_DSN`, the app uses auto-created SQLite at `SQLITE_PATH` (default `./data/invoiceman.db`); a `postgres://` DSN selects PostgreSQL, and any other `SQL_DSN` value is a startup error (fail fast — MySQL is not supported). Schema setup is GORM `AutoMigrate` at startup and in test setup; do not add or expect migration files or a migration CLI.
- `.env` is auto-loaded by `main.go`. Local development needs no database or Redis service: SQLite and in-memory sessions are the defaults. Docker builds require a root `.env` because the `Dockerfile` copies it into the scratch image. Start from `.env.example` when needed.

## Verification

- Focused backend check: `go test ./...`.
- Focused route flow check: `go test ./pkg/routes/ -run TestClientInvoiceFlow -v`.
- `make test` additionally runs `clean`, `gocritic`, `gosec`, `golangci-lint`, and verbose coverage; these tools may not be installed. `make build` depends on that full target.
- `make run` regenerates Swagger, runs the full build/test chain, then starts on port `5000`.
- The Docker flow is `make docker.run`; it creates `template-network`, `template-postgres`, `template-fiber`, and `template-redis`. Stop it with `make docker.stop`.

## Frontend

- From `web/`, use `npm run dev`, `npm run build`, or `npm run lint`. There is no frontend test script.
- Vite serves on `5173` and proxies `/api` to `http://localhost:5000`; run the Go API separately for API-backed development.
- `@` aliases to `web/src`. TanStack Query is configured for one retry, no refetch on window focus, and a 30-second stale time. Axios normalizes API failures in `web/src/api/http.js`.
- Backend routes are currently implemented for auth, clients, invoices, dashboard, items, expenses, payments, reports, settings, AI, admin users, public payments, and the central multi-gateway payment relay (`/api/gateway/*` service intents + `POST /api/webhooks/:gateway` + `/api/admin/gateway/*`; providers implement `platform/gateway.Gateway`, Midtrans is the first). The SPA also calls all of these; verify any new frontend call has backend support before assuming it works.

## Structure Conventions

- Backend layering: `app/` may import `pkg/utils` and `pkg/repository`, but never `pkg/middleware` or `platform/*` in new code. Read session identity via `utils.CurrentUserID` and service identity via `utils.CurrentServiceProject`; middleware in `pkg/middleware` wires `platform/` and stores identity in context locals.
- Backend shared helpers: dates go through `pkg/utils/date.go` (`DateLayout = "2006-01-02"`, `ParseDate`/`ParseRequiredDate`/`FormatDate`/`FormatTime`) — do not define local layouts or parsers in controllers or queries. All controller responses use `utils.OK`/`utils.Fail`; DELETE/logout return `204` with no body.
- Backend naming: one domain per controller file, named for its current domain (not legacy provider names). Use `Get*` for single-resource getters (`GetSettings`, `GetDashboard`). Roles are `admin`/`user`/`moderator`, so `RequireRoles("admin", "user")` intentionally excludes moderators — keep route tests covering RBAC expectations.
- Frontend modules: `web/src/api/` and `web/src/hooks/` are one module per domain (`items`, `expenses`, `payments`, `reports`, ...). The shared axios instance lives in `web/src/api/http.js` and is the only file importing axios. `components/` is grouped by feature with primitives in `components/ui/`; `pages/` stays flat per route.
- Frontend robustness: shared format helpers in `web/src/lib/utils.js` must never throw on missing/invalid input — return `""` or `"—"` instead (e.g. `relativeTime`). The sidebar groups admin menus directly above Settings; the command palette lists frequent destinations only.

## Workflow

- Use 2-space indentation in frontend/config files and tabs in Go, per `.editorconfig`.
- `opencode.json` loads this file as the repository instruction source. Keep code, comments, and repository documentation in English.
