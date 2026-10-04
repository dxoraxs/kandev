---
status: active
system: workspaces
created: 2026-10-05
owners:
  - kandev
---

# Repository Cleanup Requirements

## Overview

Local repositories accumulate feature branches, agent branches, remote branches
that were never checked out, and worktrees left behind by finished work. Folding
all of it back into the default branch and removing what is merged is a
repetitive, judgment-heavy chore.

This capability adds a cleanup action to a repository. It starts an agent task
that merges every branch and worktree into the default branch on the
repository's main checkout, resolves conflicts, removes merged branches locally
and on the remote, removes worktrees, and pushes, after one confirmation from
the owner.

The workspace system owns this contract because it owns repository
registration and the repository settings surface. The task launcher it uses is
owned by the task system:
[Plan file adaptation system design](../../tasks/system-design/plan-file-adaptation.md#maintenance-task-launcher).
The execution model is recorded in
[ADR 2026-10-05 repository maintenance agent tasks](../../../decisions/2026-10-05-repository-maintenance-agent-tasks.md).

## Terminology

- **Default branch:** The repository's configured default branch, or the
  remote's `HEAD` branch when none is configured.
- **Cleanup task:** A task that runs an agent on the repository's main checkout
  with the built-in cleanup prompt.
- **Protected branch or worktree:** A branch or worktree attached to a
  non-archived Kandev task other than the cleanup task itself.

## Requirements

### REQ-WORKSPACES-REPO-CLEANUP-001: Cleanup action

**Intent:** The owner starts a repository cleanup with one action from the
repository.

**User story:** As a workspace owner, I want a cleanup button on a repository,
so that an agent folds every branch into the default branch and removes what is
merged.

#### Acceptance criteria

- **AC-WORKSPACES-REPO-CLEANUP-001.1:** Each local repository row in workspace
  repository settings shall show an icon-only cleanup action with a broom icon,
  an accessible name, and a tooltip on fine pointers.
- **AC-WORKSPACES-REPO-CLEANUP-001.2:** When the owner activates the action, the
  system shall show a confirmation dialog that explains the steps (merge every
  branch and worktree into the default branch, commit uncommitted worktree
  changes, resolve conflicts, ask for confirmation, then delete merged branches
  locally and on the remote, remove worktrees, and push) and that the agent works
  in the repository's main checkout.
- **AC-WORKSPACES-REPO-CLEANUP-001.3:** When the owner confirms, the system shall
  create a cleanup task with the workspace default agent profile, start it on the
  repository's main checkout, and open the task.
- **AC-WORKSPACES-REPO-CLEANUP-001.4:** When any maintenance task (cleanup or
  plan adaptation) for the same repository is already active, the system shall open the existing task instead
  of creating another.
- **AC-WORKSPACES-REPO-CLEANUP-001.5:** When the workspace has no default agent
  profile or no visible workflow, the system shall not create a task and shall
  show a localized error that names the missing setting.
- **AC-WORKSPACES-REPO-CLEANUP-001.6:** For a repository whose source is not a
  local path, the action shall not be shown.
- **AC-WORKSPACES-REPO-CLEANUP-001.7:** On a phone, the action shall be reachable
  in the repository row with a touch target of at least 44px, and the
  confirmation shall use the shared dialog's phone presentation with full-width
  actions.
- **AC-WORKSPACES-REPO-CLEANUP-001.8:** When the `features.repositoryCleanup`
  runtime flag is off, the action shall not render and the endpoint shall reject
  the cleanup kind.

### REQ-WORKSPACES-REPO-CLEANUP-002: Merge phase

**Intent:** Every branch and worktree is folded into the default branch locally
before anything irreversible happens.

#### Acceptance criteria

- **AC-WORKSPACES-REPO-CLEANUP-002.1:** The cleanup prompt shall instruct the
  agent to fetch with pruning, update the default branch from the remote with a
  fast-forward-only pull, and enumerate local branches, remote branches without
  a local branch, and worktrees.
- **AC-WORKSPACES-REPO-CLEANUP-002.2:** The prompt shall instruct the agent to
  commit uncommitted changes in each non-protected worktree, and in the main
  checkout, to that worktree's branch as a work-in-progress commit before
  merging.
- **AC-WORKSPACES-REPO-CLEANUP-002.3:** The prompt shall instruct the agent to
  merge each branch that has commits not in the default branch with a merge
  commit, to resolve conflicts itself, and to run the project's test command
  after a conflicted merge.
- **AC-WORKSPACES-REPO-CLEANUP-002.4:** The prompt shall instruct the agent to
  undo a merge when it is not confident in a conflict resolution or the tests
  fail, keep that branch, and report it.
- **AC-WORKSPACES-REPO-CLEANUP-002.5:** The prompt shall list protected branches
  and worktrees and instruct the agent not to commit to, merge, delete, or remove
  them.

### REQ-WORKSPACES-REPO-CLEANUP-003: Confirmation and irreversible phase

**Intent:** Remote deletion and push happen only after the owner agrees with a
concrete summary.

#### Acceptance criteria

- **AC-WORKSPACES-REPO-CLEANUP-003.1:** The prompt shall instruct the agent,
  after the merge phase, to stop and post a summary of merged branches, kept
  branches with reasons, local branches and worktrees to remove, remote branches
  to delete, and the push target, and to wait for the owner's explicit
  confirmation in the task conversation.
- **AC-WORKSPACES-REPO-CLEANUP-003.2:** The prompt shall instruct the agent, only
  after confirmation, to push the default branch, remove merged worktrees, delete
  merged local branches with a safe delete, and delete remote branches that are
  merged into the pushed default branch.
- **AC-WORKSPACES-REPO-CLEANUP-003.3:** The prompt shall forbid force pushes,
  force deletes of unmerged branches, deletion of the default branch, and
  rewriting published history; a rejected push shall be reported, not retried
  with force.
- **AC-WORKSPACES-REPO-CLEANUP-003.4:** The prompt shall instruct the agent to
  finish with a report of what was merged, kept, deleted locally and remotely,
  removed, and pushed.

## Out of scope

- Repositories whose source is not a local path.
- Backend-enforced git permissions or a technical gate on the confirmation.
- Scheduling cleanup automatically.
- Cleaning up Kandev-managed worktrees of non-archived tasks; those follow the
  task lifecycle.
- Pull-request based merging through a provider.
