---
id: "05-write-back"
title: "Board write-back"
status: done
wave: 4
depends_on: ["04-sync-pass"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-FILES-003
acceptance_criteria:
  - AC-TASKS-PLAN-FILES-003.1
  - AC-TASKS-PLAN-FILES-003.2
  - AC-TASKS-PLAN-FILES-003.3
  - AC-TASKS-PLAN-FILES-003.5
  - AC-TASKS-PLAN-FILES-003.6
system_design:
  - ../../specs/tasks/system-design/repository-plan-files.md
---

# Task 05: Board write-back

## Summary

Subscribe to task move, reorder, and update events; detect board edits by
divergence from the `synced_*` values; write `board`, `priority`, and
`order` to the plan file with a hash compare-and-swap; record a notice on
conflict.

## In scope

- `WriteBackSubscriber` registration beside the poller, under the flag.
- Shared order routine (`order.go`): keep the keys of the longest already
  sorted run, give midpoint or edge ±1 values to the others, `10, 20, …` from
  the top through the moved task when a kept neighbour above lacks `order`.
- Pass-side handling of unhandled board edits: a task that diverges from its
  `synced_*` values while the file hash is unchanged is written back by the
  pass, not reverted; with a changed hash the file wins and `notice` is set.
  The same for a step whose board order differs while all its files are
  unchanged.
- Handoff steps: no write, synced step updated.
- Metric `plan_files_writeback_total`.

## Out of scope

- Editing title, body, executor, or dependencies.

## Acceptance

- A human move to the `done` step writes `board: done` and nothing else; the
  next pass makes no task changes.
- A file edited after the last read is not overwritten; the task shows the
  notice and returns to the file's state on the next pass.
- A full sync pass, including its own moves and reorders, triggers zero file
  writes.
- A board move or reorder that a pass sees before the subscriber does is
  written to the file, not reverted (AC-003.5 last sentence).

## Verification

```bash
cd apps/backend && go test ./internal/planfiles/... -count=1 -race
cd apps/backend && golangci-lint run ./internal/planfiles/...
```

## Files likely touched

- `apps/backend/internal/planfiles/writeback.go`, `writeback_test.go`
- `apps/backend/internal/planfiles/order.go`, `order_test.go`
- `apps/backend/internal/planfiles/sync_task.go`, `sync_order.go`
- `apps/backend/internal/backendapp/main.go`

## Dependencies

Task 04.

## Risks

- Event ordering on the bus is asynchronous; the subscriber reloads the task
  under the workspace mutex instead of trusting event payloads.
