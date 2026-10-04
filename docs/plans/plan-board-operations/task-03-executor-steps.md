---
id: "03-executor-steps"
title: "Executor steps"
status: pending
wave: 3
depends_on: ["02-config-and-settings"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-001
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-001.2
  - AC-TASKS-PLAN-BOARD-OPS-001.3
  - AC-TASKS-PLAN-BOARD-OPS-001.4
  - AC-TASKS-PLAN-BOARD-OPS-001.5
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 03: Executor steps

## Summary

Make a step with an executor name write `executor` on entry, refuse a plan another executor already holds, and keep the pass from leaving a task in an executor step that no longer matches the file.

## In scope

- `boardKeys`/`writeBoardEdit`: `executor` key on entry into an executor step; quoting of names that are not plain YAML scalars.
- Claim conflict: no write, move back with the system actor, notice `plan is already taken by <executor>`.
- `desiredStep`/`staysInHandoff`: executor equality rule; unnamed handoff steps unchanged.

## Out of scope

- Stopping a session started by the step entry.
- Changes to the built-in Plans template.

## Acceptance

- Moving a `queued` plan into the executor step of `Codex` writes `executor: Codex` and no other byte; the next pass changes nothing.
- Moving an `in_progress` plan held by `Claude` into the `Codex` step writes nothing, returns the task to the in-progress step, and the description header shows the notice.
- A full pass over a board with tasks parked in executor and unnamed handoff steps produces zero writes and zero moves.

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
```

## Files likely touched

- apps/backend/internal/planfiles/writeback_core.go, writeback.go
- `apps/backend/internal/planfiles/sync_task.go`
- apps/backend/internal/planfiles/writeback_executor_test.go, sync_executor_test.go (new)

## Dependencies

Task 02

## Risks

- `sync_task.go` is 361 lines and shared with unified plan sync; add helpers in a new file `executor.go` and keep edits in `sync_task.go` to call sites.
- The write-back subscriber and the pass both move tasks; the move back must update `SyncedStepID` under the workspace lock so it is not seen as a board edit.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Pending.
