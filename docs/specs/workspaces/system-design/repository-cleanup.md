---
status: draft
system: workspaces
requirements:
  - REQ-WORKSPACES-REPO-CLEANUP-001
  - REQ-WORKSPACES-REPO-CLEANUP-002
  - REQ-WORKSPACES-REPO-CLEANUP-003
---

# Repository Cleanup System Design

## Purpose and boundaries

The workspace system owns the cleanup action on a repository and its prompt
contract. Task creation and launch reuse the repository maintenance task
launcher owned by the task system:
[Maintenance task launcher](../../tasks/system-design/plan-file-adaptation.md#maintenance-task-launcher).
This design adds the `repository_cleanup` kind, its prompt, its protected-list
input, its feature flag, and the repository-row UI. It adds no git automation
to the backend.

Contracts used but not owned here:

- Endpoint, active-task guard, profile, workflow, and base-branch resolution:
  the task-system launcher.
- Kandev worktree records: `worktree.SQLiteStore.GetWorktreesByRepositoryID`.
- Repository settings surface: `workspace-repositories-client.tsx` and
  `repository-card-preview.tsx`.

Execution model: [ADR 2026-10-05 repository maintenance agent tasks](../../../decisions/2026-10-05-repository-maintenance-agent-tasks.md).

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-WORKSPACES-REPO-CLEANUP-001` | [Frontend](#frontend), [Feature flag](#feature-flag), [Cleanup kind](#cleanup-kind) |
| `REQ-WORKSPACES-REPO-CLEANUP-002` | [Cleanup prompt](#cleanup-prompt), [Protected list](#protected-list) |
| `REQ-WORKSPACES-REPO-CLEANUP-003` | [Cleanup prompt](#cleanup-prompt) |

## Components and responsibilities

| Component | Responsibility |
| --- | --- |
| Launcher kind `repository_cleanup` | Availability check against the flag; prompt variables. |
| Protected-list builder | Collects branches and worktree paths of non-archived Kandev tasks for the repository. |
| `config/prompts/repository-cleanup.md` | Built-in cleanup prompt template. |
| Runtime flag `features.repositoryCleanup` | Release toggle for the action and the kind. |
| Web `RepositoryCleanupButton` | Broom icon button in `RepositoryPreview`. |
| Web `RepositoryCleanupDialog` | Confirmation dialog; calls the launcher and navigates to the task. |

## Data and contracts

### Cleanup kind

`POST /api/v1/repositories/:id/maintenance-tasks` with
`{"kind": "repository_cleanup"}`. Responses and reasons are the launcher's.
`kind_unavailable` is returned when the flag is off. The task title is
`Clean up repository: <repo>`.

### Protected list

Built at request time from `GetWorktreesByRepositoryID(repoID)`: every worktree
whose task exists and is not archived contributes its path and branch. The
cleanup task itself runs on the main checkout and has no worktree. The list is
passed to the prompt; nothing is stored.

### Cleanup prompt

`config/prompts/repository-cleanup.md`, variables (tag-stripped):

| Variable | Value |
| --- | --- |
| `{repository_name}` | Repository name. |
| `{repository_path}` | `local_path`. |
| `{default_branch}` | `repository.DefaultBranch`, or empty with the instruction to resolve `origin/HEAD`. |
| `{protected_branches}` | Branch names from the protected list, one per line, or `none`. |
| `{protected_worktrees}` | Worktree paths from the protected list, one per line, or `none`. |

The template is organized as phases that map to the acceptance criteria:

1. Inventory (`AC-WORKSPACES-REPO-CLEANUP-002.1`): `git fetch --prune`, checkout default branch,
   `git pull --ff-only`, list local branches, remote branches without a local
   counterpart, and `git worktree list`.
2. Snapshot (`AC-WORKSPACES-REPO-CLEANUP-002.2`): for each non-protected dirty worktree and a dirty
   main checkout, commit all changes to its branch with a
   `chore: wip snapshot before cleanup` message.
3. Merge (`AC-WORKSPACES-REPO-CLEANUP-002.3`, `002.4`): `git merge --no-ff` per branch with unique
   commits; on conflict resolve, run the project's test command found in
   `AGENTS.md`, `CLAUDE.md`, `README`, `Makefile`, or package scripts; on doubt or
   failure `git merge --abort` or reset to the pre-merge commit, keep the branch.
4. Confirmation (`AC-WORKSPACES-REPO-CLEANUP-003.1`): post the summary and end the turn waiting for
   the owner.
5. Apply (`AC-WORKSPACES-REPO-CLEANUP-003.2`): `git push origin <default>`, `git worktree remove` for
   merged non-protected worktrees, `git branch -d` for merged branches,
   `git push origin --delete` only for remote branches listed by
   `git branch -r --merged origin/<default>`.
6. Forbidden operations (`AC-WORKSPACES-REPO-CLEANUP-003.3`): `--force`, `--force-with-lease`,
   `branch -D`, deleting the default branch, rebasing or amending published
   commits.
7. Report (`AC-WORKSPACES-REPO-CLEANUP-003.4`).

Protected branches and worktrees (`AC-WORKSPACES-REPO-CLEANUP-002.5`) are excluded in every phase.
A Go test renders the template with sample variables and asserts each phase,
the forbidden operations, and the protected lists.

## Control flow

Button → dialog → confirm → `startRepositoryMaintenanceTask(repoId,
"repository_cleanup")` → launcher → task created and launched → frontend
navigates with `linkToTask(task_id)` (info toast when `existing: true`). The
conversation continues in the standard task view; the owner's confirmation is
an ordinary chat reply.

## Failure and recovery

| Situation | Behavior |
| --- | --- |
| Launcher rejects (`no_agent_profile`, `no_workflow`, `repository_not_local`, `kind_unavailable`) | Dialog stays open with the localized reason; no task. |
| Protected-list read fails | No task; generic localized error. Proceeding without the list could remove a live task's worktree. |
| Agent cannot resolve a conflict | Branch kept and reported; cleanup continues with other branches. |
| Push rejected | Reported; nothing deleted on the remote. |
| Owner declines confirmation | Local merges and WIP commits stay on local branches; nothing is pushed or deleted. |

## Security

The endpoint uses the repository's workspace authorization and the
`ScopeOrgSettingsManage`-equivalent rule that guards repository edits. The
agent uses the host's git credentials through the local executor, as any local
task does. Branch names and paths are tag-stripped before entering the prompt.

## Feature flag

Runtime flag `features.repositoryCleanup` (`KANDEV_FEATURES_REPOSITORY_CLEANUP`),
registered in `runtimeflags/registry.go` with kind feature, stability
experimental, risk high (it merges, deletes remote branches, and pushes),
restart required, matching `features.planFiles`: the launcher reads the value
from the startup config. `profiles.yaml`:
`prod`, `dev`, `e2e` false; E2E enables it through the backend fixture
environment. The frontend reads it with `useFeature("repositoryCleanup")`.

## Frontend

- `RepositoryPreview` gains a third action before Edit: an icon-only outline
  button with `IconBroom`, `aria-label`, and a tooltip on fine pointers. It
  calls `event.stopPropagation()` like its siblings and renders only for local
  repositories when the flag is on, and not in read-only mode.
- Desktop size follows the 28px control rule; on phones and coarse pointers the
  button is 44px.
- `RepositoryCleanupDialog` uses the shared `Dialog` (bottom sheet below
  640px): title, the step list, a note that the agent works in the main
  checkout, Cancel and Start cleanup. On phones the actions stack full width
  with 44px targets.
- Copy goes through `t("workspaces:…")` in English, pseudo, and the seven
  translated catalogs; no em dash.

## Observability

`repository_maintenance_task_total{kind="repository_cleanup",outcome}` from the
launcher. Git activity is visible in the task conversation; the backend does
not observe it.

## Related decisions

- [ADR 2026-10-05 repository maintenance agent tasks](../../../decisions/2026-10-05-repository-maintenance-agent-tasks.md)
