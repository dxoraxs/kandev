---
status: active
system: tasks
created: 2026-10-04
owners:
  - platform
---

# Task Description View Requirements

## Overview

A task that has no session currently opens on an "Agent" tab whose content is
the task description, with a large "no agent profile configured" notice inside
the text and a message composer below. The description loses its headings and
list markers, and the tab name promises an agent conversation that does not
exist yet.

This capability gives the task description its own view: a sessionless task
opens on a "Description" tab that renders the description as a Markdown
document. The "Agent" tab appears only once the task has a session. When no
agent is assigned, a compact notice sits in a corner of the description view.

Tasks synchronized from repository plan files are the main audience: their
description is a status summary written for a person, and most of them wait
on people or external events rather than on an agent.

The task system owns this contract because it owns task opening and the
sessionless state defined by
[task open without agent](task-open-without-agent.md). This document
supersedes these criteria of that requirement, which were removed from it when
this one became active:

| Superseded | Replaced by |
| --- | --- |
| `AC-TASKS-TASK-OPEN-WITHOUT-AGENT-002.1` | `AC-TASKS-TASK-DESCRIPTION-VIEW-001.1`, `001.3` |
| `AC-TASKS-TASK-OPEN-WITHOUT-AGENT-002.3` | `AC-TASKS-TASK-DESCRIPTION-VIEW-002.1` |
| `AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.1` | `AC-TASKS-TASK-DESCRIPTION-VIEW-003.1` |
| `AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.4` | `AC-TASKS-TASK-DESCRIPTION-VIEW-002.1` |
| `AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.5` | `AC-TASKS-TASK-DESCRIPTION-VIEW-004.1`, `004.2` |

## Terminology

- **Sessionless task:** a task with no session of any state.
- **Description document:** the task description rendered as Markdown with
  headings, bulleted and numbered lists, emphasis, block quotes, links, and
  code formatting.
- **Unassigned notice:** the neutral notice shown when the open reports that
  no agent profile resolved for the task.

## Requirements

### REQ-TASKS-TASK-DESCRIPTION-VIEW-001: Description tab

**Intent:** The description of a task is a document with its own tab, and a
sessionless task opens on it.

**User story:** As a user opening a task from a plan board, I want to see its
formatted status summary first, so that I know what is done, what is left,
and who acts next without reading an agent transcript.

#### Acceptance criteria

- **AC-TASKS-TASK-DESCRIPTION-VIEW-001.1:** When a sessionless task opens on
  the desktop task page, the system shall show an active "Description" tab
  that renders the description document, and shall show no "Agent" tab and no
  message composer.
- **AC-TASKS-TASK-DESCRIPTION-VIEW-001.2:** The user shall be able to open the
  "Description" tab from the panel add menu for any task, including tasks that
  have sessions.
- **AC-TASKS-TASK-DESCRIPTION-VIEW-001.3:** When a sessionless task is open in
  the board preview, the system shall render the description document in place
  of the conversation.
- **AC-TASKS-TASK-DESCRIPTION-VIEW-001.4:** A long description shall scroll
  vertically inside its view to its last line, and the page shall not scroll
  horizontally.

### REQ-TASKS-TASK-DESCRIPTION-VIEW-002: Agent tab appears with the first session

**Intent:** The "Agent" tab exists only when there is an agent conversation to
show.

#### Acceptance criteria

- **AC-TASKS-TASK-DESCRIPTION-VIEW-002.1:** When the first session of an open
  sessionless task appears (from "Start agent", a board hand-off, or another
  client), the system shall add an "Agent" tab with that session's
  conversation and make it active without a page reload.
- **AC-TASKS-TASK-DESCRIPTION-VIEW-002.2:** When the "Agent" tab is added, the
  "Description" tab shall stay open until the user closes it.

### REQ-TASKS-TASK-DESCRIPTION-VIEW-003: Compact corner notice

**Intent:** The unassigned state is visible and actionable without
interrupting the description.

#### Acceptance criteria

- **AC-TASKS-TASK-DESCRIPTION-VIEW-003.1:** When the unassigned notice is
  shown, it shall be a compact neutral element in the top-right corner of the
  description view, the description shall start at the top of the view, and
  the notice shall not cover description text. The system shall not show a
  destructive error, raw backend error text, or a Retry action for this state.
- **AC-TASKS-TASK-DESCRIPTION-VIEW-003.2:** The notice shall show a short title
  and the "Start agent" action directly. The full explanation and the
  workspace settings link (when the workspace is known) shall be reachable
  from the notice in one interaction.
- **AC-TASKS-TASK-DESCRIPTION-VIEW-003.3:** While the open is preparing a
  session, the same corner shall show the "Preparing workspace" status instead
  of the notice.

### REQ-TASKS-TASK-DESCRIPTION-VIEW-004: Phone composition

**Intent:** A phone gives the same order: description first, agent after
start.

#### Acceptance criteria

- **AC-TASKS-TASK-DESCRIPTION-VIEW-004.1:** On a phone-width viewport, while
  the task is sessionless, the first bottom-navigation item shall read
  "Description" and show the description document with no session picker and
  no message composer. When the first session appears, the item shall become
  "Chat" and show the conversation.
- **AC-TASKS-TASK-DESCRIPTION-VIEW-004.2:** On a phone-width viewport, the
  unassigned notice shall be a compact corner control that opens a bottom
  drawer with the explanation, "Start agent", and the workspace settings link
  as full-width touch targets of at least 44 px.

## Out of scope

- The backend `no_agent_profile` ensure outcome and the
  "Start agent" dialog, which stay as defined in
  [task open without agent](task-open-without-agent.md).
- Editing the description from the Description tab.
- Filtering the description by task status. External synchronizers decide what
  the description contains.
