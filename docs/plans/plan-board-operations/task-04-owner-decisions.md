---
id: "04-owner-decisions"
title: "Owner decisions"
status: pending
wave: 4
depends_on: ["02-config-and-settings"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-002
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-002.1
  - AC-TASKS-PLAN-BOARD-OPS-002.2
  - AC-TASKS-PLAN-BOARD-OPS-002.3
  - AC-TASKS-PLAN-BOARD-OPS-002.4
  - AC-TASKS-PLAN-BOARD-OPS-002.5
  - AC-TASKS-PLAN-BOARD-OPS-002.6
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 04: Owner decisions

## Summary

Add the decision endpoint that writes the status and an owner note in one compare-and-swap write and moves the task, and the decision bar on the task page for desktop and phone.

## In scope

- `composeEdit` in `edit.go`; `Service.Decide`; `POST /tasks/:taskId/decision`; metric `plan_files_decision_total`.
- Web: `decidePlan` client, `usePlanBoard` and `usePlanDecision` hooks, `PlanDecisionBar` mounted under the task top bar, phone drawer in the description view, copy in every locale.

## Out of scope

- The waiting-owner page (task 11).
- Starting an agent after a return.

## Acceptance

- Return with comment `fix the totals` on a file without a notes section adds `## Owner notes` and `- <today> returned: fix the totals` at the end, sets `board: queued`, and the task is in the queue step when the response arrives.
- A decision on a file edited since the last read returns 409 `file_changed` and writes nothing.
- The bar is absent for a plan task outside the waiting-owner step and for a non-plan task; on a phone the drawer buttons are at least 44 px high.

## ASCII UI preview

UI-01: Plan decision bar. Full preview: [plan.md](plan.md#ascii-ui-preview). Covers AC-002.1, 002.6.

```text
| (i) This plan waits for your decision.        [ Return ] [ Accept v ]|

Phone drawer:
| Plan decision          [ x ] |
| Comment                      |
| [__________________________] |
| [        Accept and close  ] |
| [   Accept, back to queue  ] |
| [          Return          ] |
```

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
(cd apps && pnpm --filter @kandev/web test -- components/task/plan-decision-bar.test.tsx)
(cd apps && pnpm --filter @kandev/web test -- lib/api/domains/plan-files-api.test.ts)
(cd apps/web && pnpm run typecheck)
(cd apps/web && pnpm run i18n:check)
(cd apps && pnpm --filter @kandev/web lint)
```

## Files likely touched

- apps/backend/internal/planfiles/edit.go, decision.go, handlers_decision.go, decision_test.go, handlers_decision_test.go (new)
- apps/backend/internal/planfiles/handlers.go, metrics.go
- `apps/web/lib/api/domains/plan-files-api.ts`
- apps/web/hooks/domains/plans/use-plan-board.ts, use-plan-decision.ts (new)
- apps/web/components/task/plan-decision-bar.tsx, plan-decision-drawer.tsx, plan-decision-bar.test.tsx (new)
- apps/web/components/task/task-page-content.tsx, components/task/mobile/session-mobile-layout.tsx
- `apps/web/src/locales/*/planFiles.json`

## Dependencies

Task 02

## Risks

- The task page has many layouts; mount the bar in one place that both sessionless and session tasks render, and verify both in the component test.
- The route takes a task ID, not a workspace ID; authorize the workspace of the row before anything else.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Pending.
