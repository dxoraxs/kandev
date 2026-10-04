---
id: "03-phone-composition"
title: "Phone composition"
status: done
wave: 2
depends_on:
  - "01-description-document-and-notice"
plan: "plan.md"
requirements:
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-004
acceptance_criteria:
  - AC-TASKS-TASK-DESCRIPTION-VIEW-004.1
  - AC-TASKS-TASK-DESCRIPTION-VIEW-004.2
system_design:
  - ../../specs/tasks/system-design/task-description-view.md
---

# Task 03: Phone composition

## Summary

While the task is sessionless, the first phone nav item reads "Description"
and shows the description document without the session picker or composer;
the corner notice opens a bottom drawer with 44 px actions.

## In scope

- `sessionless` flag in `buildMobileNavItems`; label and icon swap only, the
  internal panel id stays `chat`.
- `MobileChatPanelContent` branch for the sessionless state.
- Phone variant of `UnassignedCornerNotice`: compact 44 px control and a
  `Drawer` with stacked full-width actions.

## Acceptance

- `buildMobileNavItems` unit test: label is Description when sessionless and
  Chat otherwise.
- No session picker or composer is mounted while sessionless.

## Verification

```bash
cd apps/web && ./node_modules/.bin/vitest run components/task/mobile
cd apps/web && pnpm run typecheck
cd apps/web && pnpm run lint
```

## UI preview

UI-03 in the [plan](plan.md#ui-03-phone-sessionless-task).

## Files likely touched

- `apps/web/components/task/mobile/session-mobile-bottom-nav.tsx` and test
- `apps/web/components/task/mobile/session-mobile-layout.tsx`
- `apps/web/components/task/sessionless-task-view.tsx`

## Dependencies

01.

## Risks

- The tablet layout renders `TaskChatPanel`; it relies on the 01 fallback and
  needs no change here.

## Results

- `SessionMobileBottomNav` takes `sessionless`; the first item reads
  Description with a document icon while true (RED-first `it.each` test).
- `MobileChatPanelContent` renders `SessionlessTaskView` with
  `presentation="mobile"` and no picker, queue status, or `TaskChatPanel`.
- `UnassignedNoticeDrawer` (`sessionless-notice-drawer.tsx`): a 44 px corner
  control floated top-right that opens a bottom drawer with the explanation
  and full-width 44 px "Start agent" and settings actions.
- `vitest run components/task/mobile components/task/sessionless-task-view-model.test.ts`:
  20 files, 156 passed; `tsc --noEmit` and eslint clean.
