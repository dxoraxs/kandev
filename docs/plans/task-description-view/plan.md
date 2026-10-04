---
created: 2026-10-04
status: implemented
requirements:
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-001
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-002
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-003
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-004
system_design:
  - ../../specs/tasks/system-design/task-description-view.md
legacy_specs: []
---

# Implementation Plan: Task Description View

## Overview

A sessionless task opens on a "Description" tab that renders its description
as a Markdown document. A compact unassigned notice sits in the top-right
corner. The "Agent" tab appears with the first session and the Description
tab stays open.

Requirements: [task-description-view](../../specs/tasks/requirements/task-description-view.md).
Design: [task-description-view](../../specs/tasks/system-design/task-description-view.md).
Builds on the implemented [task-open-without-agent](../task-open-without-agent/plan.md) package.

## Scope

- Description document with `markdown-body` styling and the corner notice.
- `task-description` Dockview panel, sessionless layout target, and the
  description-to-Agent tab transition.
- Phone: Description nav item while sessionless, drawer notice.
- E2E on desktop and phone; removal of the superseded criteria from the
  task-open-without-agent requirement and promotion of this package.

Excluded: backend changes, description editing, status filtering of the
description (owned by the external synchronizer).

## Work orders

| # | Work order | Wave | Depends on |
| --- | --- | --- | --- |
| 01 | [Description document and corner notice](task-01-description-document-and-notice.md) | 1 | none |
| 02 | [Description panel and Agent tab transition](task-02-description-panel-and-agent-tab.md) | 2 | 01 |
| 03 | [Phone composition](task-03-phone-composition.md) | 2 | 01 |
| 04 | [E2E and spec reconciliation](task-04-e2e-and-spec-reconciliation.md) | 3 | 02, 03 |

02 and 03 touch disjoint files (layout manager and Dockview versus the mobile
layout) and can run in parallel after 01.

## ASCII UI preview

### UI-01: Desktop, sessionless task, no agent assigned

Entry: open a plan-board task with no session in a workspace without a default
agent profile.

Before (current, from the rendered page):

```text
+-------------------------------------------------------------------+
| dxoraxs/dmhive > [Claude] 06a ...               (o) Waiting ext.  |
| [Agent] [+]                                                        |
| File: docs/plans/06a-...                       <- no quote styling |
| 06a. Meta deletion covers ...                  <- no heading      |
| Now                                            <- no heading      |
| live check on your test connection ...         <- no bullet       |
| +---------------------------------------------------------------+ |
| | (i) No agent profile configured                               | |
| |     This task has no agent profile, and the workspace, ...    | |
| |     [ Start agent ]  Open workspace settings                  | |
| +---------------------------------------------------------------+ |
| [ Continue working on the task...                          ( ^ ) ] |
+-------------------------------------------------------------------+
```

After:

```text
+-------------------------------------------------------------------+
| dxoraxs/dmhive > [Claude] 06a ...               (o) Waiting ext.  |
| [Description] [+]                                                  |
| | File: docs/plans/06a-...        +-------------------------------+ | <- scrolls
| | Executor: Claude                | (i) No agent profile    (?)   | |
|                                   |     [ Start agent ]           | |
| # 06a. Meta deletion covers       +-------------------------------+ |
|   the new Instagram channel                                        |
| **Status:** waiting for Meta since 01.10                           |
| ## Now                                                             |
| **Waiting for:** Meta App Review decision ...                      |
| **Left:**                                                          |
|  - live check on your test connection ...                          |
|  - attach the result to App Review                                 |
| **Next:** Meta answers; then the owner runs the live check.        |
+-------------------------------------------------------------------+
```

Required: active Description tab, no Agent tab, no composer (AC-001.1);
headings, quotes, and bullets render as elements (AC-001.1); the notice sits
top-right, text wraps around it and is never covered, Start agent is visible
directly (AC-003.1, AC-003.2); (?) discloses the explanation and the settings
link (AC-003.2). Illustrative: exact card width and copy wrapping.

### UI-02: Desktop, after "Start agent"

```text
+-------------------------------------------------------------------+
| [Description] [Agent *] [+]                       * = active tab  |
| (agent conversation: task message, agent replies, tool calls)      |
| [ Continue working on the task...                          ( ^ ) ] |
+-------------------------------------------------------------------+
```

Required: the Agent tab appears and becomes active without a reload; the
Description tab stays (AC-002.1, AC-002.2).

### UI-03: Phone, sessionless task

```text
+------------------------------+
| < 06a Meta deletion ...    = |
|------------------------------|
| | File: docs/plans/06a  [(i)]| <- corner control, >= 44 px
| # 06a. Meta deletion covers  | <- scrolls
| ## Now                       |
| **Left:**                    |
|  - live check ...            |
|------------------------------|
| [Description] [Plan] [Files] | <- first item reads Description
+------------------------------+

Tapping (i) opens a bottom drawer:
+------------------------------+
| No agent profile configured  |
| This task has no agent ...   |
| [        Start agent       ] | <- full width, >= 44 px
| [  Open workspace settings ] | <- full width, >= 44 px
+------------------------------+
```

Required: no session picker, no composer, first nav item reads Description
while sessionless and Chat after the first session (AC-004.1); drawer
actions are full-width 44 px (AC-004.2).

## Verification strategy

- 01: Vitest for the view model, typecheck, lint, i18n checks.
- 02: Vitest for `runAutoSessionTabEffect`, the add-panel menu, and the placeholder title; typecheck, lint.
- 03: Vitest for `buildMobileNavItems`, typecheck, lint.
- 04: Playwright on `chromium` and `mobile-chrome`; this is the end-to-end
  evidence for every requirement in the package.

## Risks

- Saved Dockview layouts hold a `chat` panel for sessionless tasks; it stays
  the sessionless slot, renders the description, and is titled Description.
- The session-tab sync code has several entry points (restore, live add,
  handoff). The work order must cover the live add path, not only the pure
  layout function.
- The float layout depends on container width; the narrow fallback must keep
  the notice from squeezing the document into a thin column.
