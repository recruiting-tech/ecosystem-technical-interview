# ECO-1024 — Add `skill_level` attribute to camp records

| | |
|---|---|
| **Type** | Story |
| **Priority** | Medium |
| **Sprint** | 24.05 |
| **Status** | In Progress |
| **Reporter** | Sasha L. (Product) |
| **Assignee** | _you_ |
| **Components** | `apps/api-svc` |
| **Blocks** | ECO-1025 |

## Description

> As a camp admin, I want to capture the skill level a camp targets so
> athletes (and their parents) can later filter searches to camps that
> fit them.

We're starting with three values: `beginner`, `intermediate`, `advanced`.
Don't over-constrain — Sasha thinks we'll add `elite` and maybe a
`pre-K` track later in the year, so the storage shape should be friendly
to that.

## Acceptance Criteria

- [ ] `POST /camps` accepts an optional `skill_level` field in the body
- [ ] `GET /camps` and `GET /camps/:id` include `skill_level` in the response
- [ ] Existing camp rows continue to load and serialize without errors
- [ ] One round-trip integration test asserts create + fetch with the new field

## Out of scope

- Emitting the field on the `Camp.V1` event — see **ECO-1025**.
- Indexing it for search — see **ECO-1026**.
- Any frontend wiring.

## Technical notes

- Column should be **nullable** so the migration doesn't require a backfill on existing rows.
- `make gen` (root or `apps/api-svc`) regenerates the sqlc repository and the OpenAPI Gin server. Don't hand-edit `*.gen.go` / `*.sql.go` files.
- The existing test in `internal/api/camps_test.go` is the pattern to clone.

## How to verify

- **Tests:** `cd apps/api-svc && make test` — your new round-trip case (with `skill_level`) and the existing tests should both pass.
- **CLI / curl:** with `make api-svc` running, `curl -X POST -H 'Content-Type: application/json' -d '{...,"skill_level":"intermediate"}' localhost:8001/camps | jq` should echo the field; `curl localhost:8001/camps/<id>` round-trips it.
- **DB:** `docker compose exec postgres psql -U emc -d api_svc -c '\d camps'` confirms the column is present and nullable.
- **Web:** _not yet visible_ — the indexer doesn't write `skill_level` to the OpenSearch document until ECO-1026, and the row card only renders fields that flow through that pipeline. Don't be alarmed if `make web` shows your new camp without a skill chip; that lands in Ch3.

## Comments

> **Sasha L.** — _2 days ago_
> Optional in v1. We'll talk about whether it becomes required for new
> camps after we see adoption from the front-end work in 24.06.
