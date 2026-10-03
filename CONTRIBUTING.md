# Contributing to TuPay

Thanks for your interest in contributing to **TuPay**! This document explains how
to set up a local development environment, the conventions you must follow,
and the pull request process.

> By participating, you agree to follow our [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).
> For security issues, see [`SECURITY.md`](SECURITY.md) — please do not file
> public issues for vulnerabilities.

---

## Quick start

```bash
# Fork and clone
git clone https://github.com/<you>/tupay.git
cd tupay
cp .env.example .env

# Backend (Go) on :5000
make dev-be

# Frontend (Bun + Vite) on :5173
make dev-fe
```

Open Swagger UI at <http://127.0.0.1:5000/swagger/index.html> for the API.

Prerequisites: Go (matching `go.mod`), Bun, and (optionally) Docker. Local
development does **not** require any external services — SQLite is used by
default.

## Branching model

TuPay uses a three-branch flow, enforced by `make check-flow` and CI:

```
dev   ← daily development, default branch for PRs
 ↓    (fast-forward, via `make promote`)
main  ← stable releases
 ↓    (fast-forward, via `make promote-prod`, tagged `v$(VERSION)`)
master← production
```

Rules (full detail in [`ARCHITECTURE.md`](ARCHITECTURE.md) and
[`CODE_STYLE.md`](CODE_STYLE.md)):

- **Never** commit directly to `main` or `master`. Promote with
  `make promote` then `make promote-prod`.
- All PRs target `dev`.
- `VERSION` only changes in release PRs. Do not bump it in feature/fix commits.
- `make promote` is fast-forward only — no merge commits, no force pushes.

## Working on a change

1. Branch from `origin/dev`:
   ```bash
   git fetch origin --prune
   git checkout -b feat/<short-slug> origin/dev
   ```
2. Make your change. Follow the conventions below.
3. Run local checks (see "Required local checks" below).
4. Open a PR against `dev` using
   [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md).

### Commit message style

Conventional Commits, scope is the area you touched, English:

```
feat(invoices): add CSV export of invoice line items
fix(auth): correct CSRF token rotation on logout
docs(arch): document the outbox retry budget
chore(deps): bump go-fiber to v2.x
test(oidc): cover resolve probe timeout
```

## Repository conventions (read these)

These docs are the single source of truth for their topics — do not duplicate
their rules here:

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system design, data flow, security,
  deploy, branching, VERSION, payment gateway abstraction.
- [`CODE_STYLE.md`](CODE_STYLE.md) — naming, file structure, patterns, CI gates,
  commit/PR rules, file-size baseline.
- [`docs/MODULE_MAP.md`](docs/MODULE_MAP.md) — domain owners and request paths.
- [`docs/API_DOCS.md`](docs/API_DOCS.md) — API envelope, errors, money/date
  formats.
- [`docs/adr/`](docs/adr/) — Architecture Decision Records; code marked
  `INTEGRATION CONTRACT` points at one and must not be deleted as dead code.

### Backend (Go)

- Queries live in `app/queries` — **no inline queries in controllers**.
- Raw SQL is only allowed in `platform/database` (driver-specific code).
- Migrations live in `platform/database` and must keep SQLite **and**
  PostgreSQL working.
- API responses use the `utils.OK` / `utils.Fail` envelope; errors are stable
  English strings.
- Money values are **decimal strings**; dates are `YYYY-MM-DD`.

### Frontend (React + Vite)

- All axios calls must go through `webui/src/api/http.js`.
- `@react-pdf/renderer` and `recharts` are isolated behind their own modules
  to keep bundle size sane.
- i18n keys live in `webui/src/lib/i18n.{en,id}.ui.js` — add both languages.
- Run `bun run --cwd=webui check:size` after any Go/JS change; the baseline
  guard fails the build when files grow past their baseline entry.

### Tests

- Backend: `go test ./...` must run with **no external services**.
- Flow tests use fixtures — never hit real network or real databases.
- Frontend: `bun run test` (which is `node --test`). Do **not** run bare
  `bun test` — the suite uses `node:test` mock APIs that bun's runner lacks.

## Required local checks

Run these locally before pushing — CI runs them too:

```bash
# Branching invariants (target/main flow, VERSION sync, etc.)
make check-flow

# Backend tests (no external services required)
go test ./...

# Frontend tests (node:test under the hood)
bun run test

# File-size baseline guard (after any Go/JS change)
bun run --cwd=webui check:size

# VERSION sync (only matters when VERSION is touched)
node scripts/sync-version.mjs --check
```

If any of these fail, the CI pipeline will reject the PR.

## Pull request process

- Open your PR against `dev` using
  [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md).
- Keep PRs focused. Split large changes into smaller, reviewable PRs.
- Reference the issue the PR closes (`Closes #123`).
- Update relevant docs (`ARCHITECTURE.md`, `CODE_STYLE.md`, `MODULE_MAP.md`,
  `API_DOCS.md`, `webui/CHANGELOG.md`) when behavior or interfaces change.
- A maintainer will review. Expect CI to run lint, dialect checks, and the
  full test matrix.

## Reporting bugs and requesting features

- Use the issue templates in
  [`.github/ISSUE_TEMPLATE/`](.github/ISSUE_TEMPLATE/):
  - **Bug report** — `.github/ISSUE_TEMPLATE/bug_report.md`
  - **Feature request** — `.github/ISSUE_TEMPLATE/feature_request.md`
- For security issues, follow [`SECURITY.md`](SECURITY.md) — do not file a
  public issue.

## Out of scope

- Issues and PRs about third-party projects, frameworks, or packages —
  report them upstream.
- Anything that violates the [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).

## License

By contributing, you agree that your contributions are licensed under the
project's [AGPL-3.0 license](LICENSE).
