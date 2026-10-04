---
status: draft
system: tasks
created: 2026-10-05
owners:
  - platform
---

# Kanban Card Display Hints Requirements

## Overview

A kanban card currently shows the task title on one line and the first line of
the description under it in muted text. Boards fed by an external
synchronizer (for example a repository plan board) had to pack everything a
person scans for into those two strings: an executor prefix and a file number
in the title, and a date sentence and a raw Markdown header in the
description. The card shows the raw Markdown characters, the date blends into
the muted description, and the title is cut after one line.

This capability makes the card a title-first surface and gives external
writers a structured way to show the few facts that belong on a card: a date,
an executor, and checklist progress. Writers set them in a documented task
metadata object; the card renders each one as a compact tag. A task without
that object renders exactly like any other task, minus the description line.

Dependencies already have a card signal (the blocked badge of
[task dependencies](task-dependencies.md)); writers that know predecessors use
the dependency API, not this object.

The task system owns this contract because it owns the task record, its
metadata, and the kanban card.

## Terminology

- **Display hints:** the object stored under the `card_display` key of task
  metadata. Every field is optional.
- **Date tag:** the card tag rendered from `card_display.date` and
  `card_display.date_kind`.
- **Executor badge:** the card badge rendered from `card_display.executor`.
- **Progress chip:** the card chip rendered from `card_display.progress`.
- **Viewer's today:** the calendar day in the browser's local time zone.

## Requirements

### REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-001: Title-first card

**Intent:** The card names the task and leaves the description to the views
built for reading it.

**User story:** As a user scanning a board, I want each card to show its full
short name, so that I recognise the task without opening it.

#### Acceptance criteria

- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-001.1:** The kanban card shall render
  the task title on up to two lines and truncate it with an ellipsis after
  the second line.
- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-001.2:** The kanban card shall not
  render a description preview. The description stays available in the title
  hover card and the task description view.
- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-001.3:** The priority indicator, pull
  request icons, plugin indicators, and the card action menu shall keep their
  current behavior next to a two-line title.

### REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-002: Display hints contract

**Intent:** External writers have one documented, validated place for card
facts, and a malformed value never breaks a card.

**User story:** As the author of a board synchronizer, I want to set a date,
an executor, and progress on a task through the existing task API, so that
the card shows them without encoding them into the title or description.

#### Acceptance criteria

- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-002.1:** A client shall be able to set
  `metadata.card_display` when it creates or updates a task through the
  existing task API, and the board shall reflect a change without a reload.
- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-002.2:** The card shall ignore each
  invalid or unknown display hint field on its own: an invalid field renders
  nothing, valid sibling fields still render, and no raw value is shown.
- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-002.3:** A task without
  `card_display` shall render no date tag, executor badge, or progress chip.

### REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-003: Date tag

**Intent:** The next date that matters for a card stands out from the card
background and signals when it is close or past.

**User story:** As a user, I want a card's date as a tag with a colour that
tells me whether it is soon or overdue, so that I notice what needs attention
today.

#### Acceptance criteria

- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-003.1:** When `card_display.date` is a
  valid calendar date, the card shall render a date tag with a tinted
  background, a kind icon, and the day and abbreviated month in the UI
  locale, adding the year when it differs from the viewer's current year.
- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-003.2:** The date tag shall use the
  neutral tone by default, the warning tone when the date is the viewer's
  today or tomorrow, and the danger tone when the date is before the viewer's
  today. For `date_kind` `deferred`, the danger tone is not used: a past or
  current date uses the warning tone.
- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-003.3:** The date tag shall expose its
  full meaning ("Waiting until", "Due", or "Deferred until" plus the full
  date) as its accessible name and as a tooltip. An absent or unknown
  `date_kind` reads as "Due".

### REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-004: Executor badge and progress chip

**Intent:** Who acts next and how far the work is show as compact signals,
not text in the title.

**User story:** As a user, I want to see at a glance which cards an agent
owns, which are mine, and how much of each is done.

#### Acceptance criteria

- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-004.1:** When
  `card_display.executor.name` is a non-empty string, the card shall render a
  round executor badge: the first letter of the name for kind `agent` (or an
  absent kind), and a person icon for kind `person`. Its accessible name and
  tooltip name the executor.
- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-004.2:** When
  `card_display.progress` has integers `done` and `total` with
  `0 <= done <= total` and `total > 0`, the card shall render a progress chip
  reading `done/total`, with the success tone when `done` equals `total`.
- **AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-004.3:** The date tag, progress chip,
  and executor badge shall render on desktop and phone cards, stay inside the
  card width at 375 px, and not start a card drag or open the task when only
  hovered.

## Scenarios

- **GIVEN** a task whose description starts with `> File: docs/plans/06a.md`,
  **WHEN** the board renders, **THEN** the card shows only its title and
  chips, and no description text.
- **GIVEN** `card_display` `{date: "2026-10-05", date_kind: "waiting"}` and a
  viewer whose today is 2026-10-05, **WHEN** the card renders, **THEN** it
  shows a warning-tone tag "5 Oct" whose tooltip reads "Waiting until
  5 October 2026".
- **GIVEN** `card_display` `{date: "2026-13-40", progress: {done: 1,
  total: 3}}`, **WHEN** the card renders, **THEN** it shows the "1/3" chip and
  no date tag.
- **GIVEN** `card_display.executor` `{name: "Claude", kind: "agent"}`,
  **WHEN** the card renders, **THEN** it shows a round "C" badge whose tooltip
  reads "Executor: Claude".

## Out of scope

- A first-class due-date column, sorting or filtering by date.
- Editing display hints in the Kandev UI.
- Changes to the synchronizer that writes the hints (owned by its repository).
- Hiding repository chips on single-repository boards.
