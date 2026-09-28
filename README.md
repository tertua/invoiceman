# Invoiceman

Invoice management API + SPA: invoices, clients, payments with shareable public pay links (QRIS via Midtrans, crypto via NOWPayments), a gateway relay for third-party integrations, expenses, dashboards, reports, and AI-assisted text (Gemini, English/Indonesian).

- Backend: Go (Fiber), GORM over SQLite/PostgreSQL, optional Redis.
- Frontend: React + Vite SPA in `webui/`.
- API docs: Swagger UI at http://127.0.0.1:5000/swagger/index.html (regenerate with `make swag` after changing annotations).

## Quick start

Docker (Postgres + Redis + API):

1. Copy `.env.example` to `.env` and adjust as needed.
2. Run: `make docker.run`
3. Open Swagger: http://127.0.0.1:5000/swagger/index.html

Local dev (no external services):

1. `make dev-be` — builds to `/tmp/opencode/invoiceman`, serves `:5000`. Rebuild + restart after BE changes.
2. `make dev-fe` — Vite dev on `:5173`, proxies `/api` and `/uploads` to the BE. Hot-reloads; no restart needed.
3. Health: `curl localhost:5000/healthz`

## Configuration

`.env` is loaded automatically. The dev defaults need zero external services:

- Empty `SQL_DSN` selects an auto-created SQLite file (`SQLITE_PATH`, under `data/`).
- Empty `REDIS_HOST` selects in-memory session/cache stores.
- Local state lives in gitignored `data/` and survives BE restarts — except sessions: with in-memory sessions every BE restart invalidates all logins (expect 401s until re-login).

Payments are configured via env (`MIDTRANS_*`, `NOWPAYMENTS_*`); the public pay page and gateway relay stay inert until keys are set. Optional `GEMINI_API_KEY` enables the AI text endpoints; optional `TURNSTILE_SECRET` enables CAPTCHA (see below).

## API conventions

- Versioned routes live under `/api/v1`, legacy routes under `/api`; `/api/v1` is registered first.
- Successes carry data fields directly (`utils.OK`); errors use the `error.message`/`details` shape (`utils.Fail`).
- Session-cookie mutations require `X-CSRF-Token`; money-moving mutations require an `Idempotency-Key`. Gateway routes authenticate by API key.
- Money crosses the API as decimal strings; dates use `YYYY-MM-DD`.

## Docker

The image contains only the binary — never bake secrets in. Inject at run time:

`docker run --env-file .env apiserver`

There are two deploy modes:

- **Split (default, `Dockerfile`)**: API-only. Build the FE (`make webui.check`) then host it separately (nginx/Cloudflare). Because the FE calls `/api/v1` relatively, the FE host must proxy `/api` and `/uploads` to the BE — for a different origin, set `CORS_ORIGINS` and use HTTPS (the `Secure` session cookie does not work over plain HTTP/IP). One BE can serve many FEs.
- **Single container (`Dockerfile.dev`)**: FE embedded into the image, `/api/*` same-origin — no CORS, proxy, or second domain.

  ```bash
  docker build -f Dockerfile.dev -t invoiceman:dev .
  docker run --rm -p 5000:5000 --env-file .env invoiceman:dev
  ```

Compose can use `image:` + `env_file: .env`; no env is needed at build time.

## Turnstile (CAPTCHA)

Optional, active only when the secret is set. There are **two different** keys:

| Key | For | Goes in |
|---|---|---|
| Secret key | Server-side token verification | Runtime env `TURNSTILE_SECRET` (`.env`) |
| Site key | Rendering the widget in the browser | Build-time `VITE_TURNSTILE_SITE_KEY` |

`VITE_*` is baked into the bundle by Vite at `npm run build`, so the site key is not a runtime env. For the embed image, pass it as a build arg:

```bash
docker build -f Dockerfile.dev \
  --build-arg VITE_TURNSTILE_SITE_KEY=<site-key> -t invoiceman:dev .
```

Both keys must come from the same Cloudflare pair. When unused, leave them empty: the widget is not rendered and the server skips verification.

## Commands

- `make dev-be` / `make dev-fe` — local API / Vite dev (see Quick start).
- `make run` — `swag init`, build, then start the API.
- `make test` — clean, gocritic, gosec, golangci-lint, then coverage tests. `make build` depends on it, so it is not a quick binary.
- `go test ./...` — default suite, no external services; focus a route flow with `go test ./pkg/routes -run TestName -v`.
- `make webui.check` — FE lint, tests, chart/bundle checks, and build.
- `make docker.run` / `make docker.stop` — Docker Postgres + Redis + API.

`VERSION` is canonical (see `VERSION`); changes are noted in `webui/CHANGELOG.md`.

## Frontend bundle boundaries

The SPA lazy-loads routes and keeps heavy dependencies out of the initial bundle:

- `@react-pdf/renderer` may only be imported by `webui/src/components/invoice/InvoiceDocument.jsx` and `InvoicePdfDownloadContent.jsx`.
- `recharts` is reserved for the full chart pages: Dashboard, ClientDetail, and Reports. Small dashboard sparklines use SVG.
- Run `npm --prefix webui run check:bundles` for an advisory report or `make webui.check` for strict lint, build, and bundle-budget checks.
