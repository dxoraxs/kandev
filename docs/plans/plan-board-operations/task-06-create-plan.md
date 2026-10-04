---
id: "06-create-plan"
title: "Create a plan from the board"
status: pending
wave: 6
depends_on: ["02-config-and-settings"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-007
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-007.1
  - AC-TASKS-PLAN-BOARD-OPS-007.2
  - AC-TASKS-PLAN-BOARD-OPS-007.3
  - AC-TASKS-PLAN-BOARD-OPS-007.4
  - AC-TASKS-PLAN-BOARD-OPS-007.5
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 06: Create a plan from the board

## Summary

Create a plan file from a dialog on the plan board and return once its task exists.

## In scope

- `scan.CreateFile` (exclusive create inside the root, creates the directory).
- `Service.CreatePlan` with file-name derivation and validation; `POST /plans`.
- Web: `createPlan` client, `useCreatePlan`, `NewPlanDialog`, the header button shown only on the plan board, the phone entry in board display options, copy in every locale.

## Out of scope

- Editing or deleting plan files from the board.
- Project-specific numbering.

## Acceptance

- Creating `Заказ в чате` with no file name produces `plan-<YYYYMMDD-HHMM>.md`; creating `Fix: totals #2` produces `fix-totals-2.md`; both parse back to the given title.
- A second create with the same file name returns 409 `file_exists` and leaves the first file unchanged.
- The response carries the task ID of the new plan task, and the task is in the queue step.

## ASCII UI preview

UI-03: New plan. Full preview: [plan.md](plan.md#ascii-ui-preview). Covers AC-007.1, 007.5.

```text
+------------------------- New plan --------------------------+
| Repository   [ dmhive                                   v ] |
| Directory    [ docs/plans                               v ] |
| Title        [____________________________________________] |
| File name    [ my-plan.md                                 ] |
| Priority     [ Medium v ]      Executor [ (none)        v ] |
| Body (optional)                                             |
|                                     [ Cancel ] [ Create ]   |
+-------------------------------------------------------------+
Phone: full-height surface, sticky Create button.
```

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
(cd apps && pnpm --filter @kandev/web test -- components/kanban/new-plan-dialog.test.tsx)
(cd apps && pnpm --filter @kandev/web test -- lib/api/domains/plan-files-api.test.ts)
(cd apps/web && pnpm run typecheck)
(cd apps/web && pnpm run i18n:check)
(cd apps && pnpm --filter @kandev/web lint)
```

## Files likely touched

- apps/backend/internal/planfiles/scan/create.go, create_test.go (new)
- apps/backend/internal/planfiles/create.go, create_test.go, handlers_ops.go
- `apps/web/lib/api/domains/plan-files-api.ts`
- apps/web/hooks/domains/plans/use-create-plan.ts (new)
- apps/web/components/kanban/new-plan-dialog.tsx, new-plan-dialog.test.tsx (new)
- apps/web/components/kanban/kanban-header.tsx, mobile-display-options.tsx
- `apps/web/src/locales/*/planFiles.json`

## Dependencies

Task 02

## Risks

- `kanban-header.tsx` has separate desktop, tablet, and phone headers; add the entry to each or the button disappears at tablet width.
- The executor choices come from the config executor steps; with none configured the field is hidden.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Pending.
