---
created: 2026-10-04
status: implemented
requirements:
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-001
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-002
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-003
system_design:
  - ../../specs/tasks/system-design/task-open-without-agent.md
legacy_specs: []
---

# Implementation Plan: Task Open Without Agent

## Overview

Opening a sessionless task whose agent profile does not resolve currently
fails with a red "could not start a session" banner, and the page shows an
empty chat. This plan makes the passive open return a typed
`no_agent_profile` outcome, renders the task description for every
sessionless task, and replaces the error with a neutral notice that offers
"Start agent" and the workspace settings link.

Requirements: [task-open-without-agent](../../specs/tasks/requirements/task-open-without-agent.md).
Design: [task-open-without-agent](../../specs/tasks/system-design/task-open-without-agent.md).

## Scope

- Backend: short-circuit in `Service.EnsureSession` for passive opens with no
  resolved profile.
- Web: `unassigned` ensure status, `SessionlessTaskView`, mounts in
  `TaskChatPanel` and the board preview, removal of the dead
  agent-profile-missing branch in `describeEnsureError`.
- E2E on desktop and mobile projects.

Excluded: a "waiting" step kind, parsing description fields, profile
resolution changes (see the requirement's Out of scope).

## Work orders

| # | Work order | Wave | Depends on |
| --- | --- | --- | --- |
| 01 | [Ensure unassigned outcome](task-01-ensure-unassigned-outcome.md) | 1 | none |
| 02 | [Sessionless task view](task-02-sessionless-task-view.md) | 2 | 01 |
| 03 | [E2E desktop and mobile](task-03-e2e-desktop-and-mobile.md) | 3 | 02 |

01 and 02 touch disjoint trees, but 02 consumes the response literal defined
by 01, so they run in order.

## ASCII UI preview

### UI-01: Task page, sessionless task, no agent assigned (desktop)

Entry: open a task with no session in a workspace where no agent profile
resolves. Before this change the page shows a red banner above an empty chat.

Before (current, from the reported screenshot):

```text
+--------------------------------------------------------------+
| dxoraxs/dmhive > [Claude] 06a ...                            |
| +----------------------------------------------------------+ |
| | (!) Could not start a session              [destructive] | |
| |     Backend rejected the session request.                | |
| |     > Technical details                                  | |
| |     [ > Retry ]                                          | |
| +----------------------------------------------------------+ |
| [Agent] [+]                                                  |
|                                                              |
|          No messages yet. Start the conversation!            |
|                                                              |
+--------------------------------------------------------------+
```

After:

```text
+--------------------------------------------------------------+
| dxoraxs/dmhive > [Claude] 06a ...                            |
| [Agent] [+]                                                  |
| +----------------------------------------------------------+ | <- scrolls
| | > File: docs/plans/06a-instagram-data-deletion.md        | |
| | # 06a. Meta deletion covers the new Instagram channel    | |
| | **Next responsible:** owner: deploy migration ...        | |
| | ... (task description, rendered Markdown) ...            | |
| +----------------------------------------------------------+ |
| +----------------------------------------------------------+ |
| | (i) No agent profile configured             [neutral]    | |
| |     This task has no agent profile, and the workspace,   | |
| |     workflow, and workflow step have no default set. ... | |
| |     [ Start agent ]   Open workspace settings            | |
| +----------------------------------------------------------+ |
+--------------------------------------------------------------+
```

Required: no destructive banner, no Retry, no raw error text (AC-003.1);
description above the notice (AC-002.1); "Start agent" is the primary action
(AC-003.2); settings link present when the workspace is known (AC-003.3).
Illustrative: spacing, exact copy wording (existing locale keys are reused).

### UI-02: Same state on a phone

```text
+------------------------------+
| < 06a Meta deletion ...      |
|------------------------------|
| > File: docs/plans/06a-...   | <- scrolls
| # 06a. Meta deletion covers  |
| the new Instagram channel    |
| **Next responsible:** owner: |
| deploy migration ...         |
| ...                          |
| +--------------------------+ |
| | (i) No agent profile     | |
| |     configured           | |
| |  This task has no agent  | |
| |  profile ...             | |
| | [      Start agent     ] | | <- full width, >= 44 px
| | [ Open workspace settings]| | <- full width, >= 44 px
| +--------------------------+ |
|------------------------------|
| [Chat] [Plan] [Changes] ...  | <- existing bottom nav, fixed
+------------------------------+
```

Required on phone: actions stacked full-width with 44 px touch targets, no
horizontal scroll (AC-003.5). The notice lives in the conversation area, not
in the page-level feedback slot.

### UI-03: Board preview, sessionless task

The preview's sessionless slot shows the same `SessionlessTaskView`
(description, then the notice or the "Preparing workspace…" line) instead of
the centered empty-state text. The preview's error branch is unchanged.

## Verification strategy

- 01: Go unit tests in `internal/orchestrator` and the WS handler package,
  plus `golangci-lint`.
- 02: Vitest for the hook and the view's pure decision helper, typecheck,
  lint, i18n checks.
- 03: Playwright on `chromium` and `mobile-chrome` projects; this is the
  end-to-end evidence for REQ-002 and REQ-003.

## Risks

- `TaskChatPanel` renders in several hosts (dockview, mobile, embedded). The
  sessionless view must only replace the transcript when no session resolves
  and a task id is known, or an embedded host without a task id would render
  an empty description.
- A sessionless task in the `preparing` state now shows its description
  briefly before the transcript appears; the transcript's own synthetic
  description message takes over, so the content does not jump in meaning.
- HTTP `sessions/ensure` keeps erroring on a missing profile by design
  (AC-001.3); callers that rely on it are not passive opens.
