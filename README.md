# em-challenge

A live-pairing technical interview challenge for the **ecosystem team**.

Suitable for any engineering hire — IC software engineers (mid → staff)
through hands-on engineering managers. The four sequential challenges
probe the same set of judgment calls at any level: codegen-driven
navigation, Avro schema evolution under cross-team rollout constraints,
idempotent indexing with backfill, and PR review with seeded
correctness issues.

Calibration is up to the interviewer — see `INTERVIEWER_GUIDE.md` for
the rubric and which prompts to lean on for IC vs. EM signal. (The
manager-flavor probes in Ch2 / Ch4 are optional and can be dropped or
softened for IC interviews.)

## Start here

Your sprint board is in [`challenges/`](./challenges/). Read all four
tickets before writing code — they're sequential and each unblocks the
next.

## How to read this repo

> **These four apps live as separate repos in real life — owned by different teams, deployed independently. They're packaged together here only so you can clone once. When you're reasoning about cross-app changes, treat them as separate.**

- `apps/api-svc/` — Go service (Gin, sqlc, Goose migrations, oapi-codegen, Watermill+Avro). Owns `Camp` reference data; publishes `Camp.V1` events.
- `apps/athlete-svc/` — Python service (Django, confluent-kafka, transactional outbox). Owns `Athlete` and `Registration`; publishes `AthleteCreated.V1` and (will publish) `AthleteRegistered.V1`.
- `apps/indexer/` — Go service (Cobra, Watermill consumer, OpenSearch, Goose). Consumes events and indexes a denormalized `camps` document.
- `apps/web/` — TypeScript app (React Router v7, Tailwind, openapi-client). Search UI for camps.

Shared:

- `schemas/` — Avro schemas and `schemas.yaml` mapping subjects → files → compatibility policy.
- `compose.yml` — Postgres, Kafka (KRaft single-node), Confluent Schema Registry, OpenSearch.
- `Makefile` — top-level orchestration. Each app has its own `Makefile` for app-specific work.

## Quick start

```bash
make up                # start infra (compose), wait for health
make register-schemas  # POST Avro schemas to Schema Registry
make test              # run all app test suites
```

You'll spend most of your time inside one or two `apps/*` directories. Each has its own README with the commands you need.

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
