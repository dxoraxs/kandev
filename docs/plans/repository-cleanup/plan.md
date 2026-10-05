---
created: 2026-10-05
status: implemented
requirements:
  - REQ-WORKSPACES-REPO-CLEANUP-001
  - REQ-WORKSPACES-REPO-CLEANUP-002
  - REQ-WORKSPACES-REPO-CLEANUP-003
system_design:
  - ../../specs/workspaces/system-design/repository-cleanup.md
legacy_specs: []
---

# Implementation Plan: Repository Cleanup

## Overview

Add a broom action to the board's top bar, aimed at the repository the board
is filtered to (Task 04 moved it there from the repository settings rows). It starts
an agent task on the repository's main checkout that merges every branch and
worktree into the default branch, asks the owner to confirm a summary, then
deletes merged branches locally and on the remote, removes worktrees, and
pushes. Behavior lives in a built-in prompt; the backend adds a launcher kind,
a protected list, and a release flag.

Owner decisions recorded for this package (2026-10-05): the agent resolves
conflicts itself and keeps a branch when unsure or tests fail; one confirmation
before irreversible steps; uncommitted worktree changes are committed as WIP
and merged.

Related decision:
[ADR 2026-10-05 repository maintenance agent tasks](../../decisions/2026-10-05-repository-maintenance-agent-tasks.md).

## Scope

### In scope

- Runtime flag `features.repositoryCleanup` (off in every shipped profile).
- Launcher kind `repository_cleanup`, protected list, cleanup prompt and its
  rendering test.
- Broom button, confirmation dialog, navigation, i18n, desktop and phone E2E,
  public docs.

### Out of scope

- The launcher itself (built by
  [Plan file adaptation Task 02](../plan-file-adaptation/task-02-maintenance-launcher.md)).
- Backend git automation or a backend-enforced confirmation gate.
- Remote repositories; scheduled cleanup.

## Work orders

| Task | Wave | Depends on | Result |
| --- | --- | --- | --- |
| [01 Runtime flag](task-01-runtime-flag.md) | 1 | none | `features.repositoryCleanup` end to end, off. |
| [02 Cleanup kind and prompt](task-02-cleanup-kind-and-prompt.md) | 2 | 01, plan-file-adaptation 02 | Kind, protected list, prompt. |
| [03 Broom button](task-03-broom-button.md) | 3 | 02 | Button, dialog, E2E, docs. |
| [04 Board placement](task-04-board-placement.md) | 4 | 03 | Broom in the board top bar and phone listing menu; removed from settings rows. |

## Risks

- **Destructive operations by an agent.** Mitigated by the prompt's forbidden
  operations, merged-only deletion checks, the protected list, and the
  confirmation step; residual risk is accepted in the ADR.
- **WIP commits in someone's worktree.** The owner chose this; protected
  worktrees of live Kandev tasks are excluded.
- **Branch protection on the remote.** A rejected push is reported; no force.

## Verification strategy

- Go tests: flag registry completeness, kind availability, protected list,
  prompt rendering (every phase, forbidden operations, protected lists).
- Vitest for the button and dialog.
- Playwright desktop and `mobile-*` specs with the mock agent: button visible
  for a local repository, dialog content, task created and opened, existing
  task reused, flag off hides the button.

## ASCII UI preview

### UI-01: Broom in the board top bar

Desktop and tablet (kanban page; tooltip "Clean up repository"). The broom
targets the repository selected in the filter menu (the sliders button).

```text
+-- Home ---------------------[ search ]---------------------------------------+
|        [New plan] [kanban|graph|threads|list] [broom] [By priority v] [filter] |
+------------------------------------------------------------------------------+
```

No repository selected and more than one local repository: the broom is
disabled; tooltip "Choose a repository in the filter to clean it up".

Phone (below 768px): an entry in the listing menu (opened from the page
title), after New plan; 44px, full width.

```text
+-- Display options ---------------+
| [file+] New plan                 |
| [broom] Clean up repository      |
|   Choose a repository in the     |
|   filter to clean it up.         |  (only when disabled)
| > Filters                        |
| > Sort                           |
+----------------------------------+
```

### UI-02: Confirmation

Desktop dialog (shared `Dialog`); phone bottom sheet (shared `Drawer` when `isMobile`).

```text
+-- Clean up city_companion? ---------------------------------+
| An agent will, in this repository's checkout:               |
|  1. Commit uncommitted worktree changes as WIP.             |
|  2. Merge every branch and worktree into main, resolving    |
|     conflicts; branches it is unsure about are kept.        |
|  3. Show you a summary and wait for your confirmation.      |
|  4. Push main, delete merged branches locally and on the    |
|     remote, and remove worktrees.                           |
| Branches of active Kandev tasks are not touched.            |
|                                  [Cancel] [Start cleanup]   |
+-------------------------------------------------------------+
```

Phone: the same content in a bottom sheet; actions stacked full width,
primary first, 44px targets.

Structural requirements: icon-only action with accessible name in the top bar
(a labeled entry on phones);
the dialog lists the four steps and the protected-task note and has exactly
two actions. Wording and spacing are illustrative; copy is localized.
