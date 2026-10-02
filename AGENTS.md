# Repository guide

Read in order: this file → `ARCHITECTURE.md` (system design, data flow, security, deploy) → `CODE_STYLE.md` (naming, file structure, patterns, CI gates) → `docs/MODULE_MAP.md` (domain owners + request path) → `main.go` → `pkg/routes/versioning.go`. Don't re-derive ownership by grepping. `ARCHITECTURE.md` and `CODE_STYLE.md` are the single source of truth for their topics — do not duplicate their rules here.

## Chat output (assistant)

- **Chat replies: always Bahasa Indonesia.** Every final answer you give the user in chat must be written in Bahasa Indonesia — no exceptions, even when the user writes in English or Indonesian.
- That language rule covers **chat output only**. Code, identifiers, commit messages, PR text, comments, docs, and agent prompts (`delegated task`, `spawn_agent`) stay in English as they are today — the chat language never switches them, and their language never switches the chat.
- Keep answers short and direct (CLI-friendly); no unnecessary preamble or postamble.
- When a request is ambiguous (scope, phrasing, defaults, or direction), resolve it with the `ask user` instead of assuming — offer concrete options and a recommended first choice.
- Ask-user lifecycle: close the octto session (`end_session`) **immediately after the answer is received**, before doing any further thinking/work — never leave the browser window open while working.

## Invariants (summary only — full detail in the docs above)

- Branching: never commit to `main`/`master`; promote with `make promote` (dev→main) then `make promote-prod` (main→master); `VERSION` changes only in releases. → `ARCHITECTURE.md`
- Files: never grow a file past its `scripts/file-size-baseline.json` entry; run `bun run --cwd=webui check:size` after any Go/JS change; lower a baseline entry after a split, never raise it. → `CODE_STYLE.md`
- Data: queries in `app/queries` (no inline in controllers), raw SQL only in `platform/database`; SQLite + PostgreSQL must both work. → `ARCHITECTURE.md`
- API: `utils.OK` / `utils.Fail` envelope, stable English errors; money as decimal strings; dates `YYYY-MM-DD`. → `ARCHITECTURE.md`
- Frontend: axios only in `api/http.js`; `@react-pdf/renderer` / `recharts` / i18n boundaries are CI-enforced. → `CODE_STYLE.md`
- Tests: `go test ./...` needs no external services; flow tests must use fixtures. → `CODE_STYLE.md`
- Frontend tests: run `bun run test` (= `node --test`), never bare `bun test` — the suite uses `node:test` mock APIs (`stub.reset()`) that bun's runner lacks. → `CODE_STYLE.md`
