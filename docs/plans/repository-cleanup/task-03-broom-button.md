---
id: "03-broom-button"
title: "Repository cleanup broom button"
status: pending
wave: 3
depends_on: ["02-cleanup-kind-and-prompt"]
plan: "plan.md"
requirements:
  - REQ-WORKSPACES-REPO-CLEANUP-001
acceptance_criteria:
  - AC-WORKSPACES-REPO-CLEANUP-001.1
  - AC-WORKSPACES-REPO-CLEANUP-001.2
  - AC-WORKSPACES-REPO-CLEANUP-001.3
  - AC-WORKSPACES-REPO-CLEANUP-001.4
  - AC-WORKSPACES-REPO-CLEANUP-001.5
  - AC-WORKSPACES-REPO-CLEANUP-001.6
  - AC-WORKSPACES-REPO-CLEANUP-001.7
  - AC-WORKSPACES-REPO-CLEANUP-001.8
system_design:
  - ../../specs/workspaces/system-design/repository-cleanup.md
---

# Task 03: Broom button

## Summary

Add the icon-only broom action to local repository rows in workspace
repository settings, the confirmation dialog, and navigation to the cleanup
task. Follow `/mobile-parity`.

## In scope

- `RepositoryCleanupButton` in `RepositoryPreview` (before Edit), `IconBroom`,
  `aria-label`, fine-pointer tooltip, `stopPropagation`, 28px desktop and 44px
  phone/coarse-pointer target; hidden for non-local repositories, read-only
  mode, and when `useFeature("repositoryCleanup")` is off.
- `RepositoryCleanupDialog` per UI-02 using the shared `Dialog`.
- `startRepositoryMaintenanceTask(id, "repository_cleanup")` (client from the
  adaptation plan), navigation with `linkToTask`, info toast for an existing
  task, localized reason errors.
- Copy in English, pseudo, and the seven translated catalogs; no em dash.
- `docs/public/` section on repository cleanup (via `/docs-maintainer`).
- E2E: `settings/repository-cleanup.spec.ts` and
  `settings/mobile-repository-cleanup.spec.ts` with the mock agent.

## Out of scope

- Backend changes.

## Acceptance

- Matches UI-01 and UI-02; on a phone the broom's rendered hit target is at
  least 44px and dialog actions are full width.
- Confirming creates the task and opens it; confirming again opens the same
  task; Cancel creates nothing.
- With the flag off or for a remote repository, the button does not render.

## ASCII UI preview

See [UI-01 and UI-02 in the plan](plan.md#ascii-ui-preview). Excerpt:

```text
| [git] city_companion  [Local]          [broom] [Edit] [Delete] |
```

Phone: the action row wraps below the name; the broom is a 44px icon button.

## Verification

```bash
cd apps && pnpm --filter @kandev/web test -- components/settings/repository-cleanup
cd apps/web && pnpm run typecheck
cd apps/web && pnpm run i18n:check
cd apps && pnpm --filter @kandev/web lint
make build-web build-backend
cd apps/web && pnpm e2e:run -- tests/settings/repository-cleanup.spec.ts
cd apps/web && pnpm e2e:run -- --project mobile-chrome tests/settings/mobile-repository-cleanup.spec.ts
```

## Files likely touched

- `apps/web/components/settings/repository-card-preview.tsx`
- `apps/web/components/settings/repository-cleanup-button.tsx` (new)
- `apps/web/components/settings/repository-cleanup-dialog.tsx` (new)
- `apps/web/src/locales/*/workspaces.json`
- `apps/web/e2e/tests/settings/` (two new specs)
- `docs/public/` (repository cleanup section)

## Dependencies

Task 02.
