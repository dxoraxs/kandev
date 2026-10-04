---
created: 2026-10-05
status: draft
requirements:
  - REQ-TASKS-PLAN-ADAPT-001
  - REQ-TASKS-PLAN-ADAPT-002
  - REQ-TASKS-PLAN-ADAPT-003
  - REQ-TASKS-PLAN-ADAPT-004
system_design:
  - ../../specs/tasks/system-design/plan-file-adaptation.md
legacy_specs: []
---

# Implementation Plan: Plan File Adaptation

## Overview

Make unadapted plan files visible, then let the owner hand their adaptation to
an agent with one action. Build bottom-up: detection in `internal/planfiles`
first (useful on its own: the status line explains an empty board), then the
repository maintenance task launcher with the `plan_adaptation` kind and its
prompt, then the web entry points and E2E. The feature stays behind the
existing `features.planFiles` flag.

The launcher built here is reused by
[Repository cleanup](../repository-cleanup/plan.md), which depends on Task 02.

Related decision:
[ADR 2026-10-05 repository maintenance agent tasks](../../decisions/2026-10-05-repository-maintenance-agent-tasks.md).

## Scope

### In scope

- `PassCounts.Unadapted`, `Service.UnadaptedCounts`,
  `GET /api/v1/plan-files/unadapted`, `Service.EnsureBoard`.
- `POST /api/v1/repositories/:id/maintenance-tasks` with kind
  `plan_adaptation`, active-task guard, default resolution, metrics.
- `config/prompts/plan-file-adaptation.md` and its rendering test.
- Settings rows, status count, post-add offer dialog, API clients, i18n,
  desktop and phone E2E.

### Out of scope

- The `repository_cleanup` kind (separate plan).
- Deterministic conversion; other plan formats; remote repositories.
- Public docs beyond a short section in `docs/public/plan-files.md`, which
  Task 03 adds.

## Work orders

| Task | Wave | Depends on | Result |
| --- | --- | --- | --- |
| [01 Unadapted detection](task-01-unadapted-detection.md) | 1 | none | Count, endpoint, `EnsureBoard`. |
| [02 Maintenance launcher](task-02-maintenance-launcher.md) | 2 | 01 | Endpoint, guard, `plan_adaptation` kind, prompt. |
| [03 Web entry points](task-03-web-entry-points.md) | 3 | 01, 02 | Rows, offer dialog, E2E, docs. |

## Risks

- **Workflow start-step side effects.** The first visible workflow's start step
  can carry on-enter actions or an auto-start prompt that competes with the
  built-in prompt. Task 02 tests the launch against the default Kanban
  template and documents the choice in the endpoint test.
- **Shared main checkout.** The agent edits the checkout the owner works in.
  The offer copy states this; the guard prevents two maintenance agents.
- **Prompt drift.** Rules live in a Markdown template; the rendering test
  pins every acceptance-criterion rule.

## Verification strategy

- Go unit tests for detection, `EnsureBoard`, launcher resolution and guard,
  prompt rendering.
- Vitest for API clients and components.
- Playwright desktop and `mobile-*` specs with the mock agent: offer dialog
  after adding a repository with unadapted plans, Adapt with agent from
  settings, navigation to the created task, and the existing-task path.

## ASCII UI preview

### UI-01: Plan files section, unadapted rows (settings, sync enabled or not)

Desktop:

```text
+-- Plan files ---------------------------------------------- [on] --+
| Board      [ Plans          v ]                                     |
| ...status mapping, directories (unchanged)...                       |
| Last sync 13:42  ok   created 21  updated 2  not adapted 16         |
|                                                                     |
| Plan files not on the board                                         |
| +-----------------------------------------------------------------+ |
| | beaver-blocks       16 files without board status               | |
| |                                       [ Adapt with agent ]      | |
| +-----------------------------------------------------------------+ |
|                                          [Sync now]  [Save]         |
+---------------------------------------------------------------------+
```

When no config exists, the row adds a helper line: "A Plans board will be
created and sync turned on."

Phone (below 768px): rows stack.

```text
+-- Plan files ----------- [on] --+
| ...                              |
| Plan files not on the board      |
| beaver-blocks                    |
| 16 files without board status    |
| [      Adapt with agent       ]  |  <- full width, 44px
+----------------------------------+
```

### UI-02: Offer after adding a repository

Desktop dialog; phone bottom sheet (shared `Dialog` below 640px).

```text
+-- Plans found in beaver-blocks -----------------------------+
| 16 plan files are not in the board format, so they do not   |
| appear on the board. An agent can read the repository       |
| history, set each plan's status, and commit only the plan   |
| files. It works in this repository's checkout.              |
| A Plans board will be created and sync turned on.  (*)      |
|                                  [Not now] [Adapt with agent]|
+-------------------------------------------------------------+
(*) only when plan sync is not configured
```

Phone: the same content in a bottom sheet; actions stacked full width, primary
first, 44px targets.

Structural requirements: one row per repository with name, count, and action;
the action is the row's primary control; the offer has exactly two actions.
Spacing and wording are illustrative; copy is localized.
