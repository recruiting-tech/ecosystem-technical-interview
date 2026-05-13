# ECO-1026 — Index `skill_level` in OpenSearch + backfill existing camps

| | |
|---|---|
| **Type** | Story |
| **Priority** | Medium |
| **Sprint** | 24.06 |
| **Status** | Open |
| **Reporter** | Jenna R. (Search) |
| **Assignee** | _you_ |
| **Components** | `apps/indexer`, `apps/web` (stretch) |
| **Depends on** | ECO-1025 |

## Description

Now that `Camp.V1` is carrying `skill_level` (per ECO-1025), we need the
indexer to (a) write the field into the OpenSearch document and (b) fill
it in for camps that were created _before_ the producer change shipped —
otherwise our search experience is split-brain for ~6 weeks of camp
data.

## Acceptance Criteria

- [ ] OpenSearch `camps` mapping includes `skill_level` (keyword type — we filter, don't free-text)
- [ ] `internal/domain/camp_doc.go` populates the new field
- [ ] New `indexer backfill` Cobra subcommand re-emits known camps as `Camp.V1` events
- [ ] Backfill is idempotent — re-running it does not duplicate or flap docs

## Stretch (only if time)

`apps/web/src/App.tsx` has a `skill_level` filter form control already
shaped — currently commented out with `TODO(ch3-stretch)`. If you have
time after the backfill is solid, wire it through to the search API.

## Operational notes

- Dev compose runs OpenSearch on `localhost:19200`; `docker compose logs
  opensearch` is your friend if writes start 4xx-ing.
- The backfill needs camp data from somewhere — think about which
  service boundary it's polite to cross.

## How to verify

- **Tests:** `cd apps/indexer && make test` — extend `internal/handlers/camp_test.go` to assert `skill_level` lands on the doc after decoding a Camp.V1 message that carries it.
- **Web (this is the payoff):** after `make api-svc && make indexer && make web`, POST a camp with `skill_level`, refresh the web — the row card should show a small `skill_level` chip next to the sport. Backfill verification: stop the indexer, post several camps, restart the indexer, run `indexer backfill` — every camp shows up in the search results within a few seconds.
- **OpenSearch directly:**
  - `curl 'http://localhost:19200/camps/_count?q=_exists_:skill_level'` → number of indexed docs that have the field.
  - `curl 'http://localhost:19200/camps/_mapping' | jq '.camps.mappings.properties.skill_level'` → confirm `keyword` type.
- **Backfill idempotency:** run `indexer backfill` twice. The doc count from `_count` should not change, and search results should be stable. (Give the indexer ~5s after each run for messages to be fully consumed before checking.)
- **Stretch:** the `/search` endpoint already accepts `?skill_level=X` (we scaffolded the backend). The Ch3 stretch is purely web-side: wire the form control to send the param and re-render. Verify by selecting a level and watching the result list narrow.
