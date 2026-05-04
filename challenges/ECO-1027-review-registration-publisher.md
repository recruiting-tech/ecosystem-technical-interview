# ECO-1027 — Review registration-publisher PR (athlete-svc)

| | |
|---|---|
| **Type** | Code Review |
| **Priority** | High — blocking athlete-svc release cut |
| **Sprint** | 24.05 |
| **Status** | Awaiting review |
| **Reporter** | Maya S. (athlete-svc tech lead) |
| **Assignee** | _you_ |
| **Components** | `apps/athlete-svc` |

## Description

A senior IC on the athlete-svc team (which would be its own repo in real
life) has put up a PR adding the long-pending registration publisher.
Please review with the same standards you'd apply to any production
change.

- **Branch:** `pr/registration-publisher-WIP`
- **PR description:** [`PR_DESCRIPTION.md`](../PR_DESCRIPTION.md) at the repo root
- **CI:** green

To pull up the diff:

```bash
git checkout pr/registration-publisher-WIP
git diff main..pr/registration-publisher-WIP
# Or if you prefer a tool: open the branch in your usual diff viewer.
```

## Acceptance Criteria

- [ ] Concrete review comments, prioritized **highest-impact-first** (don't list nits before correctness issues)
- [ ] For each issue: a one-sentence _why it's a problem_, and a recommended fix
- [ ] Where the existing repo has a relevant pattern, cite the file path
- [ ] Be prepared to discuss how you'd handle disagreement with the author

## Author note (visible in the PR)

> _Author (senior IC):_ I considered the outbox pattern but publishing
> inline is simpler and we've never lost a message in this service.
> Happy to revisit if reviewers feel strongly.

## Reminder from the team norms doc

> Reviewers own the merge decision regardless of authorship seniority.
> If you'd block, block; if you'd ask for changes, ask for changes;
> only ✅ if you'd actually ship.

## How to verify your review

- **Confirm CI green for yourself:** `cd apps/athlete-svc && make test`. You'll find that both tests pass — including the new `tests/test_registrations.py`. If a green test suite would be enough to ship this change, what's missing? (Hint: open `tests/test_athlete_created_outbox.py` next to it.)
- **Stage the diff cleanly:** `git diff main..pr/registration-publisher-WIP -- apps/athlete-svc schemas`. Eleven files. Read in order: schema → event dataclass → kafka_publisher mapping → views → migration → test.
- **For each issue you flag, cite the existing pattern.** The repo *already* shows the right way to do most of this — your review is stronger when the citation is concrete (file path + line range).
- **Don't run the changes against the live stack.** Some seeded issues only surface at runtime (e.g. the `int` overflow on a 2026 timestamp); spotting them in code review without ever running them is exactly the signal we want.

## Comments

> **Maya S.** — _1 hour ago_
> I'm out for ~3 hrs in a planning offsite. If you have questions for
> me put them on the PR and I'll answer when I'm back. Don't wait on
> me to merge — if it's good, ship it; if it's not, request changes.
