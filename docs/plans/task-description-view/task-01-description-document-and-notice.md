---
id: "01-description-document-and-notice"
title: "Description document and corner notice"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-001
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-003
acceptance_criteria:
  - AC-TASKS-TASK-DESCRIPTION-VIEW-001.3
  - AC-TASKS-TASK-DESCRIPTION-VIEW-001.4
  - AC-TASKS-TASK-DESCRIPTION-VIEW-003.1
  - AC-TASKS-TASK-DESCRIPTION-VIEW-003.2
  - AC-TASKS-TASK-DESCRIPTION-VIEW-003.3
system_design:
  - ../../specs/tasks/system-design/task-description-view.md
---

# Task 01: Description document and corner notice

## Summary

Replace `SessionlessTaskView` with `TaskDescriptionDocument` (description
rendered inside `markdown-body`) and `UnassignedCornerNotice` (compact card
floated top-right, info disclosure for the explanation and settings link).
Mount them in the board preview and in the `TaskChatPanel` fallback.

## In scope

- Rewrite `components/task/sessionless-task-view.tsx` in place (the file
  name is kept); keep the context and hook exports.
- `markdown-body` wrapper; document is its own scroll container.
- Corner notice: title, small "Start agent" button, info control disclosing
  the detail text and the workspace settings link; `preparing` spinner in the
  same corner; narrow-container fallback to full width above the document.
- `TaskChatPanel` fallback hides the composer while it shows the document.
- New keys `task:panelDescription`, `task:unassignedNoticeDetails` in all
  locales (zh-hant via the converter, pseudo regenerated); guard list entry.

Out of scope: Dockview panel registration (02), phone drawer (03).

## Acceptance

- The board preview of a sessionless task shows headings and bullets as
  elements and the corner notice without the old full-width alert.
- `sessionless-task-view-model.test.ts` stays green; no Retry or destructive
  styling for the unassigned state.

## Verification

```bash
cd apps/web && ./node_modules/.bin/vitest run components/task/sessionless-task-view-model.test.ts
cd apps/web && pnpm run typecheck
cd apps/web && pnpm run lint
cd apps/web && pnpm run i18n:check
```

## UI preview

UI-01 in the [plan](plan.md#ui-01-desktop-sessionless-task-no-agent-assigned)
(document and corner card region).

## Files likely touched

- `apps/web/components/task/sessionless-task-view.tsx`
- `apps/web/components/task/task-chat-panel.tsx`
- `apps/web/components/task/preview-session-tabs.tsx`
- `apps/web/src/locales/*/task.json`, `apps/web/eslint.i18n.options.mjs`

## Dependencies

None.

## Risks

- Floats inside a flex scroll container: the document wrapper must be a
  block container so text wraps around the card.

## Results

- `TaskDescriptionDocument` renders the description inside `markdown-body`
  as its own scroll container; `UnassignedCornerNotice` is a compact card
  floated top-right (full width above the document in narrow containers) with
  "Start agent" and a (?) popover holding the detail and settings link.
- `TaskChatPanel` hides the composer while it shows the sessionless view.
- `vitest run components/task/sessionless-task-view-model.test.ts` green;
  `tsc --noEmit`, eslint, and `pnpm run i18n:check` clean.
