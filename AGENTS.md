# AGENTS.md

Go (Fiber v3) API + Vite React SPA in `web/`. Two separate apps, no shared workspace. Backend module `github.com/tertua/invoiceman`, entry `main.go`. Frontend entry `web/src/main.jsx` → `App.jsx` → `routes.jsx`.

## Backend (Go, repo root)

- Base API path is `/api/v1` (`pkg/routes/*_routes.go`); Swagger at `/swagger/index.html`. Regenerate after annotation/controller changes: `swag init` (requires `swag` CLI; also runs as part of `make run`, `make docker.run`). `docs/` is generated but committed.
- Layering (enforced by convention, see `app/BUSINESS_LOGIC.md`, `pkg/PROJECT_SPECIFIC.md`, `platform/PLATFORM_LEVEL.md`):
  - `app/` — business logic only: `controllers/`, `models/`, `queries/`. No infra imports.
  - `pkg/` — project wiring: `configs/`, `middleware/`, `routes/`, `repository/` (consts), `utils/`.
  - `platform/` — infra: `database/` (pgx/mysql via `OpenDBConnection`, `DB_TYPE=pgx|mysql`), `cache/redis.go`, `migrations/`.
- Env: `main.go` uses `godotenv/autoload`, so `.env` (gitignored) is required locally — copy from `.env.example`. Tests (`pkg/routes/*_test.go`) load `../../.env.test` explicitly via `godotenv.Load`, not `.env`. Note hostnames differ: `.env.example` uses `template-postgres`/`template-redis` (docker network), `.env.test` uses `host.docker.internal`.
- Migrations use `golang-migrate/migrate` CLI on `platform/migrations`: `make migrate.up|down` (`make migrate.force version=N`). `Makefile` hardcodes `DATABASE_URL=postgres://postgres:password@template-postgres/postgres?sslmode=disable` — it ignores `.env`, so migrations only work against that docker-network host.

## Makefile gotchas

- `make test` = `clean` + `gocritic` + `gosec` + `golangci-lint` + `go test ./...`, and `make build` depends on `test`. Those three linters plus `migrate`/`swag` CLIs may not be installed — for a focused check run `go test ./...` or `go test ./pkg/routes/ -run TestPrivateRoutes -v` directly instead of `make test`.
- `make run` = `swag` + `build` (build itself re-runs full `test` chain) then runs `./build/apiserver`. Expects Postgres/Redis reachable per `.env`.
- `make docker.run` chains `network → postgres → swag → fiber → redis → migrate.up` with container names `template-postgres`/`template-fiber`/`template-redis`. `Dockerfile` (scratch image) `COPY`s `.env` into the image, so `.env` must exist before `docker build`.
- Server port defaults to `5000` (`SERVER_PORT`); Swagger URL `http://127.0.0.1:5000/swagger/index.html`.

## Frontend (`web/`)

- Commands (run in `web/`): `npm run dev` (vite `:5173`), `npm run build`, `npm run lint` (flat `eslint.config.js`, ignores `dist`). No tests.
- Vite dev proxy sends `/api` → `http://localhost:8000`, but the Go server defaults to port `5000` and serves `/api/v1` book/auth/token routes, while `web/src/api/*.js` calls `/auth/*`, `/invoices`, etc. via `baseURL: "/api"`. Frontend and backend are currently **not wired together** — do not assume an endpoint exists on both sides; verify in `pkg/routes/` vs `web/src/api/`.
- Conventions: `@` → `src/` (set in both `vite.config.js` and `jsconfig.json`); routing via `createBrowserRouter` in `src/routes.jsx` with `ProtectedShell` (AuthContext) guarding `/dashboard`, `/invoices/*`, `/clients/*`, `/expenses`, `/payments`, `/items`, `/reports`, `/settings`; public `/pay/:token`. Data fetching via TanStack Query (`retry: 1`, no window refocus, 30s stale); axios `apiClient` normalizes errors to `{status, message, details}`.

## Style / ops

- `.editorconfig`: 2-space indent everywhere except Go (tabs). No CI workflows or pre-commit hooks in repo. `opencode.json` loads this file as instructions.
- Language: all code, comments, docs, commit messages in English. Indonesian only for chat with the user, never in the repo.
