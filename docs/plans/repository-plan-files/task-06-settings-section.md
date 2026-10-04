---
id: "06-settings-section"
title: "Plan files settings section"
status: done
wave: 4
depends_on: ["04-sync-pass"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-FILES-005
acceptance_criteria:
  - AC-TASKS-PLAN-FILES-005.1
  - AC-TASKS-PLAN-FILES-005.2
  - AC-TASKS-PLAN-FILES-005.3
  - AC-TASKS-PLAN-FILES-005.5
  - AC-TASKS-PLAN-FILES-005.6
system_design:
  - ../../specs/tasks/system-design/repository-plan-files.md
---

# Task 06: Plan files settings section

## Summary

Add the Plan files section to the workspace Workflows tab with the enable
switch, board selection and creation, status mapping, directory editor, last
pass status with file errors, and Sync now. Follow `/mobile-parity`.

## In scope

- `plan-files-api.ts` client and tests.
- `plan-files-section.tsx` and children, gated by `useFeature("planFiles")`.
- Copy in English, pseudo, and the seven translated catalogs; no em dash.

## Out of scope

- Workflow editor changes; board rendering changes.

## Acceptance

- The section matches `UI-01` structure on desktop and phone.
- Save is disabled until every visible status has a step; errors from the API
  show inline.
- With the flag off, nothing renders.

## ASCII UI preview

See [UI-01 in the plan](plan.md#ascii-ui-preview). Desktop excerpt:

```text
| Board      [ Plans          v ]  [Create Plans board] |
| queued            [ Queue              v ]            |
| Last sync 13:42  ok   created 2  updated 1  ...       |
|   ! city_companion  docs/superpowers/plans/x.md  ...  |
|                                  [Sync now] [Save]    |
```

Phone: label above full-width select; actions stacked full width.

## Verification

```bash
cd apps && pnpm --filter @kandev/web test -- components/settings/plan-files-section.test.tsx lib/api/domains/plan-files-api.test.ts
cd apps/web && pnpm run typecheck
cd apps/web && pnpm run i18n:check
cd apps && pnpm --filter @kandev/web lint
```

## Files likely touched

- `apps/web/lib/api/domains/plan-files-api.ts`, `plan-files-api.test.ts`
- `apps/web/components/settings/plan-files-section.tsx`, `plan-files-section.test.tsx`
- `apps/web/components/settings/plan-files-mapping.tsx`
- `apps/web/components/settings/plan-files-status.tsx`
- `apps/web/app/settings/workspace/workspace-workflows-client.tsx`
- `apps/web/src/locales/*/settings.json` (or the namespace the workflow-sync section uses)

## Dependencies

Task 04 (pass status in the config response).

## Risks

- `workspace-workflows-client.tsx` is near the TS size limits; mount a single
  component and keep logic in the new files.
