# Repository guide

Read in order: this file → `ARCHITECTURE.md` (system design, data flow, security, deploy) → `CODE_STYLE.md` (naming, file structure, patterns, CI gates) → `docs/MODULE_MAP.md` (domain owners + request path) → `main.go` → `pkg/routes/versioning.go`. Don't re-derive ownership by grepping. `ARCHITECTURE.md` and `CODE_STYLE.md` are the single source of truth for their topics — do not duplicate their rules here.

## Chat output (assistant)

- **Chat replies: always Bahasa Indonesia.** Every final answer you give the user in chat must be written in Bahasa Indonesia — no exceptions, even when the user writes in English or Indonesian.
- That language rule covers **chat output only**. Code, identifiers, commit messages, PR text, comments, docs, and agent prompts (`delegated task`, `spawn_agent`) stay in English as they are today — the chat language never switches them, and their language never switches the chat.
- Keep answers short and direct (CLI-friendly); no unnecessary preamble or postamble.
- When a request is ambiguous (scope, phrasing, defaults, or direction), resolve it with the `ask user` instead of assuming — offer concrete options and a recommended first choice.
- Ask-user lifecycle: close the octto session (`end_session`) **immediately after the answer is received**, before doing any further thinking/work — never leave the browser window open while working.

## Naming

- **Never invent a name** — route URL, API field, file, identifier, env var, branch. Derive it from what the system already calls the thing (`docs/MODULE_MAP.md`, existing routes/models/queries, `CODE_STYLE.md`). Enterprise-grade means *consistent with the codebase*, not a fresh invention: for anything client-visible (URLs, JSON fields, public IDs) stop and `ask user` with concrete options + a recommended first choice instead of picking one yourself.
- **Singular, not plural: `X`, never `Xs`.** No `-s` pluralization in any new name (route segment, file, identifier, API field) — `database` not `databases`, `client` not `clients`, `invite` not `invites`. The plural names that already exist (`/invoices`, `/payments`, `/settings`, …) are legacy and frozen: never rename them, or the FE and any existing client break.
- **Debug first, names second.** While debugging, do not rename/restructure code and do not introduce new abstraction layers (`pipeline`, `manager`, `factory`, `service`, `helper`) — find the root cause and show evidence (diff, failing/passing test) first. Any rename or new layer is a separate, explicit proposal the user approves, never a side effect of a fix.

## Integration door command ("buka pintu integrasi")

When the pilot says **"buka pintu integrasi ke module X ke aplikasi Y"** (open an integration door from module X to application Y), execute directly — no options menu, no confirmation round, no architecture survey; the pilot holds the direction. Deliver exactly these three, then stop:

1. **DB schema** — new GORM model in `app/models/*_model.go` (singular, ≤200-line budget), registered in the `AutoMigrate` list in `platform/database/open_db_connection.go` with `SchemaVersion` bumped by 1; SQLite **and** PostgreSQL both work; reference the file in `docs/MODULE_MAP.md`.
2. **ADR** — next free `docs/adr/NNNN-*.md` (kebab title from module + application name), status `Accepted`, context = which module, which application, why this door; add it to the index in `docs/adr/README.md`.
3. **Door stub** — the not-yet-called integration point (interface, hook, or entry function) that will eventually reach Y, marked with **that** ADR, not 0001: `// INTEGRATION CONTRACT — do not delete. See docs/adr/NNNN-<title>.md`

Then run `go test ./...` + `bun run --cwd=webui check:size`, report the three paths, and wait for the next command. Wiring, routes, controllers, and callers are **not** part of this command — they arrive with the pilot's next instruction. Marker rules: `docs/adr/0001-integration-contract.md`.

## Invariants (summary only — full detail in the docs above)

- Branching: never commit to `main`/`master`; promote with `make promote` (dev→main) then `make promote-prod` (main→master); `VERSION` changes only in releases. → `ARCHITECTURE.md`
- Files: never grow a file past its `scripts/file-size-baseline.json` entry; run `bun run --cwd=webui check:size` after any Go/JS change; lower a baseline entry after a split, never raise it. → `CODE_STYLE.md`
- Dead code: code marked `INTEGRATION CONTRACT` is a deliberate not-yet-used integration point with a written ADR — never delete it as unused; retire it only via its ADR first. → `docs/adr/0001-integration-contract.md`
- Data: queries in `app/queries` (no inline in controllers), raw SQL only in `platform/database`; SQLite + PostgreSQL must both work. → `ARCHITECTURE.md`
- API: `utils.OK` / `utils.Fail` envelope, stable English errors; money as decimal strings; dates `YYYY-MM-DD`. → `ARCHITECTURE.md`
- Frontend: axios only in `api/http.js`; `@react-pdf/renderer` / `recharts` / i18n boundaries are CI-enforced. → `CODE_STYLE.md`
- Tests: `go test ./...` needs no external services; flow tests must use fixtures. → `CODE_STYLE.md`
- Frontend tests: run `bun run test` (= `node --test`), never bare `bun test` — the suite uses `node:test` mock APIs (`stub.reset()`) that bun's runner lacks. → `CODE_STYLE.md`
