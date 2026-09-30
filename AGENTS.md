# Repository guide

Read in order: this file → `ARCHITECTURE.md` (system design, data flow, security, deploy) → `CODE_STYLE.md` (naming, file structure, patterns, CI gates) → `docs/MODULE_MAP.md` (domain owners + request path) → `main.go` → `pkg/routes/versioning.go`. Don't re-derive ownership by grepping. `ARCHITECTURE.md` and `CODE_STYLE.md` are the single source of truth for their topics — do not duplicate their rules here.

## Chat output (assistant)

- Always respond user chat in Bahasa Indonesia, regardless of the language used in code, commits, spawn_agent, comments or docs.
- Keep answers short and direct (CLI-friendly); no unnecessary preamble or postamble.
- When a request is ambiguous (scope, phrasing, defaults, or direction), resolve it with the `question` tool instead of assuming — offer concrete options and a recommended first choice.

## Invariants (summary only — full detail in the docs above)

- Branching: never commit to `main`/`master`; promote with `make promote` (dev→main) then `make promote-prod` (main→master); `VERSION` changes only in releases. → `ARCHITECTURE.md`
- Files: never grow a file past its `scripts/file-size-baseline.json` entry; run `npm --prefix webui run check:size` after any Go/JS change; lower a baseline entry after a split, never raise it. → `CODE_STYLE.md`
- Data: queries in `app/queries` (no inline in controllers), raw SQL only in `platform/database`; SQLite + PostgreSQL must both work. → `ARCHITECTURE.md`
- API: `utils.OK` / `utils.Fail` envelope, stable English errors; money as decimal strings; dates `YYYY-MM-DD`. → `ARCHITECTURE.md`
- Frontend: axios only in `api/http.js`; `@react-pdf/renderer` / `recharts` / i18n boundaries are CI-enforced. → `CODE_STYLE.md`
- Tests: `go test ./...` needs no external services; flow tests must use fixtures. → `CODE_STYLE.md`
