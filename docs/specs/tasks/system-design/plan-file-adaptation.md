---
status: draft
system: tasks
requirements:
  - REQ-TASKS-PLAN-ADAPT-001
  - REQ-TASKS-PLAN-ADAPT-002
  - REQ-TASKS-PLAN-ADAPT-003
  - REQ-TASKS-PLAN-ADAPT-004
---

# Plan File Adaptation System Design

## Purpose and boundaries

This design adds two things on top of
[Repository plan files](repository-plan-files.md):

1. Unadapted-file detection in `internal/planfiles`, exposed in the pass status
   and through a read endpoint.
2. A repository maintenance task launcher in the task system: one endpoint
   that creates and starts an agent task on a repository's main checkout with
   a built-in prompt. Plan adaptation is its first kind;
   [Repository cleanup](../../workspaces/system-design/repository-cleanup.md)
   is the second and reuses the launcher without changing it.

Contracts used but not owned here:

- Plan file format, scanner, sync pass, Plans template, and config API:
  [Repository plan files](repository-plan-files.md).
- Local executor behavior on the main checkout: `LocalPreparer` in
  `internal/agent/runtime/lifecycle/env_preparer_local.go`.
- Task creation and session launch: `task/service.CreateTask` and
  `orchestrator.LaunchSession`, as used by `httpStartQuickChat`.

Execution model: [ADR 2026-10-05 repository maintenance agent tasks](../../../decisions/2026-10-05-repository-maintenance-agent-tasks.md).

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-TASKS-PLAN-ADAPT-001` | [Unadapted detection](#unadapted-detection), [Frontend](#frontend) |
| `REQ-TASKS-PLAN-ADAPT-002` | [Frontend](#frontend) |
| `REQ-TASKS-PLAN-ADAPT-003` | [Maintenance task launcher](#maintenance-task-launcher), [Plan adaptation kind](#plan-adaptation-kind) |
| `REQ-TASKS-PLAN-ADAPT-004` | [Adaptation prompt](#adaptation-prompt) |

## Components and responsibilities

| Component | Responsibility |
| --- | --- |
| `planfiles` pass (`sync_pass.go`) | Counts files where `format.Parse` returns `ok == false` as unadapted, per repository. |
| `planfiles.Service.UnadaptedCounts` | Scans one or all local repositories of a workspace with configured or default directories and returns counts, independent of the sync config. |
| `planfiles.Service.EnsureBoard` | Creates the Plans board and saves an enabled config with default directories when the workspace has none. Idempotent. |
| `task/handlers` maintenance handler | `POST /api/v1/repositories/:id/maintenance-tasks`; validates kind and repository, resolves defaults, applies the active-task guard, creates and launches the task. |
| `task/service` maintenance helpers | Active-task lookup by metadata; default workflow resolution; the step is the auto-start step chosen by `CreateTask`. |
| `config/prompts/plan-file-adaptation.md` | Built-in adaptation prompt template. |
| Web `PlanFilesUnadaptedRows` | Per-repository rows with Adapt with agent in the Plan files section. |
| Web `AdaptPlansOfferDialog` | Offer shown after a repository is added. |
| Web `startRepositoryMaintenanceTask` | API client for the maintenance endpoint. |

## Data and contracts

### Unadapted detection

- `PassCounts` gains `Unadapted int` (`json:"unadapted"`). It is persisted inside
  the existing `last_counts` JSON column, so no migration is needed. It never
  adds a `FileErrorRow` and never affects `outcome`.
- `GET /api/v1/plan-files/unadapted?workspace_id=<id>[&repository_id=<id>]`
  returns:

  ```json
  {"repositories": [
    {"repository_id": "…", "repository_name": "…", "count": 21,
     "directories": ["docs/superpowers/plans"]}
  ]}
  ```

  Only local repositories with a count above zero are listed. `directories`
  are the scanned directories that contained unadapted files. The scan reuses
  `scan.ScanRepository` with the workspace config directories, or
  `DefaultDirectories()` when no config exists. It applies the same safety
  limits, reads no file outside the plan directories, and stores nothing.
- The route is registered with the other plan-file routes, so it exists only
  when `features.planFiles` is on.

### Maintenance task endpoint

`POST /api/v1/repositories/:id/maintenance-tasks`

Request: `{"kind": "plan_adaptation" | "repository_cleanup"}`.

Responses:

| Status | Body | When |
| --- | --- | --- |
| 201 | `{"task_id", "session_id", "existing": false}` | Task created and launched. |
| 200 | `{"task_id", "session_id", "existing": true}` | An active maintenance task of any kind exists for the repository. |
| 409 | `{"reason": "no_agent_profile"}` | Workspace has no default agent profile. |
| 409 | `{"reason": "no_workflow"}` | Workspace has no visible workflow other than the plan board. |
| 409 | `{"reason": "repository_not_local"}` | Repository source type is not `local` or `local_path` is empty. |
| 404 | `{"reason": "kind_unavailable"}` | The kind's feature flag is off. |

The frontend maps `reason` to localized copy and never renders backend text.

Task metadata on creation:

- `repository_maintenance_kind`: the kind.
- `repository_maintenance_repository_id`: the repository ID.
- `models.MetaKeyAgentProfileID`, `models.MetaKeyExecutorID` (`exec-local`).

## Control flow

### Maintenance task launcher

1. Load the repository; reject non-local repositories.
2. Check the kind's availability (plan adaptation needs a non-nil
   `PlanFilesSetup`; cleanup needs its own flag).
3. Active-task guard: find a non-archived task in the repository's workspace
   whose metadata carries any maintenance kind for this repository and whose latest session is not
   in a terminal state (`COMPLETED`, `FAILED`, `CANCELLED`). If found, return
   it with `existing: true`. The lookup and creation run under a
   per-repository mutex in the handler, so two clicks cannot create two tasks.
4. Resolve the agent profile: `workspace.DefaultAgentProfileID`, else reject.
5. Resolve the workflow: the first workflow of the workspace by sort order that
   is not hidden and is not the configured plan board; the task is created with `StartAgent`, so
   `CreateTask` places it on the workflow's auto-start step
   (`ResolveAutoStartStep`, falling back to the start step).
6. Resolve the base branch with `RepositoryCurrentBranch(repoID)`, so
   `LocalPreparer` performs no checkout.
7. Run the kind's preparation hook (plan adaptation: `EnsureBoard`).
8. Render the prompt (`sysprompt.Resolve(name, vars)` with every value passed
   through `sysprompt.StripTags`).
9. `CreateTask` with `WorkspaceID`, `WorkflowID`, `WorkflowStepID`, a localized
   title key resolved server-side to English (`Adapt plan files: <repo>`),
   `Description` = rendered prompt, `Repositories: [{RepositoryID, BaseBranch}]`,
   `ExecutorID: exec-local`, `StartAgent: true`, and the metadata above.
10. `LaunchSession` with `IntentStart`, the agent profile, and the prompt, as in
    `httpStartQuickChat`. On launch failure the task is kept and the error is
    returned; the existing task launch-failure recovery applies.

### Plan adaptation kind

The preparation hook is `PlanFilesSetup.EnsureBoard(ctx, workspaceID)`,
implemented by `planfiles.Service`: when `GetConfig` returns not found, it calls
`CreateBoard` and `PutConfig` with `enabled: true`, the returned mapping, and
`DefaultDirectories()`. An existing config, enabled or not, is left unchanged.
The interface is injected into the task handlers at startup; it is nil when
`features.planFiles` is off, which makes the kind unavailable.

### Adaptation prompt

`config/prompts/plan-file-adaptation.md`, variables:

| Variable | Value |
| --- | --- |
| `{repository_name}` | Repository name. |
| `{repository_path}` | `local_path`. |
| `{current_branch}` | Base branch from step 6. |
| `{plan_directories}` | Directories from `UnadaptedCounts` for this repository. |
| `{board_statuses}` | The closed status set from `format`. |

The template states, as numbered rules, the obligations in
`AC-TASKS-PLAN-ADAPT-004.1` to `004.4`: evidence sources with checkboxes marked
unreliable; frontmatter keys and placement with every other byte unchanged;
files with `board` left alone; one path-limited commit
(`git commit -- <directories>`), verified with
`git show --name-only --format= HEAD`, owner's staged changes untouched, no
push; the final table and uncertainty list. A Go test renders the template and
asserts each rule's presence.

## Frontend

- `plan-files-api.ts`: `getUnadaptedPlanFiles(workspaceId, repositoryId?)`.
- `repository-api` client: `startRepositoryMaintenanceTask(repositoryId, kind)`
  returning `{task_id, session_id, existing}` or a typed reason.
- `PlanFilesSection`: below the status block, `PlanFilesUnadaptedRows` lists
  each repository from the unadapted endpoint with name, count, and
  Adapt with agent. The section renders the rows even when sync is disabled.
  When the config is absent, the row's helper text says a Plans board will be
  created and sync enabled.
- `PlanFilesStatus` shows `unadapted` in the counts line when above zero.
- `AdaptPlansOfferDialog`: opened from `saveNewRepository` in
  `workspace-repositories-client.tsx` after a successful create, when
  `useFeature("planFiles")` is on and the unadapted endpoint returns a row for
  the new repository. Other repository add paths that route through
  `saveNewRepository` get the same behavior.
- On success both entry points navigate with `linkToTask(task_id)`; on
  `existing: true` they navigate to the existing task and show an info toast.
- Phone: the offer uses the shared `Drawer` (bottom sheet) when `isMobile`,
  and the shared `Dialog` otherwise; actions stack full width with 44px touch targets. Settings rows
  stack name and count over a full-width action button.
- All copy goes through `t()` in English, pseudo, and the seven translated
  catalogs; no em dash.

## Failure and recovery

| Situation | Behavior |
| --- | --- |
| Unadapted scan fails for one repository | That repository is omitted; others are listed. Errors are logged with repository ID only. |
| `EnsureBoard` fails | No task is created; 500 with a generic localized error. |
| Agent launch fails after task creation | Task remains with the launch error; owner retries from the task. |
| Owner adds more plans later | Counts update on the next pass or endpoint read; a new adaptation task can start once the previous one is no longer active. |

## Security

The endpoint uses the repository's workspace authorization. The agent receives
only the repository path and plan directories; repository-derived strings are
tag-stripped before entering the prompt. File contents are never logged.

## Feature flag

Plan adaptation lives under `features.planFiles`. No additional flag.

## Observability

- The unadapted count is visible in the pass status; no separate metric.
- `repository_maintenance_task_total{kind,outcome}` with outcome `created`,
  `existing`, `rejected`; also logged as `repository_maintenance.metric.*`.
  No repository, workspace, or task identifier is a label.

## Related decisions

- [ADR 2026-10-05 repository maintenance agent tasks](../../../decisions/2026-10-05-repository-maintenance-agent-tasks.md)
- [ADR 2026-10-04 repository plan files](../../../decisions/2026-10-04-repository-plan-files-source-of-truth.md)
