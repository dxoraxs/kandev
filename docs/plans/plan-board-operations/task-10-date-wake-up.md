---
id: "10-date-wake-up"
title: "Date wake-up and date sort"
status: pending
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

Pending.
