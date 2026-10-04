---
status: active
system: tasks
created: 2026-10-04
owners:
  - kandev
---

# Repository Plan Files Requirements

## Overview

Teams keep implementation plans as Markdown files inside their repositories.
This capability turns every plan file that declares a board status into a task
on a chosen workflow board, keeps the task current while the file changes, and
writes board edits back into the file. A plan written in any local repository
of a workspace appears on that workspace's board without a separate script.

The repository file stays the source of truth. The board is a projection of
the files that also accepts a small set of edits (status, order, priority) and
records them in the file.

The task system owns this contract because it owns task identity, external
identifiers, workflow steps, moves, ordering, and archival. Repository
registration and worktrees stay with the [workspace system](../../workspaces).
The agent-authored per-task plan document (`task_plans`) is a different
capability; see [Task plan requirements](plan-safe-edits.md).

## Terminology

- **Plan file:** A Markdown file in a scanned plan directory of a local
  repository whose YAML frontmatter contains a `board` key.
- **Board status:** The value of the `board` key. The closed set is `queued`,
  `in_progress`, `waiting_owner`, `waiting_external`, `deferred`, `done`, and
  `hidden`.
- **Plan board:** The workflow that a workspace selects to display its plan
  files.
- **Status mapping:** The per-workspace assignment of each board status except
  `hidden` to one step of the plan board.
- **Handoff step:** A plan board step that no board status maps to, typically
  one that starts an agent on entry.
- **Plan task:** A task that this capability created or adopted for a plan file.
- **Sync pass:** One reconciliation of every plan file of a workspace with its
  plan tasks.

## Requirements

### REQ-TASKS-PLAN-FILES-001: Plan file format

**Intent:** Every project uses one machine-readable format, so tools and agents
can create and update plan files without project-specific parsing.

#### Acceptance criteria

- **AC-TASKS-PLAN-FILES-001.1:** When a Markdown file starts with a YAML
  frontmatter block that contains a `board` key, the system shall treat the
  file as a plan file. A file without frontmatter or without a `board` key
  shall be ignored and shall not produce a task or an error.
- **AC-TASKS-PLAN-FILES-001.2:** The system shall accept these optional
  frontmatter keys: `title` (text), `priority` (`critical`, `high`, `medium`,
  or `low`; default `medium`), `order` (a decimal number), `executor` (text),
  `depends_on` (a list of plan file names in the same directory), and
  `external_id` (text, at most 255 bytes). Unknown keys shall be preserved and
  ignored.
- **AC-TASKS-PLAN-FILES-001.3:** When `title` is absent, the system shall use
  the text of the first level-one heading of the file body, and the file name
  without its extension when the body has no such heading.
- **AC-TASKS-PLAN-FILES-001.4:** When a plan file has a `board` value outside
  the closed set or an invalid optional value, the system shall still show its
  plan task, keep its current step (or place a new task in the step mapped to
  `queued`), and show the parse error in the task description header. Board
  status values are compared case-insensitively after trimming whitespace.

### REQ-TASKS-PLAN-FILES-002: Plan files appear on the plan board

**Intent:** A plan saved in any local repository of an enabled workspace shows
up on the board without manual task creation.

**User story:** As a workspace owner, I want every plan file in my
repositories to appear as a task on one board, so that I can see all open plans
across projects in one place.

#### Acceptance criteria

- **AC-TASKS-PLAN-FILES-002.1:** When plan files are enabled for a workspace,
  the system shall scan every local repository of that workspace. The scanned
  directories default to `docs/plans` and `docs/superpowers/plans`, relative to
  the repository root, and the workspace owner can change the list. Only `*.md`
  files directly inside a scanned directory are read.
- **AC-TASKS-PLAN-FILES-002.2:** When a plan file is created or changed in the
  working tree of a scanned repository, the system shall reflect the change on
  the plan board within 90 seconds, including uncommitted changes.
- **AC-TASKS-PLAN-FILES-002.3:** The system shall create one plan task per plan
  file in the step that the file's board status maps to. The task title shall
  be the plan title, prefixed with `[<executor>] ` when `executor` is set and
  the status is not `done`, truncated to the task title limit with an ellipsis.
- **AC-TASKS-PLAN-FILES-002.4:** The task description shall start with a header
  that names the repository and the file path, the executor when set, links to
  the plan tasks of the `depends_on` entries that exist, and any parse error,
  followed by the file body without its frontmatter. A body longer than
  16 KiB shall be cut to at most 16 KiB, and the header shall say that the
  full plan is in the file. Dependencies shall not block task launch.
- **AC-TASKS-PLAN-FILES-002.5:** Within a step, plan tasks shall be ordered by
  `order` ascending; tasks without `order` follow in file path order. The task
  priority shall equal the file's `priority`.
- **AC-TASKS-PLAN-FILES-002.6:** When a plan file is deleted, moved out of the
  scanned directories, or set to `hidden`, the system shall archive its plan
  task. When the file returns with a visible status, the system shall unarchive
  the same task instead of creating a new one.
- **AC-TASKS-PLAN-FILES-002.7:** When a plan file is unchanged since the last
  pass and its task is where the file says, a sync pass shall not modify the
  task or emit a task event.
- **AC-TASKS-PLAN-FILES-002.8:** The system shall not move a plan task whose
  session is starting or running an agent turn. It shall apply the pending
  move on a later pass after the turn settles.

### REQ-TASKS-PLAN-FILES-003: Board edits write back to the plan file

**Intent:** The owner can manage plans from the board, and the repository file
remains the single source of truth that agents and other tools read.

**User story:** As a workspace owner, I want to drag a plan to another column
and have the plan file record the new status, so that the board and the file
never disagree for long.

#### Acceptance criteria

- **AC-TASKS-PLAN-FILES-003.1:** When a plan task is moved, by a person or by
  a workflow action, into a step that a board status maps to and the file
  holds a different status, the system shall write that status to the file's
  `board` key.
- **AC-TASKS-PLAN-FILES-003.2:** When a plan task is reordered within its step,
  the system shall write `order` values so that the next pass reproduces the
  new order. When both new neighbours carry `order`, only the moved task's file
  shall change, receiving a value between theirs. Otherwise the system shall
  also write `order` to the plan tasks above the moved task in that step, and
  to no other files.
- **AC-TASKS-PLAN-FILES-003.3:** When a plan task's priority is changed on the
  board, the system shall write the new value to the file's `priority` key.
- **AC-TASKS-PLAN-FILES-003.4:** A write shall change only the frontmatter keys
  it sets. Every other byte of the file, including the body, other keys, key
  order, and comments, shall stay unchanged. The write leaves the change
  uncommitted in the repository working tree.
- **AC-TASKS-PLAN-FILES-003.5:** When the file changed on disk after the system
  last read it, the system shall not write. The next pass shall restore the
  task to the state the file describes, and the task shall show a notice that
  the board edit was not saved because the file changed. A board edit that a
  pass sees before it was written, while the file is unchanged, shall be
  written by that pass and not reverted.
- **AC-TASKS-PLAN-FILES-003.6:** When a plan task is moved into a handoff step,
  the system shall not write to the file. While the file's status is `queued`
  or `in_progress`, sync passes shall leave the task in the handoff step. Any
  other status shall move the task to its mapped step.

### REQ-TASKS-PLAN-FILES-004: Adopting existing tasks

**Intent:** A workspace that already tracks plans as tasks can switch to this
capability without losing task history or agent sessions.

#### Acceptance criteria

- **AC-TASKS-PLAN-FILES-004.1:** When a plan file sets `external_id`, the
  system shall use that value as the plan task's external identifier.
  Otherwise the identifier shall be derived from the repository and the file
  path relative to the repository root.
- **AC-TASKS-PLAN-FILES-004.2:** When a task with the plan file's external
  identifier already exists in the workspace, including an archived task, the
  system shall adopt and update that task instead of creating another one, and
  shall move it onto the plan board when it is on another workflow.
- **AC-TASKS-PLAN-FILES-004.3:** When two plan files of one workspace resolve
  to the same external identifier, the system shall sync neither of them and
  shall report the conflict for both files.

### REQ-TASKS-PLAN-FILES-005: Plan file settings and sync status

**Intent:** The workspace owner can turn the capability on, choose the board,
and see why a file did not appear.

#### Acceptance criteria

- **AC-TASKS-PLAN-FILES-005.1:** Workspace settings shall show a Plan files
  section where the owner can enable or disable sync, select an existing
  workflow or create a plan board from the built-in Plans template, assign a
  step to each board status, and edit the scanned directories.
- **AC-TASKS-PLAN-FILES-005.2:** When the owner creates a plan board from the
  template, the system shall fill the status mapping automatically. When the
  owner selects another workflow, saving shall require a step for every board
  status except `hidden`.
- **AC-TASKS-PLAN-FILES-005.3:** The section shall show the time and outcome of
  the last sync pass, counts of created, updated, moved, archived, and failed
  plan tasks, and one row per file that failed with its path and reason. A
  Sync now action shall start a pass immediately.
- **AC-TASKS-PLAN-FILES-005.4:** When sync is disabled, the system shall stop
  scanning and writing, and shall leave existing plan tasks unchanged.
- **AC-TASKS-PLAN-FILES-005.5:** On a phone, the section shall use the native
  settings layout with full-width controls, and every desktop action shall be
  available.
- **AC-TASKS-PLAN-FILES-005.6:** When the `features.planFiles` runtime flag is
  off, the section shall not render and no pass shall run.

### REQ-TASKS-PLAN-FILES-006: Safe file access

**Intent:** Reading and writing repository files cannot escape the repository
or overwhelm the board.

#### Acceptance criteria

- **AC-TASKS-PLAN-FILES-006.1:** The system shall read and write only regular
  files whose resolved path is inside the resolved repository root. A plan
  file that is itself a symbolic link, and a scanned directory that resolves
  outside the root, shall be skipped and reported.
- **AC-TASKS-PLAN-FILES-006.2:** A file larger than 1 MiB shall be skipped and
  reported. A scanned directory with more than 1,000 Markdown files shall be
  read up to that count and reported as truncated.
- **AC-TASKS-PLAN-FILES-006.3:** The system shall scan only repositories whose
  source is a local path. Remote repositories and their managed clones shall
  not be scanned or written.
- **AC-TASKS-PLAN-FILES-006.4:** A missing repository path, an unreadable
  file, or a failed write shall be reported for that repository or file and
  shall not stop the pass for other files.

## Out of scope

- Committing or pushing plan file changes. Writes stay uncommitted.
- Creating plan files from the board. New plans are created in the repository.
- Editing the plan body, title, executor, or dependencies from the board.
- Blocking task launch on `depends_on`.
- Scanning remote repositories through provider APIs.
- Filesystem change notifications. Polling satisfies the 90-second bound.
- Agent instructions that tell agents to set `board` when they create or
  finish a plan. Those live in each project's agent guidance.
