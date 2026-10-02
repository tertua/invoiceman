# ADR 0001: `INTEGRATION CONTRACT` marker for not-yet-used integration point

- Status: Accepted
- Date: 2026-10-03

## Context

Integration point created before its caller exists — a future payment provider
capability, a planned route, a webhook verifier wired for an upcoming provider,
a config key read by code that ships in the next milestone. Such a symbol has
zero references at the moment it lands.

Reviewers and maintenance agents (human or automated) treat zero-reference as
dead code and delete it. The deletion is locally correct and globally wrong:
it removes a deliberate door plus the decision behind it, and the door has to
be rebuilt from scratch later — usually under deadline, usually worse.

The repo needs a marker that is visible in the code, unambiguous to a reader
skimming a diff, and backed by a written record that says *why the door exists*
and *when it may close*.

## Decision

Every integration point that has no caller yet carries a single-line marker on
its own comment line, immediately above the declaration:

```go
// INTEGRATION CONTRACT — do not delete. See docs/adr/0001-integration-contract.md
func VerifyWebhookSignature(payload []byte, sig string) error { ... }
```

Marker syntax by language: `//` in Go and JavaScript, `--` in SQL, `#` in
shell/YAML, `<!-- -->` in markup. Rules:

1. **One line, directly above the declaration it protects.** Never scatter it
   in a header block away from the symbol.
2. **The path must point to an existing `docs/adr/*.md`** that explains why
   this door exists. No ADR → no marker → the code is fair game.
3. **A marked symbol is not dead code.** Never delete it, "clean it up", or
   collapse it just because it has no call site. No caller is the designed
   state, not a defect.
4. **Retiring goes through the ADR first.** Update the ADR (status →
   `Deprecated` or `Superseded`, plus why the door closed), then remove marker
   and code in the same change. Marker never outlives its rationale, and code
   is never removed before the rationale is recorded.
5. **The exemption is narrow.** The marker buys exemption from dead-code
   removal only. Marked code still answers to lint, tests, the file-size
   baseline, and security review like any other code.

## Consequences

- + A door that is unused today survives review tomorrow; the rationale is one
  comment line away.
- + Grep for `INTEGRATION CONTRACT` lists every deliberate open door in the
  system — useful when auditing what is half-wired.
- + Retirement is an explicit, recorded act instead of an accidental deletion.
- − Markers can go stale if a door is closed without touching its ADR; rule 4
  exists to make that a reviewable omission.
- − The marker can be abused to shield genuinely dead code; a reviewer who is
  unconvinced asks for the ADR link (rule 2) and, if the rationale no longer
  holds, deprecates the ADR before removing anything.
