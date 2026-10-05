---
status: active
system: tasks
created: 2026-10-05
owners:
  - kandev
---

# Plan File Adaptation Requirements

## Overview

[Repository plan files](repository-plan-files.md) show a Markdown file on the
plan board only when its frontmatter declares a `board` status. Existing plan
collections were written before that format existed, so a repository can hold
dozens of plans and the board stays empty without any visible reason.

This capability makes unadapted plan files visible and lets the workspace owner
hand their adaptation to an agent with one action. The agent infers each plan's
status from repository evidence, writes the frontmatter, and commits only the
plan files. The owner reviews the result in the task conversation.

The task system owns this contract because it extends repository plan files and
creates the adaptation task. Repository registration stays with the
[workspace system](../../workspaces). The execution model is recorded in
[ADR 2026-10-05 repository maintenance agent tasks](../../../decisions/2026-10-05-repository-maintenance-agent-tasks.md).

## Terminology

- **Unadapted plan file:** A Markdown file in a scanned plan directory of a
  local repository whose content has no closing-fenced frontmatter with a
  `board` key.
- **Adaptation task:** A task that runs an agent on a repository's main
  checkout with a built-in prompt to adapt that repository's unadapted plan
  files.

## Requirements

### REQ-TASKS-PLAN-ADAPT-001: Unadapted plan files are visible

**Intent:** The owner learns why plans do not appear on the board instead of
seeing an empty board with a successful sync.

**User story:** As a workspace owner, I want to see how many plan files in each
repository are not in the board format, so that I know the board is incomplete.

#### Acceptance criteria

- **AC-TASKS-PLAN-ADAPT-001.1:** When a sync pass scans a plan directory, the
  system shall count unadapted plan files per repository and include the total
  in the last pass status shown in the Plan files settings section.
- **AC-TASKS-PLAN-ADAPT-001.2:** When the owner opens the Plan files settings
  section and a local repository of the workspace has at least one unadapted
  plan file, the section shall show one row for that repository with its name,
  the unadapted file count, and an Adapt with agent action.
- **AC-TASKS-PLAN-ADAPT-001.3:** The unadapted count shall be available for a
  repository even when plan sync is disabled or not yet configured for the
  workspace, using the configured directories or, when none are configured,
  the default directories `docs/plans` and `docs/superpowers/plans`.
- **AC-TASKS-PLAN-ADAPT-001.4:** An unadapted file shall not produce a file
  error row and shall not change the pass outcome.

### REQ-TASKS-PLAN-ADAPT-002: Offer adaptation when a repository is added

**Intent:** Adding a repository that already holds plans leads straight to
getting them on the board.

**User story:** As a workspace owner, I want Kandev to offer adapting plans as
soon as I add a repository that has them, so that I do not have to discover the
missing format later.

#### Acceptance criteria

- **AC-TASKS-PLAN-ADAPT-002.1:** When the owner adds a local repository to a
  workspace and the repository has at least one unadapted plan file, the system
  shall show a dialog with the repository name, the unadapted file count, an
  Adapt with agent action, and a Not now action.
- **AC-TASKS-PLAN-ADAPT-002.2:** When the added repository has no unadapted plan
  files, or its source is not a local path, the system shall not show the
  dialog.
- **AC-TASKS-PLAN-ADAPT-002.3:** Not now shall close the dialog without side
  effects; the settings row from AC-TASKS-PLAN-ADAPT-001.2 remains available.

### REQ-TASKS-PLAN-ADAPT-003: Start an adaptation task

**Intent:** One action starts an agent that adapts a repository's plans the way
an experienced owner would, without manual setup.

#### Acceptance criteria

- **AC-TASKS-PLAN-ADAPT-003.1:** When the owner chooses Adapt with agent for a
  local repository, the system shall create a non-ephemeral task on the start
  step of the workspace's first visible workflow other than the plan board,
  start it with the workspace default agent profile on the repository's main
  checkout, and open that task. When no such workflow exists, the system shall
  not create a task and shall show a localized error.
- **AC-TASKS-PLAN-ADAPT-003.2:** When plan sync is not configured for the
  workspace, starting an adaptation task shall also create a plan board from
  the built-in Plans template and save an enabled configuration with the
  default directories. The dialog and the settings row shall state this before
  the owner confirms.
- **AC-TASKS-PLAN-ADAPT-003.3:** When the workspace has no default agent
  profile, the system shall not create a task and shall show a localized error
  that names the missing setting.
- **AC-TASKS-PLAN-ADAPT-003.4:** When any maintenance task (plan
  adaptation or cleanup) for the same repository is already active, the system
  shall not create another task and shall open the existing one.
- **AC-TASKS-PLAN-ADAPT-003.5:** When the repository's source is not a local
  path, the action shall not be offered.

### REQ-TASKS-PLAN-ADAPT-004: Adaptation agent contract

**Intent:** The agent's result is predictable, reviewable, and limited to the
plan files.

#### Acceptance criteria

- **AC-TASKS-PLAN-ADAPT-004.1:** The adaptation prompt shall instruct the agent
  to classify every unadapted plan file into one board status using repository
  evidence (explicit status sections, merged branches and commits, the presence
  of the code the plan creates, superseding documents) and to treat checkbox
  counts as unreliable evidence.
- **AC-TASKS-PLAN-ADAPT-004.2:** The prompt shall instruct the agent to add a
  frontmatter block with `board`, `priority`, `order`, and, only for an
  explicit dependency on an unfinished plan, `depends_on`, at the top of each
  unadapted file without changing any other byte, and to leave files that
  already declare `board` unchanged.
- **AC-TASKS-PLAN-ADAPT-004.3:** The prompt shall instruct the agent to create
  one commit on the current branch that contains only the changed plan files,
  using a path-limited commit, to verify that the commit contains no other
  path, to leave changes the owner already staged untouched, and not to push.
- **AC-TASKS-PLAN-ADAPT-004.4:** The prompt shall instruct the agent to finish
  with a table of every adapted file with its status and one line of evidence,
  followed by the files whose status is uncertain.

## Out of scope

- Deterministic or heuristic conversion without an agent.
- Converting other plan formats that a project-specific tool already syncs,
  such as a body status line.
- Adapting remote repositories or managed clones.
- Pushing the adaptation commit.
- Changing the plan file format or the sync pass rules in
  [Repository plan files](repository-plan-files.md).
