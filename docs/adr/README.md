# Architecture Decision Record (ADR)

An ADR records one decision that is expensive to reverse: the context that
forced it, the decision itself, and its consequences. One decision, one file.
The ADR is the written authority a reader (human or agent) consults before
touching the code that implements it.

## File naming

`docs/adr/NNNN-kebab-title.md` — four-digit sequence, zero-padded, never
renumbered and never reused:

```
docs/adr/0001-integration-contract.md
```

## Status

Every ADR carries a status line near the top:

- `Proposed` — drafted, not accepted yet.
- `Accepted` — in force.
- `Deprecated` — kept for history; say in which ADR it was superseded.
- `Superseded by ADR NNNN` — replaced; link the replacement.

## When to write one

- The choice is expensive to reverse (data format, public contract, provider
  abstraction, marker convention).
- Code needs written permission to exist in a state that looks wrong — most
  importantly an integration point with no caller yet (see
  [ADR 0001](0001-integration-contract.md)).

## Template

```markdown
# ADR NNNN: Title

- Status: Proposed | Accepted | Deprecated | Superseded by ADR NNNN
- Date: YYYY-MM-DD

## Context

What forced the decision. Facts and constraints, not opinions.

## Decision

What was decided, stated as rules someone can follow and check.

## Consequences

- Positive: what this buys.
- Negative: what it costs, what it rules out, what can go stale.
```

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [0001](0001-integration-contract.md) | `INTEGRATION CONTRACT` marker for not-yet-used integration point | Accepted |
