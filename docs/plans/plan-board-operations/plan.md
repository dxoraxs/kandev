---
created: 2026-10-05
status: draft
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-001
  - REQ-TASKS-PLAN-BOARD-OPS-002
  - REQ-TASKS-PLAN-BOARD-OPS-003
  - REQ-TASKS-PLAN-BOARD-OPS-004
  - REQ-TASKS-PLAN-BOARD-OPS-005
  - REQ-TASKS-PLAN-BOARD-OPS-006
  - REQ-TASKS-PLAN-BOARD-OPS-007
  - REQ-TASKS-PLAN-BOARD-OPS-008
  - REQ-TASKS-PLAN-BOARD-OPS-009
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
legacy_specs: []
---

# Implementation Plan: Plan Board Operations

## Overview

Extend `internal/planfiles` so the plan board records executor claims, owner
decisions, date wake-ups, dependencies, progress, warnings, an index, new
plans, and commits in the plan files. The order is: pure format functions
first, then config and storage, then one vertical slice per requirement, and
finally E2E and public docs. Slices that do not depend on the `date` key or
the `card_display` merge come first, because those two arrive with
[Unified plan sync](../unified-plan-sync/plan.md) tasks 01 and 02, which
another branch is still delivering.

Related decisions:
[ADR 2026-10-04](../../decisions/2026-10-04-repository-plan-files-source-of-truth.md),
[ADR 2026-10-05](../../decisions/2026-10-05-plan-board-writes-notes-and-commits.md).

## Prerequisites

- Branch `feat/plan-board-operations` starts from
  `feat/repository-maintenance-tasks` with `feat/kanban-card-display-hints`
  merged in.
- Tasks 09, 10, and 11 require
  [Unified plan sync task 01](../unified-plan-sync/task-01-card-facts.md)
  (`PlanFile.Date`, `projectCardFacts`, the metadata compare-and-merge) to be
  on the branch. Task 02 requires nothing from it, but both it and
  [Unified plan sync task 02](../unified-plan-sync/task-02-summary-section.md)
  add columns to `plan_file_configs`; whichever lands second reuses the
  other's add-column helper.

## Scope

### In scope

- Format: `tracks`, item counting, note appending, new-plan rendering.
- Config, storage, and API for executor steps, notes heading, wake-up, stale
  threshold, and index file.
- Executor steps with claim write-back and conflict notice.
- Owner decision endpoint and task-page decision bar.
- Git state, commit action, plan creation, plan index, dependencies.
- Progress and card flags, date wake-up with notification, date sort.
- Waiting-for-owner page across workspaces.
- E2E on desktop and phone; public documentation; spec promotion.

### Out of scope

- Unified plan sync tasks 01 to 03 (their own package).
- Pushing, subtasks for tracked items, editing `date`, `depends_on`, or
  `tracks` from the board, stopping a session after a claim conflict.

## Technical approach

Backend, all in `apps/backend/internal/planfiles` unless noted:

- `format/parse.go` (`Tracks`), `format/items.go`, `format/notes.go`,
  `format/newplan.go`.
- `store.go`, `models.go`, `service_config.go`: new columns and config
  fields; registrations in `internal/persistence/requiredstores` and
  `storeconformance`.
- `writeback_core.go` (`boardKeys`), `sync_task.go` (`desiredStep`,
  `staysInHandoff`): executor steps.
- New files: `edit.go` (`composeEdit`), `decision.go`,
  `handlers_decision.go`, `create.go`, `index.go`, `dependencies.go`,
  `wake.go`, `waiting.go`, `gitstate/gitstate.go`, `handlers_ops.go`.
- `scan/`: `CreateFile`, `WriteGenerated`, `FileInfo.ModTime`.
- `internal/notifications/service`: `plan_file.date_reached`.
- `internal/user/models/kanban_view_preferences.go`: `date_asc`.
- Wiring in `internal/backendapp` (`DateNotifier`, `WorkspaceLister`).

Frontend, in `apps/web`:

- `lib/api/domains/plan-files-api.ts`, `hooks/domains/plans/*`.
- `components/task/plan-decision-bar.tsx`,
  `components/kanban/new-plan-dialog.tsx`,
  `components/settings/plan-files-*.tsx`, `app/plans-waiting/`.
- `lib/kanban/card-display.ts`, `components/kanban-card-display-hints.tsx`,
  `lib/kanban/kanban-sort.ts`, `lib/kanban/task-order.ts`.
- Locales `planFiles.json`, `kanban.json` in `en`, `pseudo`, and the seven
  translated locales.

## ASCII UI preview

Structural requirements: control order, grouping, and the phone surfaces.
Spacing and copy are illustrative; copy is localized.

### UI-01: Plan decision bar (task page, plan waits for owner)

Covers AC-TASKS-PLAN-BOARD-OPS-002.1, 002.6. Desktop, under the task top bar:

```text
+----------------------------------------------------------------------+
| Task top bar                                                         |
+----------------------------------------------------------------------+
| (i) This plan waits for your decision.        [ Return ] [ Accept v ]|
+----------------------------------------------------------------------+
| panels ...                                                           |
```

`Accept v` is a split control: Accept and close (default) / Accept, back to
queue. Both actions open the comment dialog:

```text
+----------------------- Return plan ------------------------+
| Comment (required)                                         |
| [______________________________________________________]  |
|                                   [ Cancel ] [ Return ]    |
+------------------------------------------------------------+
```

Phone, in the description view; the trigger opens a bottom drawer:

```text
+------------------------------+      +------------------------------+
| Description                  |      | Plan decision          [ x ] |
| [ Plan decision          > ] |  ->  | Comment                      |
| plan text ...                |      | [__________________________] |
|                              |      | [        Accept and close  ] |
|                              |      | [   Accept, back to queue  ] |
|                              |      | [          Return          ] |
+------------------------------+      +------------------------------+
```

Drawer buttons are full width and at least 44 px. Return stays disabled
while the comment is empty.

### UI-02: Plan files settings additions

Covers AC-001.1, 003.1, 005.4, 006.1, 008.2. Added below the status mapping;
the phone layout stacks every control full width.

```text
Executor columns
  [ Hand to Claude      v ]  [ Claude        ]  [ x ]
  [ Hand to Codex       v ]  [ Codex         ]  [ x ]
  [ + Add executor column ]

Notes section heading   [ Owner notes        ]
[x] Move plans to "Waiting for owner" when their date arrives
Mark plans in progress as stale after  [ 7 ] days (0 = never)
Index file name         [ INDEX.md           ]  (empty = no index)

Uncommitted plan files
  dmhive        3 files      [ Commit plan files ]
  kandev        0 files
```

`Commit plan files` opens a dialog with the editable message and the file
list; a failure shows the reason and the last lines of git output inline.

### UI-03: New plan (plan board header)

Covers AC-007.1, 007.5. Desktop: a `New plan` button in the board header,
shown only on the plan board, opens a dialog:

```text
+------------------------- New plan --------------------------+
| Repository   [ dmhive                                   v ] |
| Directory    [ docs/plans                               v ] |
| Title        [____________________________________________] |
| File name    [ my-plan.md                                 ] |
| Priority     [ Medium v ]      Executor [ (none)        v ] |
| Body (optional)                                             |
| [__________________________________________________________]|
|                                     [ Cancel ] [ Create ]   |
+-------------------------------------------------------------+
```

Phone: entry `New plan` in the board display options; the form is a
full-height surface with an internal scroll region and a sticky Create
button.

### UI-04: Waiting for owner page

Covers AC-009.1 to 009.4. Desktop:

```text
Waiting for owner (5)
+-------------+--------------------------------+-----------+---------+---------+
| Workspace   | Plan                           | Repository| Date    | Executor|
+-------------+--------------------------------+-----------+---------+---------+
| dmhive      | 06a Instagram data deletion    | dmhive    | 5 Oct   | Claude  |
| dmhive      | 31 Chat checkout               | dmhive    |         | Claude  |
| kandev      | Plan board operations          | kandev    |         |         |
+-------------+--------------------------------+-----------+---------+---------+
! Could not load: tg-hub
```

Phone: one full-width row per plan (title, then workspace, date, executor on
a second line), each row a tap target of at least 44 px.

### UI-05: Card flags

Covers AC-005.5. Flags render after the date, executor, and progress hints
as warning pills: `Stale`, `Open items`, `Uncommitted`. Same row on phone.

```text
| Title of the plan                         |
| [5 Oct] [Claude] [3/7] [Stale] [Uncommitted] |
```

## Tests

| Criteria | Evidence |
| --- | --- |
| AC-005.1, 005.2 (counting, tracks) | `format/items_test.go`, `format/parse_tracks_test.go`, `projection_progress_test.go` |
| AC-002.4 (note bytes) | `format/notes_test.go`, `format/notes_fuzz_test.go` |
| AC-001.1 | `service_config_ops_test.go` |
| AC-001.2 to 001.5 | `writeback_executor_test.go`, `sync_executor_test.go` |
| AC-002.2, 002.3, 002.5 | `decision_test.go`, `handlers_decision_test.go` |
| AC-003.1 to 003.3 | `wake_test.go`, `internal/notifications/service/plan_date_test.go` |
| AC-003.4 | `apps/web/lib/kanban/task-order.test.ts`, `kanban_view_preferences_test.go` |
| AC-004.1 to 004.4 | `dependencies_test.go` |
| AC-005.3, 005.4 | `projection_flags_test.go` |
| AC-005.5 | `apps/web/lib/kanban/card-display.test.ts`, `components/kanban-card-display-hints.test.tsx` |
| AC-006.1 to 006.4 | `index_test.go`, `scan/generated_test.go` |
| AC-007.2 to 007.4 | `create_test.go`, `format/newplan_test.go`, `scan/create_test.go` |
| AC-008.1, 008.3, 008.4 | `gitstate/gitstate_test.go` (real temporary repositories), `git_ops_test.go` |
| AC-009.1, 009.3 | `waiting_test.go` |

## E2E tests

`apps/web/e2e/tests/plans/plan-board-operations.spec.ts` (project
`chromium`) and `mobile-plan-board-operations.spec.ts` (project
`mobile-chrome`):

| Flow | Criteria |
| --- | --- |
| Return a waiting plan with a comment; the file gains the note and `board: queued` | AC-002.1, 002.3, 002.6 |
| Create a plan from the board; the card appears and the file exists | AC-007.1, 007.4, 007.5 |
| A board edit shows the Uncommitted flag; Commit plan files clears it | AC-005.5, 008.1 to 008.3 |
| Waiting for owner page lists the plan and opens it | AC-009.1, 009.2, 009.4 |

## Work orders

- [ ] [Task 01: Format additions](task-01-format-additions.md)
- [ ] [Task 02: Config, storage, and settings fields](task-02-config-and-settings.md)
- [ ] [Task 03: Executor steps](task-03-executor-steps.md)
- [ ] [Task 04: Owner decisions](task-04-owner-decisions.md)
- [x] [Task 05: Git state and commit](task-05-git-state-and-commit.md)
- [ ] [Task 06: Create a plan from the board](task-06-create-plan.md)
- [ ] [Task 07: Plan index](task-07-plan-index.md)
- [ ] [Task 08: Plan dependencies](task-08-dependencies.md)
- [ ] [Task 09: Progress and card flags](task-09-progress-and-flags.md)
- [ ] [Task 10: Date wake-up and date sort](task-10-date-wake-up.md)
- [ ] [Task 11: Waiting for owner page](task-11-waiting-owner-page.md)
- [ ] [Task 12: E2E, public docs, and promotion](task-12-e2e-and-docs.md)

All work orders are sequential: they share `internal/planfiles`, the store
schema, `plan-files-api.ts`, and the locale files.

## Verification results

Pending.

## Risks

- Another branch is changing `projection.go`, `sync_task.go`, `store.go`,
  `service_config.go`, and `plan-files-section.tsx` for unified plan sync.
  Tasks here touch the same files; merge that branch in before tasks 09 to 11
  and resolve conflicts once.
- A claim conflict cannot prevent the step's `on_enter` auto-start from
  firing before the move back.
- Git hooks of a user repository run under the backend's environment and can
  be slow; the commit action uses the `GitInteractive` class and reports
  failures without retry.
- The wake-up writes files on a timer; an agent editing the same file loses
  no data because of the compare-and-swap, but the wake-up may be delayed by
  one pass.
