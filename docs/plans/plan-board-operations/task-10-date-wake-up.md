---
id: "10-date-wake-up"
title: "Date wake-up and date sort"
status: done
wave: 10
depends_on: ["04-owner-decisions", "09-progress-and-flags"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-003
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-003.1
  - AC-TASKS-PLAN-BOARD-OPS-003.2
  - AC-TASKS-PLAN-BOARD-OPS-003.3
  - AC-TASKS-PLAN-BOARD-OPS-003.4
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 10: Date wake-up and date sort

## Summary

Move `waiting_external` and `deferred` plans to `waiting_owner` when their date arrives, with an owner note and a notification, and add the date sort to the board. Requires `PlanFile.Date` from unified plan sync task 01.

## In scope

- `wake.go`: the pass step with an injectable clock; file error `wake_failed`; metric `plan_files_wake_total`.
- `DateNotifier` interface, `notifications/service.HandlePlanDateReached`, event `plan_file.date_reached` in `AvailableEvents`, wiring in `backendapp`, the event label in the notification settings copy.
- `date_asc` in `kanbanSortValues`, `kanban-sort.ts`, and the comparator in `task-order.ts`; label in every locale.

## Out of scope

- Writing the date from the board.
- Per-plan notification settings.

## Acceptance

- With the clock at 2026-10-06, a `waiting_external` plan dated 2026-10-06 becomes `waiting_owner` with the note `- 2026-10-06 date reached: was waiting_external`, the task moves in the same pass, and one notification call is recorded; a second pass does nothing.
- A `queued` plan with a past date is not changed; with wake-up off no file is changed.
- In a column sorted by date, dated tasks come first in ascending order and undated tasks keep board order.

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
(cd apps/backend && go test ./internal/notifications/... ./internal/user/models/... -count=1)
(cd apps && pnpm --filter @kandev/web test -- lib/kanban/task-order.test.ts)
(cd apps && pnpm --filter @kandev/web test -- lib/kanban/kanban-sort.test.ts)
(cd apps/web && pnpm run typecheck)
(cd apps/web && pnpm run i18n:check)
(cd apps && pnpm --filter @kandev/web lint)
```

## Files likely touched

- apps/backend/internal/planfiles/wake.go, wake_test.go (new)
- apps/backend/internal/planfiles/sync_pass.go, sync_types.go, metrics.go
- apps/backend/internal/notifications/service/service.go, plan_date_test.go (new)
- apps/backend/internal/backendapp/services.go, helpers.go
- apps/backend/internal/user/models/kanban_view_preferences.go and its test
- apps/web/lib/kanban/kanban-sort.ts, task-order.ts and tests
- apps/web/src/locales/*/kanban.json and the notification settings namespace

## Dependencies

Task 04, Task 09

## Risks

- The notification event list is validated against provider subscriptions; existing providers do not subscribe to the new event until the owner enables it. Say so in the public docs.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Wake-up step `wakeDue` runs after `resolveAll` and before `applyTracked`, so the task move, card facts, and index reflect the new status in the same pass. "Today" is the server local time zone (`time.Local`, same as owner notes); tests cover 23:59:59 and 00:00:00 local. The note date is today, so a past-dated plan gets the wake day. The hash compare-and-swap failure is reported as `wake_failed` and retried next pass. Notifications are queued during the pass and delivered when the workspace lock is released (`releaser`). Plans whose task has an unwritten board edit are skipped until that edit is written back. Existing providers do not subscribe to `plan_file.date_reached` until the owner enables it (public docs belong to work order 12). The local provider shows the event through a new WS handler. The event label keys live in the `common` namespace because the harness-lint hook rejects any edit to a file named settings.json.

Verification (final output line):

- `(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)`: ok .../planfiles/scan 1.643s (all four packages ok)
- `make -C apps/backend lint`: 0 issues.
- `(cd apps/backend && go test ./internal/notifications/... ./internal/user/models/... -count=1)`: ok .../user/models 0.450s (all ok)
- `(cd apps && pnpm --filter @kandev/web test -- lib/kanban/task-order.test.ts)`: Tests 33 passed (33)
- `(cd apps && pnpm --filter @kandev/web test -- lib/kanban/kanban-sort.test.ts)`: Tests 7 passed (7)
- `(cd apps/web && pnpm run typecheck)`: tsc --noEmit, no errors
- `(cd apps/web && pnpm run i18n:check)`: i18n keys OK ... ja, ko, pt-pt, ru, zh-cn, zh-hk, zh-tw complete.
- `(cd apps && pnpm --filter @kandev/web lint)`: no warnings or errors
- `(cd apps/web && pnpm run i18n:ratchet)`: i18n new-code ratchet clean
- `go test ./internal/backendapp/...`: fails only TestBackendStartupConflictStopsBeforeSharedStateInitialization, TestBackendStartupExternalDatabaseConflictDiagnostic, TestSystemTemporaryCapacityRootsUseDisposableE2ERoot (macOS /var vs /private/var); the same three fail on a clean checkout of the parent commit.
- `golangci-lint run ./internal/planfiles/... ./internal/backendapp/... ./internal/notifications/... ./internal/user/...`: 0 issues.
- `python3.12 scripts/list-docs.py validate`: Validated 350 decisions and 1347 specifications.
- `python3.12 scripts/lint-spec-files.py --all`: All specification files passed.
