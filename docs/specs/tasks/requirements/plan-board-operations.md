---
status: draft
system: tasks
created: 2026-10-05
owners:
  - kandev
---

# Plan Board Operations Requirements

## Overview

[Repository plan files](repository-plan-files.md) put Markdown plans on a
workflow board and write status, order, and priority back to the file. An
owner who runs several projects from that board still does a second set of
chores by hand: marking who took a plan, answering plans that wait for a
review, remembering dates, checking that predecessors are closed, keeping a
priority index, creating new plan files, and committing the edits the board
made.

This capability moves those chores to the board. Every operation records its
result in the plan file, so the repository stays the source of truth and
agents and other tools keep reading plain Markdown.

The task system owns this contract because it extends repository plan files
and uses task dependencies, task metadata, and the kanban card. Repository
registration stays with the [workspace system](../../workspaces). Card
rendering of dates, executors, progress, and flags is owned by
[Kanban card display hints](kanban-card-display-hints.md); the `date` key and
the summary section are owned by [Plan file card facts](plan-file-card-facts.md).

All behavior in this document is active only while the `features.planFiles`
runtime flag is on and plan files are enabled for the workspace.

## Terminology

- **Executor step:** A plan board step that the workspace owner assigns to one
  executor name. It is a handoff step of
  [Repository plan files](repository-plan-files.md) with a name attached.
- **Owner note:** A dated line that the system appends to the notes section of
  a plan file to record an owner decision or an automatic status change.
- **Notes section:** The level-two section of a plan file that receives owner
  notes. Its heading is a workspace setting and defaults to `Owner notes`.
- **Tracked item:** A file or directory named in a plan file's `tracks`
  frontmatter key whose completion counts toward the plan's progress.
- **Card flag:** A short warning shown on a plan task's card: `stale`,
  `open_items`, or `uncommitted`.
- **Plan index:** A generated Markdown file that lists the plan files of one
  scanned directory in board order.

## Requirements

### REQ-TASKS-PLAN-BOARD-OPS-001: Executor steps

**Intent:** A board can hand a plan to one of several agents by column, and the
file records who took it, so two executors do not pick up the same plan.

**User story:** As a workspace owner, I want one column per agent, so that
dropping a plan into a column both starts that agent and marks the plan as
taken by it.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-001.1:** Workspace plan-file settings shall let the
  owner assign an executor name (1 to 40 bytes) to any plan board step that no
  board status maps to. One step shall have at most one executor name, and one
  executor name at most one step. Saving shall reject a mapped status step, a
  duplicate, and an unknown step.
- **AC-TASKS-PLAN-BOARD-OPS-001.2:** When a plan task is moved into an executor
  step and the file's status is `queued` or `in_progress`, the system shall
  write that step's executor name to the file's `executor` key and shall not
  change the `board` key.
- **AC-TASKS-PLAN-BOARD-OPS-001.3:** When a plan task is moved into an executor
  step while the file's status is `in_progress` and its `executor` names a
  different executor, the system shall not write, shall move the task back to
  the step mapped to its status, and shall show a notice on the task that the
  plan is already taken by the named executor.
- **AC-TASKS-PLAN-BOARD-OPS-001.4:** A sync pass shall never move a plan task
  into an executor step. When a task sits in an executor step and the file's
  `executor` no longer equals that step's executor name, the pass shall move
  the task to the step mapped to the file's status.
- **AC-TASKS-PLAN-BOARD-OPS-001.5:** A handoff step without an executor name
  shall keep the behavior of AC-TASKS-PLAN-FILES-003.6.

### REQ-TASKS-PLAN-BOARD-OPS-002: Owner decisions

**Intent:** The owner answers a plan that waits for review from the task, and
the answer is recorded in the file where the executor reads it.

**User story:** As a workspace owner, I want to accept a plan or return it
with a comment in one action, so that the board and the file show my decision
without my editing Markdown.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-002.1:** When a plan task is in the step mapped to
  `waiting_owner`, its task view shall offer Accept and Return actions. Other
  plan tasks and non-plan tasks shall not show them.
- **AC-TASKS-PLAN-BOARD-OPS-002.2:** Accept shall take an optional comment and
  a result of `done` (default) or `queued`. The system shall write the result
  to the `board` key and append one owner note
  `- <YYYY-MM-DD> accepted: <comment>` (without the colon and comment when the
  comment is empty) to the notes section.
- **AC-TASKS-PLAN-BOARD-OPS-002.3:** Return shall require a non-empty comment.
  The system shall write `board: queued` and append one owner note
  `- <YYYY-MM-DD> returned: <comment>`.
- **AC-TASKS-PLAN-BOARD-OPS-002.4:** A comment shall be a single line of at
  most 500 characters after trimming; line breaks shall be replaced by spaces.
  An owner note shall be appended as the last line of the notes section. When
  the file has no notes section, the system shall add the section at the end
  of the file. Every other byte of the file shall stay unchanged.
- **AC-TASKS-PLAN-BOARD-OPS-002.5:** When the file changed on disk after the
  system last read it, the action shall fail without writing and shall tell
  the owner that the file changed. After a successful action the task shall
  be in the step mapped to the new status before the response returns.
- **AC-TASKS-PLAN-BOARD-OPS-002.6:** On a phone, the actions shall be reachable
  from the task's description view as full-width controls of at least 44 px,
  and the comment shall be entered in a bottom drawer.

### REQ-TASKS-PLAN-BOARD-OPS-003: Dates that act

**Intent:** A plan that waits for a date comes back to the owner on that date
instead of relying on the owner's memory.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-003.1:** When date wake-up is on for the workspace
  (default on) and a plan file with status `waiting_external` or `deferred`
  has a `date` that is today or earlier in the server's local time zone, the
  next sync pass shall write `board: waiting_owner` to the file and append the
  owner note `- <YYYY-MM-DD> date reached: was <previous status>`.
- **AC-TASKS-PLAN-BOARD-OPS-003.2:** A wake-up shall happen once per date: a
  file whose status is not `waiting_external` or `deferred` shall not be
  changed, whatever its date.
- **AC-TASKS-PLAN-BOARD-OPS-003.3:** When a wake-up is written, the system
  shall send a `plan_file.date_reached` notification to the workspace owner
  through every enabled notification provider subscribed to that event, with
  the plan title in the message. A notification failure shall not undo the
  wake-up.
- **AC-TASKS-PLAN-BOARD-OPS-003.4:** The kanban sort picker shall offer a date
  sort that orders the tasks of a column by `card_display.date` ascending and
  places tasks without a date after dated tasks in board order.

### REQ-TASKS-PLAN-BOARD-OPS-004: Plan dependencies

**Intent:** A plan that depends on an unfinished plan is visibly blocked and is
not started by an executor column.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-004.1:** For each `depends_on` entry that names a
  plan file of the same scanned directory with a plan task, the sync pass
  shall make the plan task depend on that task through
  [task dependencies](task-dependencies.md). The plan task's dependencies on
  other plan tasks shall equal the resolvable `depends_on` entries after every
  pass; dependencies on non-plan tasks shall be left alone.
- **AC-TASKS-PLAN-BOARD-OPS-004.2:** A `depends_on` entry that names no plan
  file, and a set of entries that task dependencies reject (a cycle or a
  self-reference), shall be reported as a file error for that plan and shall
  not stop the pass or change the plan task's existing dependencies.
- **AC-TASKS-PLAN-BOARD-OPS-004.3:** A blocked plan task that enters a step
  with agent auto-start shall not start an agent, as task dependencies define.
  This replaces the last sentence of AC-TASKS-PLAN-FILES-002.4.
- **AC-TASKS-PLAN-BOARD-OPS-004.4:** When a pass leaves a plan task's
  dependencies unchanged, it shall not call the dependency API for that task.

### REQ-TASKS-PLAN-BOARD-OPS-005: Progress and card flags

**Intent:** The card shows how far a plan is and warns about plans that look
wrong, without opening the file.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-005.1:** The system shall accept an optional
  `tracks` frontmatter key: a list of at most 20 paths relative to the
  repository root. An entry that leaves the repository, is a symbolic link, or
  does not exist shall be reported as a file error and skipped.
- **AC-TASKS-PLAN-BOARD-OPS-005.2:** Progress shall count Markdown task-list
  items outside fenced code blocks: `- [ ]` and `* [ ]` as open, `[x]` and
  `[X]` as done. The total is the items of the plan file body plus, for each
  tracked file, its items, plus, for each tracked directory, one item per
  `task-*.md` file directly inside it, done when that file's frontmatter
  `status` is `done`. When the total is greater than zero and the status is
  not `done`, the sync shall set `card_display.progress` to the counts;
  otherwise it shall remove the progress fact.
- **AC-TASKS-PLAN-BOARD-OPS-005.3:** The sync shall set the card flag
  `open_items` when the status is `done` and at least one counted item is
  open.
- **AC-TASKS-PLAN-BOARD-OPS-005.4:** The sync shall set the card flag `stale`
  when the status is `in_progress`, no agent turn is starting or running for
  the task, and neither the plan file nor any tracked item was modified within
  the workspace's stale threshold (default 7 days; 0 turns the flag off).
- **AC-TASKS-PLAN-BOARD-OPS-005.5:** The kanban card shall render each card
  flag of `card_display.flags` as a warning tag with an accessible name, after
  the other display hints, on desktop and phone. Unknown flag values shall be
  ignored. This extends the contract of
  REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-002.

### REQ-TASKS-PLAN-BOARD-OPS-006: Plan index

**Intent:** A repository keeps a readable list of its plans in board order
without anyone maintaining it by hand.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-006.1:** When the workspace sets an index file
  name (a `*.md` file name without a path; empty turns the index off), a sync
  pass shall write that file into every scanned directory that holds at least
  one plan file. The index shall list each visible plan file once, grouped by
  board status in the closed-set order, in board order within a group, with
  the title as a relative link, the priority, the executor, and the date.
- **AC-TASKS-PLAN-BOARD-OPS-006.2:** The index shall start with a marker line
  that names it as generated. The system shall overwrite only a file that
  starts with that marker or does not exist; another file with the same name
  shall be left unchanged and reported.
- **AC-TASKS-PLAN-BOARD-OPS-006.3:** The index shall contain no timestamp or
  other value that changes between passes over unchanged plan files, and the
  system shall not write the file when its content would not change.
- **AC-TASKS-PLAN-BOARD-OPS-006.4:** The index file shall not be counted as a
  plan file or as an unadapted plan file.

### REQ-TASKS-PLAN-BOARD-OPS-007: Create a plan from the board

**Intent:** An idea can be captured as a plan file from the board, including
from a phone, in the format the sync expects.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-007.1:** When the plan board is shown, the board
  shall offer a New plan action that asks for a local repository, one of the
  scanned directories, a title (required), a file name, a priority, and an
  optional executor and body.
- **AC-TASKS-PLAN-BOARD-OPS-007.2:** The file name shall default to a slug of
  the title's ASCII letters and digits and to `plan-<YYYYMMDD-HHMM>` when the
  title yields none; it shall match `[A-Za-z0-9][A-Za-z0-9._-]*\.md` and be at
  most 120 bytes.
- **AC-TASKS-PLAN-BOARD-OPS-007.3:** The system shall create the file with
  frontmatter `board: queued`, the title, the priority, and the executor when
  given, followed by a level-one heading with the title and the body. It shall
  refuse to replace an existing file and shall apply the file-safety rules of
  REQ-TASKS-PLAN-FILES-006.
- **AC-TASKS-PLAN-BOARD-OPS-007.4:** After the file is created, the plan task
  shall be on the board before the action reports success.
- **AC-TASKS-PLAN-BOARD-OPS-007.5:** On a phone, the action shall be reachable
  from the board and the form shall use a full-height surface with full-width
  controls.

### REQ-TASKS-PLAN-BOARD-OPS-008: Git state of plan files

**Intent:** Board edits stay uncommitted by design; the owner sees which plans
have uncommitted edits and commits them without leaving Kandev.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-008.1:** When a scanned repository is a git
  working tree, each sync pass shall set the card flag `uncommitted` on every
  plan task whose file is modified, staged, or untracked, and clear it
  otherwise. A repository that is not a git working tree shall produce no flag
  and no error.
- **AC-TASKS-PLAN-BOARD-OPS-008.2:** The settings section shall show, per
  repository, the number of plan files and plan indexes with uncommitted
  changes and a Commit plan files action when the number is not zero.
- **AC-TASKS-PLAN-BOARD-OPS-008.3:** The action shall create one commit on the
  repository's current branch that contains exactly those files and no other
  change, including changes that were already staged. The message shall
  default to `docs(plans): update plan files` and can be edited. The system
  shall not push.
- **AC-TASKS-PLAN-BOARD-OPS-008.4:** When the repository is in the middle of a
  merge, rebase, or cherry-pick, or the commit fails (for example a hook
  rejects it), the action shall leave the working tree and index as they were
  and show the reason with the last lines of the git output.

### REQ-TASKS-PLAN-BOARD-OPS-009: Plans waiting for the owner across workspaces

**Intent:** An owner with several workspaces sees every plan that waits for
them in one list.

#### Acceptance criteria

- **AC-TASKS-PLAN-BOARD-OPS-009.1:** The application shall offer a Waiting for
  owner page that lists, across every workspace the user can access with plan
  files enabled, the plan tasks in the step mapped to `waiting_owner`, with
  the workspace name, title, repository, date, and executor, ordered by date
  ascending with undated plans last.
- **AC-TASKS-PLAN-BOARD-OPS-009.2:** Selecting a row shall open the task. The
  navigation entry shall show the number of listed plans and shall be hidden
  when the `features.planFiles` flag is off.
- **AC-TASKS-PLAN-BOARD-OPS-009.3:** A workspace that fails to load shall be
  named in the page as failed and shall not hide the plans of other
  workspaces.
- **AC-TASKS-PLAN-BOARD-OPS-009.4:** On a phone, the page shall be reachable
  from the navigation sheet and shall render one full-width row per plan with
  a tap target of at least 44 px.

## Out of scope

- Pushing commits, and committing files other than plan files and plan
  indexes.
- Showing tracked items as subtasks, and editing `depends_on`, `tracks`, the
  title, or the body from the board.
- Writing the `date` key from the board.
- Starting an agent from an owner decision. A returned plan moves to the step
  mapped to `queued`; the owner hands it over by column.
- Project-specific card numbering for new plan files. The owner types the
  file name.
- Per-user notification preferences beyond the existing provider
  subscriptions.
- Migrating a specific project's plan files.
