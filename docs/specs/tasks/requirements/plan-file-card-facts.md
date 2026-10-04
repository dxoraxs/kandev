---
status: draft
system: tasks
created: 2026-10-05
owners:
  - kandev
---

# Plan File Card Facts Requirements

## Overview

[Repository plan files](repository-plan-files.md) is meant to be the one
synchronizer between Markdown plans and a Kandev board. One project still runs
its own synchronizer because the built-in one cannot express two things that
project relies on: a date that matters for the plan (when to check an answer,
until when it is deferred, a deadline), and a short current-status section that
stands in for the whole plan in the task. Without them, moving that project to
the built-in sync loses information the owner reads every day.

This capability lets a plan file declare a date and lets a workspace name a
summary section. The built-in sync turns the date and the executor into the
card's [display hints](kanban-card-display-hints.md) instead of packing them
into the title, and shows only the plan header and the summary section in the
task description when the section exists.

The task system owns this contract because it extends repository plan files
and writes task metadata.

## Terminology

- **Plan date:** The calendar date in a plan file's `date` frontmatter key.
- **Summary heading:** The text of a level-two heading, set per workspace in
  the plan files configuration, that marks a plan's current-status section.
- **Plan header:** The part of a plan file body before its first level-two
  heading.
- **Card facts:** The `card_display` object the sync writes into a plan task's
  metadata, as defined by
  [Kanban card display hints](kanban-card-display-hints.md).

## Requirements

### REQ-TASKS-PLAN-CARD-001: Plan date and executor become card facts

**Intent:** The next date that matters and who acts next show on the card as
structured facts, not as text in the title.

**User story:** As a workspace owner, I want a plan's date and executor on its
card, so that I see what needs attention without opening the plan.

#### Acceptance criteria

- **AC-TASKS-PLAN-CARD-001.1:** The system shall accept an optional `date`
  frontmatter key holding a calendar date in `YYYY-MM-DD` form. An invalid
  value shall be reported as a parse error in the task description header like
  other invalid optional values, and shall produce no date fact.
- **AC-TASKS-PLAN-CARD-001.2:** When a plan file has a valid `date` and its
  board status is not `done`, the sync shall set `card_display.date` to that
  date and `card_display.date_kind` to `waiting` for `waiting_owner` and
  `waiting_external`, `deferred` for `deferred`, and `due` otherwise.
- **AC-TASKS-PLAN-CARD-001.3:** When a plan file sets `executor` and its board
  status is not `done`, the sync shall set `card_display.executor` to
  `{name: <executor>, kind: "agent"}`. The task title shall no longer carry the
  `[<executor>] ` prefix.
- **AC-TASKS-PLAN-CARD-001.4:** When a plan file yields no card facts, the sync
  shall remove `card_display` from the task metadata. Every other metadata key
  shall be preserved, and a pass where the facts are unchanged shall not update
  the task.

### REQ-TASKS-PLAN-CARD-002: Summary section in the task description

**Intent:** A plan with a current-status section is represented by that
section, so the task and the agent launched from it see the current state
first.

**User story:** As a workspace owner, I want plan tasks to show the plan header
and the current-status section, so that the task reads as the current state of
the plan and not as its full history.

#### Acceptance criteria

- **AC-TASKS-PLAN-CARD-002.1:** The plan files configuration shall accept an
  optional summary heading of at most 100 characters, editable in the Plan
  files settings section on desktop and phone.
- **AC-TASKS-PLAN-CARD-002.2:** When a summary heading is set and a plan body
  has a level-two heading outside code fences whose text equals it after
  trimming and case folding, the description body shall be the plan header
  followed by that section up to the next level-two heading, and the
  description header shall state that the full plan is in the file.
- **AC-TASKS-PLAN-CARD-002.3:** When no summary heading is set or the plan has
  no matching section, the description shall be built as before, from the
  whole body.

## Scenarios

- **GIVEN** a plan with `board: waiting_external` and `date: 2026-10-12`,
  **WHEN** the sync runs, **THEN** the task metadata holds
  `card_display: {date: "2026-10-12", date_kind: "waiting"}`.
- **GIVEN** a plan with `executor: Claude` and `board: done`, **WHEN** the
  sync runs, **THEN** the task has no executor fact and no title prefix.
- **GIVEN** the summary heading `Сейчас` and a plan with `## Сейчас` and
  `## История`, **WHEN** the sync runs, **THEN** the description shows the
  header and the `Сейчас` section only.
- **GIVEN** a task whose metadata holds another key, **WHEN** the sync writes
  card facts, **THEN** that key is unchanged.

## Out of scope

- Rendering the facts on the card (owned by
  [Kanban card display hints](kanban-card-display-hints.md)).
- Progress facts from checklists; person executors.
- Writing the date back from the board.
- Migrating a specific project's plan files (an implementation work order).
