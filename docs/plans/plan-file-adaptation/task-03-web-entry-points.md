---
id: "03-web-entry-points"
title: "Plan adaptation web entry points"
status: done
wave: 3
depends_on: ["01-unadapted-detection", "02-maintenance-launcher"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-ADAPT-001
  - REQ-TASKS-PLAN-ADAPT-002
  - REQ-TASKS-PLAN-ADAPT-003
acceptance_criteria:
  - AC-TASKS-PLAN-ADAPT-001.1
  - AC-TASKS-PLAN-ADAPT-001.2
  - AC-TASKS-PLAN-ADAPT-002.1
  - AC-TASKS-PLAN-ADAPT-002.2
  - AC-TASKS-PLAN-ADAPT-002.3
  - AC-TASKS-PLAN-ADAPT-003.1
  - AC-TASKS-PLAN-ADAPT-003.2
  - AC-TASKS-PLAN-ADAPT-003.3
  - AC-TASKS-PLAN-ADAPT-003.4
system_design:
  - ../../specs/tasks/system-design/plan-file-adaptation.md
---

# Task 03: Plan adaptation web entry points

## Summary

Show unadapted plan files per repository in the Plan files section, show the
count in the status line, offer adaptation after a repository is added, and
start the adaptation task from both places. Follow `/mobile-parity`.

## In scope

- `plan-files-api.ts`: `getUnadaptedPlanFiles`; `unadapted` in
  `PlanFilePassCounts`.
- `startRepositoryMaintenanceTask(repositoryId, kind)` client with typed reasons.
- `PlanFilesUnadaptedRows` in `plan-files-section.tsx` (renders with sync
  disabled too), `PlanFilesStatus` count.
- `AdaptPlansOfferDialog` opened from `saveNewRepository` in
  `workspace-repositories-client.tsx`.
- Navigation with `linkToTask`; info toast for `existing: true`; localized
  errors for each reason.
- Copy in English, pseudo, and the seven translated catalogs; no em dash.
- `docs/public/plan-files.md`: short "Adapting existing plans" section.
- E2E: `settings/plan-files-adapt.spec.ts` and
  `settings/mobile-plan-files-adapt.spec.ts` with the mock agent.

## Out of scope

- Backend changes; the cleanup button.

## Acceptance

- The section and dialog match UI-01 and UI-02 on desktop and phone; phone
  actions are full width with rendered height of at least 44px.
- Adding a repository with unadapted plans shows the offer; Not now has no
  side effects; Adapt with agent opens the created task; a second start opens
  the same task.
- With the flag off, no row, count, or dialog renders.

## ASCII UI preview

See [UI-01 and UI-02 in the plan](plan.md#ascii-ui-preview). Excerpt:

```text
| Plan files not on the board                          |
| beaver-blocks   16 files without board status        |
|                               [ Adapt with agent ]   |
```

Phone: name and count stacked over a full-width 44px action; the offer is a
bottom sheet with stacked actions.

## Verification

```bash
cd apps && pnpm --filter @kandev/web test -- lib/api/domains/plan-files-api.test.ts components/settings/plan-files-section.test.tsx
cd apps/web && pnpm run typecheck
cd apps/web && pnpm run i18n:check
cd apps && pnpm --filter @kandev/web lint
make build-web build-backend
cd apps/web && pnpm e2e:run -- tests/settings/plan-files-adapt.spec.ts tests/settings/mobile-plan-files-adapt.spec.ts
```

## Files likely touched

- `apps/web/lib/api/domains/plan-files-api.ts`, `workspace-api.ts` or a repository API module
- `apps/web/hooks/domains/settings/use-plan-files.ts`
- `apps/web/components/settings/plan-files-section.tsx`, `plan-files-status.tsx`
- `apps/web/components/settings/plan-files-unadapted-rows.tsx` (new)
- `apps/web/components/settings/adapt-plans-offer-dialog.tsx` (new)
- `apps/web/app/settings/workspace/workspace-repositories-client.tsx`
- `apps/web/src/locales/*/settings.json` (namespace used by the Plan files section)
- `apps/web/e2e/helpers/plan-files.ts`, the two new specs
- `docs/public/plan-files.md`

## Dependencies

Tasks 01 and 02.
