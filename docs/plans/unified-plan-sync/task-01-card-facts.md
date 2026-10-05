---
id: "01-card-facts"
title: "Plan file card facts"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-CARD-001
acceptance_criteria:
  - AC-TASKS-PLAN-CARD-001.1
  - AC-TASKS-PLAN-CARD-001.2
  - AC-TASKS-PLAN-CARD-001.3
  - AC-TASKS-PLAN-CARD-001.4
system_design:
  - ../../specs/tasks/system-design/plan-file-card-facts.md
---

# Task 01: Plan file card facts

## Summary

Parse an optional `date` frontmatter key, project the date and the executor
into `metadata.card_display`, stop prefixing titles with the executor, and
write the facts with the other projected fields without touching other
metadata keys.

## In scope

- `format.PlanFile.Date` with `invalid_date` parse errors.
- `projectCardFacts` per the system design table; `projectTitle` without the
  prefix.
- Metadata compare-and-merge in `updateFields` and on task creation.
- Update `docs/specs/tasks/requirements/repository-plan-files.md`
  `AC-TASKS-PLAN-FILES-002.3` to drop the title prefix and point at this
  capability; update `docs/public/plan-files.md` for `date`.

## Out of scope

- Summary section (Task 02); card rendering.

## Acceptance

- Tests: valid and invalid dates; `date_kind` for every board status; no facts
  for `done`; executor fact without title prefix; other metadata keys kept;
  unchanged facts produce no `UpdateTask` call.

## Verification

```bash
cd apps/backend && go test ./internal/planfiles/...
make -C apps/backend lint
```

## Files likely touched

- `apps/backend/internal/planfiles/format/parse.go`, `format/types.go`
- `apps/backend/internal/planfiles/projection.go`, `sync_task.go`
- `apps/backend/internal/planfiles/fake_tasks_test.go` and tests
- `docs/specs/tasks/requirements/repository-plan-files.md`, `docs/public/plan-files.md`

## Dependencies

None. Run before or after Task 02, not in parallel.

## Results

- `(cd apps/backend && go test ./internal/planfiles/...)`: all four packages `ok` (last line `ok  github.com/kandev/kandev/internal/planfiles/scan`).
- `(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)`: `ok  github.com/kandev/kandev/internal/planfiles/scan	1.326s`.
- `(cd apps/backend && go test ./internal/backendapp/ -run PlanFiles -count=1)`: `ok  github.com/kandev/kandev/internal/backendapp	1.787s`.
- `make -C apps/backend lint`: `0 issues.`
- `(cd apps/backend && golangci-lint run ./internal/planfiles/... ./internal/backendapp/...)`: `0 issues.`
- `gofmt -l` on the changed Go packages: no output.
- `python3.12 scripts/list-docs.py validate`: `Validated 350 decisions and 1347 specifications.`
- `python3.12 scripts/lint-spec-files.py --all`: `All specification files passed.`

Notes: the sync pass owns the whole `card_display` object of a plan task; keys it does not project are removed, every other metadata key is kept. `projectCardFacts` in `planfiles/card_facts.go` assembles the facts from one helper per fact (`addDateFacts`, `addExecutorFact`); later facts add a helper to that list. No web file asserted the executor title prefix.
