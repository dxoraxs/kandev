---
id: "07-plan-index"
title: "Plan index"
status: pending
wave: 7
depends_on: ["02-config-and-settings"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-006
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-006.1
  - AC-TASKS-PLAN-BOARD-OPS-006.2
  - AC-TASKS-PLAN-BOARD-OPS-006.3
  - AC-TASKS-PLAN-BOARD-OPS-006.4
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 07: Plan index

## Summary

Write a generated, marker-owned Markdown index into each scanned directory after a pass, only when its content changes.

## In scope

- `scan.WriteGenerated` with `ErrNotGenerated`.
- `index.go`: rendering from pass state; write step after `reorder`; file error `index_not_owned`; metric `plan_files_index_total`.
- `collect` skips the index file name before parsing so it is neither a plan file nor unadapted.

## Out of scope

- Custom index templates.

## Acceptance

- Two consecutive passes over unchanged files write the index once; the second pass leaves its modification time unchanged.
- A hand-written `INDEX.md` without the marker is not modified and appears in the file errors.
- Rows inside a status section follow the board order after a reorder write-back.

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
```

## Files likely touched

- apps/backend/internal/planfiles/index.go, index_test.go (new)
- apps/backend/internal/planfiles/scan/generated.go, generated_test.go (new)
- apps/backend/internal/planfiles/sync_pass.go, service_unadapted.go, metrics.go

## Dependencies

Task 02

## Risks

- The unadapted count is computed in two places (`collect` and `UnadaptedCounts`); both must skip the index file.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Pending.
