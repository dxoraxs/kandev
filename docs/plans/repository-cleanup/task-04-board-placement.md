---
id: "04-board-placement"
title: "Move the broom to the board top bar"
status: done
wave: 4
depends_on: ["03-broom-button"]
plan: "plan.md"
requirements:
  - REQ-WORKSPACES-REPO-CLEANUP-001
acceptance_criteria:
  - AC-WORKSPACES-REPO-CLEANUP-001.1
  - AC-WORKSPACES-REPO-CLEANUP-001.6
  - AC-WORKSPACES-REPO-CLEANUP-001.7
system_design:
  - ../../specs/workspaces/system-design/repository-cleanup.md
---

# Task 04: Board placement

## Summary

Move the cleanup action from repository settings rows to the kanban page's top
bar. The action targets the repository selected in the board's repository
filter. Owner decision (2026-10-05): the broom lives only on the board.

## In scope

- `resolveCleanupTarget` (pure) and `useBoardCleanupAction` per the system
  design.
- `BoardCleanupButton` in the desktop and tablet headers before the sort
  control, on the kanban page only.
- `MobileCleanupEntry` in the phone listing menu's display options.
- Removal of `RepositoryCleanupButton` from `RepositoryPreview`; the Edit and
  Delete buttons return to their state before Task 03.
- One new string (the choose-a-repository hint) in English, pseudo, and the
  seven translated catalogs; no em dash.
- E2E moved to `kanban/repository-cleanup.spec.ts` and
  `kanban/mobile-repository-cleanup.spec.ts`; public docs updated.

## Out of scope

- Backend, prompt, and dialog content changes.

## Acceptance

- Matches UI-01 in the plan: with a local repository selected in the filter,
  the broom opens the confirmation for that repository, and confirming opens
  the cleanup task.
- With several local repositories and none selected, the action is disabled
  with the hint on desktop and phone; the phone entry is at least 44px tall.
- Repository settings rows show no broom.

## Verification

```bash
cd apps/web && pnpm exec vitest run lib/kanban components/kanban components/settings
```

```bash
cd apps/web && pnpm run typecheck && pnpm run i18n:check
```

```bash
cd apps && pnpm --filter @kandev/web lint
```

```bash
cd apps/web && pnpm e2e:run -- tests/kanban/repository-cleanup.spec.ts
```

```bash
cd apps/web && pnpm e2e:run -- --project mobile-chrome tests/kanban/mobile-repository-cleanup.spec.ts
```

## Likely files

- `apps/web/lib/kanban/cleanup-target.ts` (new) and its test
- `apps/web/components/kanban/board-cleanup-action.tsx` (new)
- `apps/web/components/kanban/kanban-header.tsx`
- `apps/web/components/kanban/kanban-header-mobile.tsx`,
  `mobile-menu-sheet.tsx`, `mobile-display-options.tsx`
- `apps/web/components/settings/repository-card-preview.tsx`,
  `repository-cleanup-button.tsx` (removed)
- `apps/web/src/locales/*/workspaces.json`
- `docs/public/tasks-and-workflows.md`

## Risks

- The tablet header is crowded; the broom is one `icon-lg` button like its
  neighbours.

## ASCII UI preview

See [UI-01 in the plan](plan.md#ascii-ui-preview).
