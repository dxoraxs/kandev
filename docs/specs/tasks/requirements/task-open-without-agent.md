---
status: active
system: tasks
created: 2026-10-04
owners:
  - platform
---

# Task Open Without Agent Requirements

## Overview

Opening a task that has no session asks the backend to ensure one. When no
agent profile resolves for the task (no step, workflow, task, assignee, or
workspace default), the open currently fails and the task page shows a red
"could not start a session" error with a Retry button that can never succeed.
The page also shows an empty chat, so the task description, which often states
what the task is waiting for, is not visible anywhere.

This capability makes "no agent assigned" an expected state of task opening,
not a failure, and makes the task description readable before any session
exists. Tasks created by external synchronizers and repository plan files are
the common case: they arrive on workflow boards without an agent profile and
often sit on steps where nobody should start an agent automatically.

The task system owns this contract because it owns task opening, session
ensuring, and agent-profile resolution for a task. The rendered states are part
of the same outcome and live in this requirement.

## Terminology

- **Sessionless task:** a task with no session of any state.
- **Passive open:** opening a task page or task preview without an explicit
  launch action from the user.
- **Unassigned task:** a task for which no agent profile resolves through the
  task's resolution chain at the moment of opening.

## Requirements

### REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-001: Unassigned open is not a failure

**Intent:** A passive open of an unassigned sessionless task reports a typed
"no agent assigned" outcome instead of an error, creates nothing, and records
nothing as a failure.

**User story:** As a user browsing a board of synchronized tasks, I want to
open any task without an error, so that I can read what it needs before I
decide whether an agent should work on it.

#### Acceptance criteria

- **AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.1:** When a passive open runs for an
  unassigned sessionless task, the system shall return a successful ensure
  result whose outcome identifies that no agent profile resolved, with no
  session identifier.
- **AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.2:** When the unassigned outcome is
  returned, the system shall not create a session, a workspace, or an
  execution for the task, and shall not log the open at error level.
- **AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.3:** When an explicit launch action
  (not a passive open) runs for an unassigned task, the system shall keep
  reporting a failure, so plugin and automation callers that require a session
  still see one.
- **AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.4:** When an agent profile later
  resolves for the task (for example, a workspace default is set), the next
  passive open shall ensure a session through the normal path with no manual
  cleanup.

### REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-002: Readable sessionless task

**Intent:** The task description is the content of a sessionless task. The
user can read it while the task has no session, whatever the ensure outcome.
Where and how it renders is defined by
[task description view](task-description-view.md).

#### Acceptance criteria

- **AC-TASKS-TASK-OPEN-WITHOUT-AGENT-002.2:** When a sessionless task has an
  empty description, the system shall show a short placeholder that says the
  task has no description.

### REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-003: Calm unassigned state with next actions

**Intent:** An unassigned task explains itself in neutral styling and offers
the two actions that resolve it. The notice's placement and phone form are
defined by [task description view](task-description-view.md).

#### Acceptance criteria

- **AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.2:** The unassigned notice shall offer
  a "Start agent" action that opens the existing new-session dialog for the
  task, where the user picks an agent profile and starts a session.
- **AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.3:** When the task's workspace is
  known, the unassigned notice shall offer a link to the workspace settings,
  where a default agent profile can be set.
- **AC-TASKS-TASK-OPEN-WITHOUT-AGENT-003.6:** Ensure failures other than the
  unassigned outcome shall keep the existing error banner, error details, and
  Retry behavior.

## Out of scope

- A new workflow step kind for "waiting" steps. Kandev has no semantic
  knowledge of step names such as "Waiting for you". Steps without the
  `auto_start_agent` on-enter action already prepare a never-started session
  instead of starting an agent; that behavior is unchanged.
- Parsing structured fields (for example, a "next responsible" line) out of the
  task description. The description is rendered as written.
- Changing the agent-profile resolution order or adding a new default level.
- The prevent-auto-start-on-open preference
  ([requirement](prevent-agent-autostart-on-open.md)); its behavior is
  unchanged for tasks where a profile resolves.
