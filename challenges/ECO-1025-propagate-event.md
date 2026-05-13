# ECO-1025 — Propagate `skill_level` through `Camp.V1` event stream

| | |
|---|---|
| **Type** | Story |
| **Priority** | Medium |
| **Sprint** | 24.05 |
| **Status** | Open |
| **Reporter** | Marie F. (Platform) |
| **Assignee** | _you_ |
| **Components** | `schemas`, `apps/api-svc`, `apps/indexer` (read-only here) |
| **Depends on** | ECO-1024 |
| **Blocks** | ECO-1026 |

## Description

ECO-1024 stops at the source-of-truth row. Now the field needs to travel
on the `Camp.V1` event stream so the indexer (and a few internal
consumers we haven't enumerated yet — the data team is sniffing around
this topic) can react to it.

## Acceptance Criteria

- [ ] `schemas/Camp.V1.avsc` carries `skill_level`
- [ ] `make check-compat` reports the change as compatible against the registered version
- [ ] `apps/api-svc` publisher emits the new field on create/update
- [ ] PR description records: which compatibility level you chose, and **why**
- [ ] Producer-first rollout — see "Notes from the platform call" below

## Notes from the platform call yesterday

> The indexer team (which owns `apps/indexer` in real life — separate
> repo, separate deploy) is in code freeze for the mobile release cut
> through **2026-05-19**. We can't ship them code right now.
>
> Design the schema bump so they can opt in later, on their own
> schedule, without us holding their hand. Producer ships now; consumer
> waits.

## Discussion welcome

- `BACKWARD` vs `FULL` compatibility for this subject

## How to verify

- **`make check-compat`** is the primary safety net — it should print `✓ Camp.V1-value compatible` against the registered version. If you make an incompatible change it'll fail with Schema Registry's verbose explanation.
- **Schema Registry directly:** `curl http://localhost:18081/subjects/Camp.V1-value/versions/latest | jq '.schema | fromjson | .fields'` — confirm `skill_level` is in the latest version.
- **End-to-end with a still-old indexer:** with `make api-svc` and `make indexer` running, POST a camp with `skill_level` and watch `/tmp/idx.log` (or wherever you redirected stderr). The indexer is still on its old `campEventV1` struct — observe what happens.
- **Web:** still no UI signal for `skill_level` (waiting on ECO-1026). The web should keep working with sport/text filters; if it stops, you've broken something else.

## Comments

> **Marie F.** — _yesterday_
> If you go optional+nullable, please write a note in the PR about what
> happens to old messages already on the topic. We don't reset
> compacted topics in dev or prod.
