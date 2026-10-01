<!--
  TuPay pull request template.
  Branch target: `dev` (CI and `make check-flow` will reject direct PRs to main/master).
-->

## Summary

<!-- One or two sentences describing what this PR changes and why. -->

## Related issue

<!-- Link the issue this PR closes, e.g. "Closes #123". Use "n/a" if there is none. -->

## Type of change

<!-- Tick all that apply. -->

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation / chores (no production code change)
- [ ] Security fix (see [`SECURITY.md`](SECURITY.md) for reporting)

## What changed

<!-- A short list of the most important changes. -->

- ...

## How it was tested

<!-- Describe the local verification you ran. CI must pass before review. -->

- [ ] `go test ./...` passes locally
- [ ] `bun run test` (= `node --test`) passes locally (when frontend is touched)
- [ ] `bun run --cwd=webui check:size` passes (when Go/JS files are touched)
- [ ] `make check-flow` passes (branching invariants)
- [ ] New/updated tests cover the change (fixtures for flow tests)
- [ ] Manually exercised in the UI / via Swagger (`/swagger/index.html`)

### Reproduction / screenshots

<!-- If applicable, add screenshots, curl traces, or reproduction steps. -->

## Checklist

<!-- Contributor Covenant 2.1 + repo invariants. -->

- [ ] My branch is forked from / rebased onto `origin/dev`
- [ ] PR target is `dev` (not `main` or `master`)
- [ ] I did not touch `VERSION` (only release PRs change it)
- [ ] I followed the conventions in [`CODE_STYLE.md`](CODE_STYLE.md) and [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [ ] For backend changes: queries live in `app/queries`, raw SQL only in `platform/database`, and SQLite + PostgreSQL both work
- [ ] For frontend changes: axios calls only in `webui/src/api/http.js`, and `@react-pdf/renderer` / `recharts` / i18n boundaries respected
- [ ] I have read and agree to the [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)
- [ ] If this PR relates to a security issue, see [`SECURITY.md`](SECURITY.md) — do not discuss the details here

## Notes for reviewers

<!-- Anything reviewers should know: rollout, follow-ups, risk areas, dependencies. -->
