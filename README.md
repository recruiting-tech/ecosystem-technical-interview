# ecosystem-technical-interview

A live-pairing technical interview environment for the **ecosystem team** —
four loosely-coupled services in one repo, wired together over a real
event bus.

## How to read this repo

> **These four apps live as separate repos in real life — owned by different teams, deployed independently. They're packaged together here only so you can clone once. When you're reasoning about cross-app changes, treat them as separate.**

- `apps/api-svc/` — Go service (Gin, sqlc, Goose migrations, oapi-codegen, Watermill+Avro). Owns `Camp` reference data; publishes `Camp.V1` events.
- `apps/athlete-svc/` — Python service (Django, confluent-kafka, transactional outbox). Owns `Athlete` and `Registration`; publishes `AthleteCreated.V1` and `AthleteRegistered.V1`.
- `apps/indexer/` — Go service (Cobra, Watermill consumer, OpenSearch, Goose). Consumes events and indexes a denormalized `camps` document.
- `apps/web/` — TypeScript app (React Router v7, Tailwind, openapi-client). Search UI for camps.

Shared:

- `schemas/` — Avro schemas and `schemas.yaml` mapping subjects → files → compatibility policy.
- `compose.yml` — Postgres, Kafka (KRaft single-node), Confluent Schema Registry, OpenSearch.
- `Makefile` — top-level orchestration. Each app has its own `Makefile` for app-specific work.
- `Procfile` — process definitions consumed by `make dev`.
- `scripts/dev.sh` — pure-bash multi-process runner used by `make dev` when no Procfile runner is installed.

## Quick start

```bash
make setup             # install toolchains (mise) + per-app deps (one-time)
make up                # start infra (compose), wait for health
make register-schemas  # POST Avro schemas to Schema Registry
make test              # run every app's test suite
```

Docker is required for `make up` and for the athlete-svc test suite
(see [Tests](#tests)).

## Setup

`make setup` reads `.tool-versions` and installs Go, Python, and Node
via [`mise`](https://mise.jdx.dev) (recommended). `mise.toml` forces
prebuilt Python binaries (via `python-build-standalone`), so you don't
need `libssl-dev` / `libffi-dev` / etc. on your machine. If you don't
have `mise`, install it first or bring your own Go 1.25 / Python ≥3.12 /
Node 24 on your `PATH` and `make setup` will use those.

`make setup` is split into four stages — if one fails, you can rerun
just that stage rather than the whole thing:

| Target | What it does |
|---|---|
| `make setup-tools` | `mise install` (toolchains from `.tool-versions`) |
| `make setup-go` | `go mod download` for `api-svc` and `indexer` |
| `make setup-py` | venv + `pip install -e ".[dev]"` for `athlete-svc` |
| `make setup-web` | `npm install` for `web` |

## Running the full stack

For day-to-day work you'll usually only need one or two apps running.
For end-to-end verification (e.g. seeing a new camp in the web UI) you
need all four, plus infra. Each app's `make` target blocks the
foreground, so pick one of these:

**Four terminals (faithful to how the team works — separate repos, separate windows):**

```bash
# Terminal 1 — infra (one-time per session)
make up && make register-schemas

# Terminal 2 — api-svc on :8001
make api-svc

# Terminal 3 — indexer on :8003 (consumer + /search HTTP)
make indexer

# Terminal 4 — web on :8000 (Vite dev, proxies /api → indexer)
make web

# (athlete-svc on :8002 — only when you're working on athlete flows)
```

**One terminal via `make dev`** — runs all four apps with interleaved
logs. Works out of the box via `scripts/dev.sh` (pure bash, ^C stops
everything). For nicer signal handling and per-process restart, install
a Procfile runner:

```bash
brew install overmind   # uses Procfile, needs tmux
# or
pipx install honcho     # uses Procfile, pure Python
```

Then:

```bash
make up && make register-schemas   # infra still goes up first
make dev                            # ^C stops everything
```

`make dev` auto-detects in this order: overmind → honcho → bash fallback.

## Common per-app commands

The root `Makefile` exposes per-app passthroughs so you don't have to
`cd` into each app directory mid-task:

| Target | Effect |
|---|---|
| `make gen` | Run codegen across every app (sqlc + oapi-codegen for `api-svc`; no-op for others) |
| `make gen/api-svc` | Just `api-svc`'s codegen |
| `make gen/<app>` | Same shape for `athlete-svc`, `indexer`, `web` |
| `make test` | Run every app's test suite |
| `make test/api-svc` | Just `api-svc`'s tests |
| `make test/<app>` | Same shape for `athlete-svc`, `indexer`, `web` |
| `make api-svc` / `make athlete-svc` / `make indexer` / `make web` | Run that app in the foreground |

Each app's own `Makefile` (e.g. `apps/api-svc/Makefile`) has more
fine-grained targets — migrations, single-component runs, etc.

## Seeding sample data

```bash
make seed   # POST a handful of camps via api-svc → Camp.V1 events
```

Requires `make api-svc` to be running (or `make dev`). If api-svc isn't
reachable on `:8001`, the script tells you so and exits clearly. If
`make indexer` is also running, you'll see the camps appear in the web
search UI within a few seconds.

## Tests

Each app's tests are isolated from the dev compose stack — running
`make test` does not pollute Postgres rows, Kafka topics, OpenSearch
documents, or Schema Registry subjects in your dev environment.

| App | Isolation strategy |
|---|---|
| `api-svc` | Per-test ephemeral Postgres via testcontainers |
| `athlete-svc` | Postgres: pytest-django uses a separate `test_athlete_svc` DB and rolls back per test. Kafka + Schema Registry: per-session ephemeral Redpanda container via testcontainers (~8s session startup) |
| `indexer` | `fakeIndexer` — no real OpenSearch calls |
| `web` | Vitest, no backing services |

The athlete-svc tests still talk to a real broker (no mocks at the
broker boundary — that convention is load-bearing for catching
wire-format / schema bugs) — just a throwaway one rather than the
shared dev broker. Postgres for athlete-svc still uses dev compose, so
`make up` is required to run those tests.

## Service map

```
                 ┌─────────────┐         ┌────────────────────┐
   HTTP    ───►  │  api-svc    │ ──Avro──► Camp.V1            │
   /camps        │  (Go)       │ ──Avro──► CampUpdated.V1     │
                 └──────┬──────┘         └────────────────────┘
                        │ Postgres                  │
                        ▼                           ▼
                 ┌─────────────┐         ┌─────────────────────┐
   HTTP    ───►  │ athlete-svc │ ──Avro──► AthleteCreated.V1   │
   /athletes     │ (Python)    │ ──Avro──► AthleteRegistered.* │
                 └──────┬──────┘         └──────────┬──────────┘
                        │ Postgres                  │
                        │                           ▼
                        │                ┌─────────────────────┐
                        │                │      indexer        │
                        │                │      (Go)           │
                        │                │  Watermill consumer │
                        │                └──────────┬──────────┘
                        │                           │ documents
                        │                           ▼
                        │                ┌─────────────────────┐
                        │                │     OpenSearch      │
                        │                └──────────┬──────────┘
                        │                           │
                        │                           ▼
                        │                ┌─────────────────────┐
                        └───────────────►│        web          │
                                  HTTP   │   (React Router)    │
                                         └─────────────────────┘
```

## Ports (host)

| Service          | Port  |
|------------------|-------|
| Postgres         | 5532  |
| Kafka            | 19092 |
| Schema Registry  | 18081 |
| OpenSearch       | 19200 |
| api-svc          | 8001  |
| athlete-svc      | 8002  |
| indexer (search) | 8003  |
| web              | 8000  |
