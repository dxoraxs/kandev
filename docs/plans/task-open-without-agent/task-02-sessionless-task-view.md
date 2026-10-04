---
id: "02-sessionless-task-view"
title: "Sessionless task view"
status: done
wave: 2
depends_on:
  - "01-ensure-unassigned-outcome"
plan: "plan.md"
requirements:
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-002
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-003
acceptance_criteria:
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-002.1
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-002.2
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-002.3
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.1
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.2
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.3
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.4
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.5
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.6
system_design:
  - ../../specs/tasks/system-design/task-open-without-agent.md
---

# Task 02: Sessionless task view

## Summary

The web client treats `no_agent_profile` as an `unassigned` ensure status and
renders a sessionless task as its description plus, when unassigned, a
neutral notice with "Start agent" and the workspace settings link.

## In scope

- `EnsureSessionResponse.source` literal in
  `apps/web/lib/services/session-launch-service.ts`.
- `unassigned` status in `useEnsureTaskSession`; no forced session reload for
  that outcome; latch unchanged.
- `SessionlessTaskContext` provided by `TaskPageContent`.
- `SessionlessTaskView` component with a pure decision helper
  (description vs placeholder, which status line).
- Mount in `TaskChatPanel` when `resolvedSessionId` is null and a task id is
  known; mount in `PreviewNoSessionsState` for `preparing`, `unassigned`,
  `idle`.
- New copy (empty-description placeholder, "Start agent" if no existing key
  fits) in all seven locales; reuse `task:noAgentProfileConfigured`,
  `task:noAgentProfileConfiguredDetail`, `task:openWorkspaceSettings`.
- Append the new component path to `i18nGuardFiles`.

## Out of scope

- Backend (task 01), E2E (task 03).
- Any change to the error banner for non-unassigned failures.

## ASCII UI preview

See [plan.md UI-01, UI-02, UI-03](plan.md#ascii-ui-preview). Required: the
notice uses the neutral `Alert` variant, sits under the description inside
the conversation area, and on phones stacks full-width 44 px actions.

## Acceptance

- Hook test: a `no_agent_profile` response sets `status === "unassigned"`,
  does not call the forced session reload, and does not re-fire on rerender.
- Helper test: the view decision returns description vs placeholder and the
  correct status line for `idle`, `preparing`, `unassigned`, `error`.
- Typecheck, lint, and i18n checks pass with no new hardcoded copy.

## Verification

```bash
cd apps/web && pnpm exec vitest run hooks/domains/session/use-ensure-task-session.test.ts components/task/sessionless-task-view-model.test.ts components/task/ensure-session-error components/task/preview-session-tabs
cd apps/web && pnpm run typecheck
cd apps && pnpm --filter @kandev/web lint
cd apps/web && pnpm run i18n:check && pnpm run i18n:ratchet
```

## Files likely touched

- `apps/web/lib/services/session-launch-service.ts`
- `apps/web/hooks/domains/session/use-ensure-task-session.ts`
- `apps/web/hooks/domains/session/use-ensure-task-session.test.ts`
- `apps/web/components/task/sessionless-task-view.tsx` (new)
- `apps/web/components/task/sessionless-task-view-model.ts` (new)
- `apps/web/components/task/sessionless-task-view-model.test.ts` (new)
- `apps/web/components/task/task-page-content.tsx`
- `apps/web/components/task/task-chat-panel.tsx`
- `apps/web/components/task/preview-session-tabs.tsx`
- `apps/web/src/locales/*/task.json`
- `apps/web/eslint.i18n.options.mjs`

## Dependencies

Task 01 (response literal).

## Risks

- `TaskChatPanel` is also used embedded; guard on a known task id.
- Kanban preview passes `isVisible=false`; the view must not depend on read
  cursors or scroll geometry.

## Results

- `useEnsureTaskSession` reports `unassigned` for `no_agent_profile` and skips
  the forced session reload; covered in `use-ensure-task-session.test.ts`.
- `SessionlessTaskView` (`sessionless-task-view.tsx`) with the pure
  `resolveSessionlessTaskView` helper (`sessionless-task-view-model.ts`, tested).
  `TaskChatPanel` mounts it through `useSessionlessTaskView`; the board preview
  renders it in `PreviewNoSessionsState` instead of the centered "no agents"
  text and the bare preparing spinner.
- Deviation: `describeEnsureError` was not changed. Its
  `agent_profile_id is required` branch is still reachable from other backend
  paths, so it is not dead code; the design was corrected.
- New copy: `task:sessionlessNoDescription` in all locales (zh-hk/zh-tw via
  the zh-hant converter, unrelated converter churn reverted).
- Vitest (4 files, 87 tests): pass. `tsc --noEmit`: pass.
  `pnpm --filter @kandev/web lint`: pass, 0 warnings. All `i18n:check`
  scripts and `i18n:ratchet`: pass. Commands were run with the local Node 22;
  the repo's engines field asks for Node 24.
