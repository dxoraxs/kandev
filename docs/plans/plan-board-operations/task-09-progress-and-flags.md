---
id: "09-progress-and-flags"
title: "Progress and card flags"
status: pending
wave: 9
depends_on: ["05-git-state-and-commit", "08-dependencies"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-005
  - REQ-TASKS-PLAN-BOARD-OPS-008
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-005.1
  - AC-TASKS-PLAN-BOARD-OPS-005.2
  - AC-TASKS-PLAN-BOARD-OPS-005.3
  - AC-TASKS-PLAN-BOARD-OPS-005.4
  - AC-TASKS-PLAN-BOARD-OPS-005.5
  - AC-TASKS-PLAN-BOARD-OPS-008.1
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 09: Progress and card flags

## Summary

Compute progress and the `stale`, `open_items`, and `uncommitted` flags in the pass, write them through the `card_display` merge, and render flags on the card. Requires unified plan sync task 01 on the branch.

## In scope

- Tracked item reading through `scan` (`FileInfo.ModTime`, directory listing of `task-*.md`), file error `invalid_track`.
- `projectCardFacts` additions: `progress`, `flags`.
- Web: `flags` in `card-display.ts`, pills in `KanbanCardHintRow`, copy in `kanban.json` in every locale.
- Update the contract table of `docs/specs/tasks/system-design/kanban-card-display-hints.md` with `flags`.

## Out of scope

- The `date` and `executor` facts (unified plan sync task 01).
- Subtasks for tracked items.

## Acceptance

- A plan with 2 of 5 items done in its body and a tracked directory with 1 of 3 work orders done shows `3/8`; the same plan with `board: done` shows no progress and the `open_items` flag.
- An `in_progress` plan whose file and tracked items are older than the threshold gets `stale`; touching a tracked file or a running turn removes it on the next pass.
- A pass over unchanged files and unchanged git state sends no task update.

## ASCII UI preview

UI-05: Card flags. Full preview: [plan.md](plan.md#ascii-ui-preview). Covers AC-005.5.

```text
| Title of the plan                            |
| [5 Oct] [Claude] [3/8] [Stale] [Uncommitted] |
```

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
(cd apps && pnpm --filter @kandev/web test -- lib/kanban/card-display.test.ts)
(cd apps && pnpm --filter @kandev/web test -- components/kanban-card-display-hints.test.tsx)
(cd apps/web && pnpm run typecheck)
(cd apps/web && pnpm run i18n:check)
(cd apps && pnpm --filter @kandev/web lint)
```

## Files likely touched

- apps/backend/internal/planfiles/projection.go, progress.go (new), projection_progress_test.go, projection_flags_test.go (new)
- apps/backend/internal/planfiles/scan/scan.go, read.go
- apps/backend/internal/planfiles/sync_task.go, sync_pass.go
- apps/web/lib/kanban/card-display.ts, card-display.test.ts
- apps/web/components/kanban-card-display-hints.tsx, kanban-card-display-hints.test.tsx
- `apps/web/src/locales/*/kanban.json`
- docs/specs/tasks/system-design/kanban-card-display-hints.md

## Dependencies

Task 05, Task 08

## Risks

- Blocked until `projectCardFacts` and the metadata compare-and-merge exist on the branch. Do not write a second merge.
- `cardDisplayFromMetadata` returns `undefined` unless one known field is valid; include `flags` in that condition.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Pending.
