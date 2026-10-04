---
id: "02-description-panel-and-agent-tab"
title: "Description panel and Agent tab transition"
status: done
wave: 2
depends_on:
  - "01-description-document-and-notice"
plan: "plan.md"
requirements:
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-001
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-002
acceptance_criteria:
  - AC-TASKS-TASK-DESCRIPTION-VIEW-001.1
  - AC-TASKS-TASK-DESCRIPTION-VIEW-001.2
  - AC-TASKS-TASK-DESCRIPTION-VIEW-002.1
  - AC-TASKS-TASK-DESCRIPTION-VIEW-002.2
system_design:
  - ../../specs/tasks/system-design/task-description-view.md
---

# Task 02: Description panel and Agent tab transition

## Summary

Register a `task-description` Dockview panel. A sessionless task keeps the
generic chat placeholder, titled "Description" and rendering the document.
When the first session replaces the placeholder, a Description tab is kept
ahead of the new, active Agent tab.

## In scope

- Registry, renderable list, structural list, known ids, `PANEL_RENDERERS`,
  `dockview-shared.tsx`, and the add-panel menu.
- Placeholder title `task:panelDescription` while sessionless.
- Live path in `dockview-session-tabs.ts` (`runAutoSessionTabEffect`) for a
  session that arrives on an open page.

Out of scope: phone layout (03).

## Acceptance

- Effect unit tests: the first session of a sessionless task leaves
  `[task-description, session, ...]` with the session active; a task with a
  session at open gets no Description tab.
- The live add path is covered by a test at its own entry point, not only by
  the pure layout function.

## Verification

```bash
cd apps/web && ./node_modules/.bin/vitest run lib/state/layout-manager components/task/dockview-session-tabs.test.ts components/task/dockview-add-panel-items.test.tsx
cd apps/web && pnpm run typecheck
cd apps/web && pnpm run lint
```

## UI preview

UI-01 (tab strip) and UI-02 in the [plan](plan.md#ui-02-desktop-after-start-agent).

## Files likely touched

- `apps/web/lib/state/layout-manager/constants.ts`,
  `renderable-components.ts`, `apps/web/lib/state/dockview-extra-panel-actions.ts`
- `apps/web/components/task/dockview-panel-content.tsx`,
  `dockview-shared.tsx`, `dockview-add-panel-items.tsx`,
  `dockview-session-tabs.ts`, `task-description-panel.tsx` (new)

## Dependencies

01 (the panel renders `TaskDescriptionDocument`).

## Risks

- Restore, handoff, and hidden-session paths also call materialization;
  run the whole layout-manager and dockview session test sets.

## Results

- Approach changed from the planned `targetPanels(null)` materialization to
  keeping the chat placeholder: restore, hand-off, and safety-net paths all
  depend on it. The system design was updated to match.
- `cd apps/web && ./node_modules/.bin/vitest run lib/state components/task`:
  693 files, 6609 passed, 4 skipped. New tests: "keeps a Description tab when
  the first session replaces a sessionless placeholder" (RED first), "adds no
  Description tab when the task already had a session at open", the add-panel
  Description row (RED first), and the sessionless placeholder title (RED first).
- `tsc --noEmit` clean; eslint clean on all touched files.
