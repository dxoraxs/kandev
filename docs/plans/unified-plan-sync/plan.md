---
created: 2026-10-05
status: draft
requirements:
  - REQ-TASKS-PLAN-CARD-001
  - REQ-TASKS-PLAN-CARD-002
system_design:
  - ../../specs/tasks/system-design/plan-file-card-facts.md
legacy_specs: []
---

# Implementation Plan: Unified Plan Sync

## Overview

Make the built-in plan files sync the only synchronizer between plan files and
Kandev boards. The dmhive project still runs its own synchronizer
(`apps/kandev-sync`, launchd job `com.dmhive.kandev-sync`) because the built-in
sync lacks a plan date and a current-status summary. Add both to the built-in
sync, then migrate dmhive's 66 cards to frontmatter, point its existing board
at the built-in sync, and retire the custom synchronizer.

Owner decision (2026-10-05): one synchronizer for every project.

## Scope

### In scope

- `date` frontmatter key, card facts in `metadata.card_display`, no executor
  title prefix.
- Workspace summary heading and summary-only descriptions.
- dmhive migration: card conversion, board configuration, adoption of the
  existing 64 tasks, retirement of the custom synchronizer and its rules.

### Out of scope

- Card rendering of the facts: the
  [Kanban card display hints](../kanban-card-display-hints/plan.md) package,
  developed on its own branch. Card facts are written before it lands and
  render once it does.
- Progress facts and person executors.

## Work orders

| Task | Wave | Depends on | Result |
| --- | --- | --- | --- |
| [01 Card facts](task-01-card-facts.md) | 1 | none | `date`, `card_display`, no title prefix. |
| [02 Summary section](task-02-summary-section.md) | 1 | none | `summary_heading` config, projection, settings field. |
| [03 dmhive migration](task-03-dmhive-migration.md) | 2 | 01, 02 | dmhive on the built-in sync; custom sync retired. |

Tasks 01 and 02 both touch `planfiles/projection.go` and `sync_task.go`; run
them one after the other.

## Progress

- [x] [Task 01: Card facts](task-01-card-facts.md)
- [ ] [Task 02: Summary section](task-02-summary-section.md)
- [ ] [Task 03: dmhive migration](task-03-dmhive-migration.md)

## Risks

- **Executor disappears from cards until display hints land.** The title prefix
  goes away in Task 01. Mitigation: land the display hints package first, or
  accept a short gap; Task 03 runs only after both are on the running build.
- **Duplicated tasks in dmhive.** Adoption needs `external_id: plan:<NN>` in
  every card; Task 03 verifies the task count and IDs before and after.
- **Owner's uncommitted edits in dmhive.** Task 03 stages only its own changes
  and asks before deleting the custom synchronizer's code.

## Verification strategy

- Go tests for date parsing, card facts projection, metadata merge and
  idempotency, summary selection with code fences, config validation and
  column addition.
- Vitest and the existing plan files desktop and phone E2E specs extended for
  the summary heading field.
- Task 03: a before/after inventory of dmhive tasks through the API.

## ASCII UI preview

### UI-01: Summary heading field in the Plan files section

Desktop:

```text
+-- Plan files ---------------------------------------------- [on] --+
| Board      [ Планы dmhive     v ]                                   |
| ...status mapping...                                                |
| Directories   docs/plans                          [+ Add directory] |
| Summary section heading   [ Сейчас                       ]          |
|   Plan tasks show the plan header and this section only.            |
|                                          [Sync now]  [Save]         |
+---------------------------------------------------------------------+
```

Phone (below 768px): label above a full-width 44px input; help text below.

```text
+-- Plan files ----------- [on] --+
| Summary section heading         |
| [ Сейчас                     ]  |
| Plan tasks show the plan header |
| and this section only.          |
+---------------------------------+
```

Structural requirements: one labeled text input with help text inside the
existing section, saved by the existing Save action.
