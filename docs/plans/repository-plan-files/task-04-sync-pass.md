---
id: "04-sync-pass"
title: "Sync pass and poller"
status: done
wave: 3
depends_on: ["01-plan-file-format", "02-repository-scanner", "03-store-flag-config-api"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-FILES-002
  - REQ-TASKS-PLAN-FILES-003
  - REQ-TASKS-PLAN-FILES-004
  - REQ-TASKS-PLAN-FILES-005
  - REQ-TASKS-PLAN-FILES-006
acceptance_criteria:
  - AC-TASKS-PLAN-FILES-002.1
  - AC-TASKS-PLAN-FILES-002.2
  - AC-TASKS-PLAN-FILES-002.3
  - AC-TASKS-PLAN-FILES-002.4
  - AC-TASKS-PLAN-FILES-002.5
  - AC-TASKS-PLAN-FILES-002.6
  - AC-TASKS-PLAN-FILES-002.7
  - AC-TASKS-PLAN-FILES-002.8
  - AC-TASKS-PLAN-FILES-003.6
  - AC-TASKS-PLAN-FILES-004.1
  - AC-TASKS-PLAN-FILES-004.2
  - AC-TASKS-PLAN-FILES-004.3
  - AC-TASKS-PLAN-FILES-005.3
  - AC-TASKS-PLAN-FILES-006.3
system_design:
  - ../../specs/tasks/system-design/repository-plan-files.md
---

# Task 04: Sync pass and poller

## Summary

Implement `Service.SyncWorkspace`, the projection rules, adoption by external
identifier, archive and unarchive, step ordering, the running-turn and handoff
guards, the 60-second poller, and `POST /api/v1/plan-files/sync`.

## In scope

- Per-workspace mutex shared with Task 05.
- Projection: title prefix and truncation via `contract.TaskTitleMaxLength`,
  description header and body, priority, desired order key.
- Create with `ExternalID` and `StartAgent: false`; update with metadata merge;
  move with actor `system`; `ReorderStepTasks` on the `admitted` band.
- Local-source repositories only; duplicate external identifiers fail both
  files.
- Pass summary and file errors stored for the config response; metrics
  `plan_files_pass_total` and `plan_files_file_errors_total`.
- Poller start and stop in `backendapp/main.go`.

## Out of scope

- Writing to files; the settings UI.

## Acceptance

- Two consecutive passes over unchanged files: the second makes zero task
  service calls that write and publishes zero task events.
- A file with `external_id` adopts an archived task on another workflow:
  same task ID, unarchived, on the plan board.
- A task whose session is `RUNNING` is not moved; after the session settles,
  the next pass moves it.

## Verification

```bash
cd apps/backend && go test ./internal/planfiles/... -count=1 -race
cd apps/backend && golangci-lint run ./internal/planfiles/... ./internal/backendapp/...
```

## Files likely touched

- `apps/backend/internal/planfiles/service_sync.go`, `service_sync_test.go`
- `apps/backend/internal/planfiles/projection.go`, `projection_test.go`
- `apps/backend/internal/planfiles/poller.go`, `poller_test.go`
- `apps/backend/internal/planfiles/metrics.go`
- `apps/backend/internal/planfiles/handlers.go`
- `apps/backend/internal/backendapp/main.go`

## Dependencies

Tasks 01, 02, 03.

## Risks

- Board list endpoints may include descriptions; confirm payload size with a
  200 KB plan before closing the task.
- Reordering must not disturb non-plan tasks in the same step.
