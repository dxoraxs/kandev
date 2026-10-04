---
id: "03-e2e-desktop-and-mobile"
title: "E2E desktop and mobile"
status: done
wave: 3
depends_on:
  - "02-sessionless-task-view"
plan: "plan.md"
requirements:
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-001
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-002
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-003
acceptance_criteria:
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.1
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.4
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-002.1
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.1
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.2
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.4
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.5
system_design:
  - ../../specs/tasks/system-design/task-open-without-agent.md
---

# Task 03: E2E desktop and mobile

## Summary

Playwright evidence that a task with no resolvable agent profile opens with
its description and a neutral notice, and that "Start agent" leads to a live
session, on desktop and phone viewports.

## In scope

- `e2e/tests/task/task-open-without-agent.spec.ts` (chromium): seed a
  workspace without a default profile and a workflow step without a profile;
  create a task with a Markdown description; open it; assert the description
  text, the notice, no `ensure-session-error-banner`, no Retry; click
  "Start agent", pick the mock profile, start; assert the transcript replaces
  the view without reload.
- Second case: set the workspace default profile, reopen the task, assert a
  session is ensured (AC-001.4).
- `e2e/tests/task/mobile-task-open-without-agent.spec.ts` (mobile-chrome,
  390x844): same open assertions, action buttons at least 44 px tall, and
  `document.documentElement.scrollWidth <= clientWidth`.

## Out of scope

- Board preview E2E beyond a single open assertion if the preview helper
  makes it cheap; unit coverage from task 02 is sufficient otherwise.

## ASCII UI preview

See [plan.md UI-01 and UI-02](plan.md#ascii-ui-preview).

## Acceptance

- Both specs pass on their projects in two consecutive runs with no fixed
  sleeps.
- The desktop spec fails if the destructive banner is reintroduced.

## Verification

```bash
cd apps/web && pnpm e2e:run --host --shards 1 --project chromium tests/task/task-open-without-agent.spec.ts
cd apps/web && pnpm e2e:run --host --shards 1 --project mobile-chrome tests/task/mobile-task-open-without-agent.spec.ts
cd apps/web && pnpm run typecheck
```

## Files likely touched

- `apps/web/e2e/tests/task/task-open-without-agent.spec.ts` (new)
- `apps/web/e2e/tests/task/mobile-task-open-without-agent.spec.ts` (new)
- `apps/web/e2e/helpers/api-client.ts` (only if a workspace-default setter is
  missing)

## Dependencies

Task 02.

## Risks

- E2E seed data may set a workspace default profile; the spec must create its
  own workspace or clear the default and restore it.

## Results

- `pnpm e2e:run --host --shards 1 --project chromium tests/task/task-open-without-agent.spec.ts`:
  2 passed, in two consecutive runs.
- `pnpm e2e:run --host --shards 1 --project mobile-chrome tests/task/mobile-task-open-without-agent.spec.ts`:
  1 passed, in two consecutive runs.
- `pnpm run typecheck`: clean.
- The runs exposed a latent race in `useEnsureTaskSession`: a same-task
  dependency rerender dropped the in-flight ensure result and left the page on
  "Preparing workspace". Fixed under work order 02 with a latch-key ownership
  check and a red-first unit test.
- Captured desktop and phone screenshots match previews UI-01 and UI-02.
- Follow-up: on a long description the view grew to its content height inside
  a non-flex host, so nothing scrolled and the notice was unreachable. Added a
  red-first desktop scenario ("scrolls a long description to reach the
  notice"); the view root now takes `h-full`. Desktop 3 passed, mobile 1 passed.
