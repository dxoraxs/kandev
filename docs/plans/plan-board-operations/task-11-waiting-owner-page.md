---
id: "11-waiting-owner-page"
title: "Waiting for owner page"
status: pending
wave: 11
depends_on: ["04-owner-decisions"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-009
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-009.1
  - AC-TASKS-PLAN-BOARD-OPS-009.2
  - AC-TASKS-PLAN-BOARD-OPS-009.3
  - AC-TASKS-PLAN-BOARD-OPS-009.4
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 11: Waiting for owner page

## Summary

List plans that wait for the owner across all accessible workspaces on one page with a navigation entry and count.

## In scope

- `WorkspaceLister` dependency, `Service.WaitingOwner`, `GET /waiting-owner`.
- Web: client, `useWaitingOwner`, route `plans-waiting`, page component, primary-nav entry with badge, phone navigation-sheet entry, copy in every locale.

## Out of scope

- Decision buttons inside the list (the row opens the task, which has the bar).
- Real-time updates beyond refetch on focus and on plan task events.

## Acceptance

- With two enabled workspaces, the endpoint returns the waiting plans of both sorted by date with undated last; a workspace the caller cannot access contributes nothing and is not listed as failed.
- A workspace whose rows fail to load is returned in `failed_workspaces` and the page names it while showing the others.
- With the flag off the route and both navigation entries are absent.

## ASCII UI preview

UI-04: Waiting for owner page. Full preview: [plan.md](plan.md#ascii-ui-preview). Covers AC-009.1 to 009.4.

```text
Waiting for owner (5)
| Workspace | Plan                         | Repository | Date  | Executor |
| dmhive    | 06a Instagram data deletion  | dmhive     | 5 Oct | Claude   |
| kandev    | Plan board operations        | kandev     |       |          |
! Could not load: tg-hub

Phone: one full-width row per plan, at least 44 px.
```

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
(cd apps && pnpm --filter @kandev/web test -- app/plans-waiting/plans-waiting-page-client.test.tsx)
(cd apps && pnpm --filter @kandev/web test -- src/spa-routes.test.ts)
(cd apps/web && pnpm run typecheck)
(cd apps/web && pnpm run i18n:check)
(cd apps && pnpm --filter @kandev/web lint)
```

## Files likely touched

- apps/backend/internal/planfiles/waiting.go, waiting_test.go, handlers_ops.go
- `apps/backend/internal/backendapp/services.go`
- `apps/web/lib/api/domains/plan-files-api.ts`
- apps/web/hooks/domains/plans/use-waiting-owner.ts (new)
- apps/web/app/plans-waiting/plans-waiting-page-client.tsx and test (new)
- apps/web/src/spa-routes.tsx, src/plans-waiting-route.tsx (new)
- `apps/web/components/app-sidebar/app-sidebar-primary-nav.tsx`
- `apps/web/components/navigation/mobile-sidebar-layout-navigation.tsx`
- `apps/web/src/locales/*/planFiles.json`

## Dependencies

Task 04

## Risks

- Without unified plan sync task 01 the date column is empty; the page must not depend on it.
- The route is flag-gated like the Needs You inbox; copy its `SpaRouteOptions` handling.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Pending.
