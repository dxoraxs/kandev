---
status: current
system: tasks
requirements:
  - REQ-TASKS-PLAN-FILES-001
  - REQ-TASKS-PLAN-FILES-002
  - REQ-TASKS-PLAN-FILES-003
  - REQ-TASKS-PLAN-FILES-004
  - REQ-TASKS-PLAN-FILES-005
  - REQ-TASKS-PLAN-FILES-006
---

# Repository Plan Files System Design

## Purpose and boundaries

A backend package, `internal/planfiles`, reconciles plan files in local
repositories with tasks on a per-workspace plan board, and writes board edits
back into the files. It is a client of task-system services that already own
task creation, external-ID idempotency, moves, reordering, and archival. It
adds no new task semantics.

Contracts used but not owned here:

- Task external identifiers and idempotent creation:
  [External ID idempotency](external-id-idempotency.md).
- Within-step ordering and bands:
  [Kanban task reordering](kanban-task-reordering.md).
- Repository registration and `local_path`:
  [Local repositories](../../workspaces/system-design/local-repositories.md).
- The workspace-scoped sync shape (config row, poller, force sync, status
  banner) follows `internal/workflowsync`, which syncs workflow definitions,
  not tasks.

The plan file stays authoritative. See
[ADR 2026-10-04 repository plan files](../../../decisions/2026-10-04-repository-plan-files-source-of-truth.md).

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-TASKS-PLAN-FILES-001` | [Plan file format](#plan-file-format) |
| `REQ-TASKS-PLAN-FILES-002` | [Sync pass](#sync-pass), [Task projection](#task-projection) |
| `REQ-TASKS-PLAN-FILES-003` | [Write-back](#write-back) |
| `REQ-TASKS-PLAN-FILES-004` | [External identifiers and adoption](#external-identifiers-and-adoption) |
| `REQ-TASKS-PLAN-FILES-005` | [HTTP API](#http-api), [Frontend](#frontend), [Feature flag](#feature-flag) |
| `REQ-TASKS-PLAN-FILES-006` | [Scanner and file safety](#scanner-and-file-safety) |

## Components and responsibilities

| Component | Responsibility |
| --- | --- |
| `planfiles/format` | Pure parse of frontmatter and body; byte-preserving key edits. No I/O. |
| `planfiles/scan` | Lists and reads plan files of one repository under the safety rules. |
| `planfiles.Store` | Owns `plan_file_configs` and `plan_file_tasks`. |
| `planfiles.Service` | Config CRUD with workspace authorization, sync pass, write-back, status. |
| `planfiles.Poller` | 60-second ticker calling `Service.SyncDueWorkspaces`. |
| `planfiles.WriteBackSubscriber` | Event-bus subscriber for `task.moved`, `task.reordered`, `task.updated`. |
| `planfiles.Handlers` | HTTP routes under `/api/v1/plan-files`. |
| `config/workflows/plans.yml` | Built-in Plans workflow template. |
| Web `PlanFilesSection` | Settings section on the workspace Workflows tab. |

## Plan file format

A plan file begins with `---` on line 1 and a closing `---` line. The block is
parsed with `gopkg.in/yaml.v3` into a mapping. Recognised keys:

| Key | Type | Rule |
| --- | --- | --- |
| `board` | string | Required. Closed set from the requirement; compared lower-case, trimmed. |
| `title` | string | Optional; else first `# ` heading; else file stem. |
| `priority` | string | `models.TaskPriority*` values; default `medium`. |
| `order` | number | Optional float64. |
| `executor` | string | Optional; trimmed, at most 40 bytes. |
| `depends_on` | list of strings | File names in the same directory. |
| `external_id` | string | Validated by `service.NormalizeExternalID` (`ExternalIDMaxBytes`). |

A block that is not valid YAML, or a frontmatter without `board`, makes the file
a non-plan file (ignored). A valid block with a bad value yields a `PlanFile`
with `ParseErrors` populated and the invalid field left at its zero value.

`format.SetKeys(content []byte, values map[string]string) ([]byte, error)`
edits only top-level lines of the frontmatter block that match `^<key>:`. It
replaces the value text after the colon, preserving the key spelling, the
original line ending, and any trailing `# comment`. A missing key is inserted
as a new line immediately before the closing `---`. A key whose current value
is a block scalar, a flow collection, or spans several lines is refused with
`ErrUnsupportedValueShape`; the write-back then reports a failed write instead
of reformatting. Values written are plain scalars produced by the service
(`in_progress`, `12.5`, `high`), so no quoting is required.

The content hash is SHA-256 over the full file bytes.

## Scanner and file safety

For each repository returned by `task/service.Service.ListRepositories` with
`SourceType == "local"` and a non-empty `LocalPath`:

1. Resolve the root with `filepath.EvalSymlinks`. A missing root records a
   repository error and skips the repository.
2. For each configured directory (relative, cleaned, rejected if absolute or
   containing `..`), `os.ReadDir` the directory. Missing directories are
   silently skipped.
3. Consider entries ending in `.md`, sorted by name, stopping after
   `maxFilesPerDirectory = 1000` with a `truncated` warning.
4. `os.Lstat` each entry. Anything not a regular file, including a symlink, is
   reported as `not_regular_file`. A size above `maxPlanFileBytes = 1 << 20` is
   reported as `too_large` without reading.
5. Read the file and confirm that `EvalSymlinks` of its directory is still
   inside the resolved root before use.

Writes use the same containment checks, then write a temporary file in the same
directory, `fsync`, and `os.Rename` over the target, keeping the original file
mode.

## Data and contracts

### Tables

`plan_file_configs`, one row per workspace:

| Column | Notes |
| --- | --- |
| `workspace_id` TEXT PK | |
| `enabled` INTEGER | |
| `workflow_id` TEXT | Plan board. |
| `status_steps` TEXT | JSON object, board status to step ID, all statuses except `hidden`. |
| `directories` TEXT | JSON array, default `["docs/plans","docs/superpowers/plans"]`. |
| `last_pass_at`, `last_pass_ok`, `last_counts`, `last_file_errors` | Status for the settings section; `last_file_errors` capped at 200 rows. |
| `created_at`, `updated_at` | |

`plan_file_tasks`, one row per plan task:

| Column | Notes |
| --- | --- |
| `task_id` TEXT PK | |
| `workspace_id` TEXT | Indexed. |
| `repository_id` TEXT, `rel_path` TEXT | Unique per workspace together. |
| `external_id` TEXT | Unique per workspace. |
| `content_hash` TEXT | Hash of the bytes last read or written. |
| `synced_step_id`, `synced_priority`, `synced_order_key` | Task state the last pass applied; divergence marks a board edit. |
| `notice` TEXT | Board-edit-not-saved notice; cleared when the file next changes. |
| `last_seen_at` | |

Both tables are registered in `persistence/requiredstores/catalog.go`, the
store conformance actions, the workspace-deletion table registry (delete by
`workspace_id`), and `backendapp/e2e_reset.go`.

### Built-in template

`apps/backend/config/workflows/plans.yml` (id `plans`) has steps `queue`,
`hand-to-agent`, `in-progress`, `waiting-owner`, `waiting-external`,
`deferred`, `done`. Only `hand-to-agent` has `on_enter: auto_start_agent`; its
prompt tells the agent that the description header names the plan file, to
work from that file in the repository's main checkout, and to keep its `board`
key current; it ends with `{{task_prompt}}`. `done` sets
`complete_task_on_enter`. No step has turn-driven transitions. The template's
step IDs map to statuses through a fixed table in `planfiles`, used when the
owner creates a board from the template. Owners can duplicate the handoff step
per agent in the existing workflow editor.

### HTTP API

All routes take `workspace_id` and are added to `integrationWorkspacePrefixes`
in `backendapp/helpers.go`. `planfiles.Service` also checks workspace scope
itself (`SetWorkspaceAuthorizer`), matching the workflow-sync authorization
pattern; an unscoped internal context is allowed.

| Route | Behavior |
| --- | --- |
| `GET /api/v1/plan-files/config` | Config plus last pass status; 404 when absent. |
| `PUT /api/v1/plan-files/config` | Validates workflow ownership, mapping completeness, step membership, directory paths. |
| `POST /api/v1/plan-files/board` | Creates a workflow from template `plans`, returns it with a filled mapping. |
| `POST /api/v1/plan-files/sync` | Runs one pass now; returns the pass summary. 409 while a pass holds the workspace lock. |

## Control flow

### Sync pass

`Service.SyncWorkspace(ctx, workspaceID)` holds a per-workspace mutex that the
write-back also takes, so a pass and a write never interleave.

1. Load config; return when disabled or the workflow is missing (recorded as a
   pass error).
2. Scan every local repository; parse every file into `PlanFile` or a file
   error.
3. Resolve external identifiers; detect duplicates (both files fail).
4. For each plan file with a visible status:
   - find the `plan_file_tasks` row, else `GetTaskByExternalID` (adoption),
     else `CreateTask` with `ExternalID`, `WorkflowStepID`, `Repositories`,
     `StartAgent: false`;
   - unarchive through `HandoffService.UnarchiveTaskTree` when archived;
   - compute the desired title, description, priority, and step (see
     [Task projection](#task-projection) and [Write-back](#write-back) for
     the handoff rule);
   - when the task diverges from its `synced_*` values (a board edit the
     write-back has not handled yet): if the file hash equals `content_hash`,
     apply the shared write-back to the file and keep the task as it is; if
     the hash changed, the file wins, the task is restored, and `notice` is
     set;
   - when the content hash and the applied state are unchanged, do nothing;
   - else `UpdateTask` for title, description, and priority (merging existing
     metadata), and `MoveTaskWithOptions` with actor `system` when the step
     differs, unless a session of the task is `STARTING` or `RUNNING`.
5. Apply ordering per step: when the plan tasks of a step, in their current
   board order, differ from the desired order, call `ReorderStepTasks` for the
   `admitted` band with the desired plan-task order, keeping non-plan tasks at
   their relative positions. When the board order differs only because of
   an unhandled reorder, write the board order back instead with the shared
   order routine. A step counts as an unhandled reorder only when none of its
   plan tasks was created, adopted, unarchived, or moved by this pass, none
   of their files changed, and every task is still at its `synced_step_id`;
   otherwise the pass reorders the board to the files.
6. Archive (`HandoffService.ArchiveTaskTree`, no cascade) plan tasks whose file
   is gone or `hidden`; keep their row so a returning file unarchives.
7. Store hashes, `synced_*` values, and the pass summary; increment metrics.

`Poller` ticks every 60 seconds and runs `SyncDueWorkspaces`, which visits
every enabled config, satisfying the 90-second bound.

### Task projection

- Title: `[executor] ` prefix rule, truncated to `contract.TaskTitleMaxLength`
  runes with `…`.
- Description: a block-quoted header (repository name, `rel_path`, executor,
  dependency links to existing plan tasks in the web task route format, parse
  errors, notice) followed by the body after the frontmatter. The body is
  capped at 16 KiB (`maxDescriptionBodyBytes`), cut on a line break near the
  limit or else on a rune boundary, with a `Truncated:` header line, because
  task descriptions travel in board list and boot payloads.
- Desired order key: `(has_order ? 0 : 1, order, rel_path)`.
- External repository link: `Repositories` holds the plan's repository so a
  handoff agent gets a worktree of it.

### Write-back

`WriteBackSubscriber` subscribes with `eventBus.Subscribe` to `task.moved`,
`task.reordered`, and `task.updated`. Bus delivery is synchronous to the
publisher, and a sync pass publishes while it holds the workspace mutex, so the
handlers only enqueue a deduplicated task or step identifier. One owned worker
goroutine drains the queue. For each affected task that has a
`plan_file_tasks` row:

1. Take the workspace mutex and reload the task and row; event payloads only
   locate the work.
2. Build the intended key changes from divergence only:
   - step differs from `synced_step_id` and the new step is mapped to a status
     different from the file's: set `board`;
   - step is unmapped (handoff): no write; record the step as synced;
   - priority differs from `synced_priority`: set `priority`;
   - for `task.reordered`, when the plan-task order of the step is no longer
     sorted by order key: run the shared order routine on the board sequence.
     It keeps the existing keys of the longest run of tasks that is already
     sorted and gives new `order` values only to the others (midpoint between
     kept neighbours, or neighbour ± 1 at an edge). A single moved task
     between two ordered neighbours therefore changes only its own file; when
     a kept neighbour above it lacks `order`, the tasks from the top of the
     step through the moved task receive `10, 20, …`.
3. Re-read each target file; if its hash differs from `content_hash`, skip the
   write, set `notice` on the row, and count a conflict. The next pass shows
   the notice in the description and restores the task. A failed write in a
   pass is reported as file error `write_failed` and restores the task.
4. Apply `format.SetKeys`, write atomically, store the new hash and `synced_*`
   values.

Because sync-applied moves, priorities, and orders equal the `synced_*` values
and the file, the subscriber produces no write for them, which prevents loops.
The handoff rule in the pass: a task in an unmapped step stays while the file
status is `queued` or `in_progress`; otherwise it moves to the mapped step.

### External identifiers and adoption

The default identifier is `plan-file:<repository_id>:<rel_path>` using `/`
separators. A frontmatter `external_id` overrides it, which keeps a task across
file renames and lets a workspace adopt tasks that another tool created with
its own identifiers. Adoption moves the task onto the plan board with actor
`system` and inserts the `plan_file_tasks` row.

## Failure and recovery

| Situation | Behavior |
| --- | --- |
| Repository root missing | Repository error row; other repositories continue. |
| File unreadable, too large, not regular | File error row; task untouched. |
| Task service error for one file | File error row with the error code; pass continues. |
| Write conflict | No write; `notice`; next pass restores the task. |
| Write I/O failure or unsupported value shape | File error `write_failed`; the pass restores the task. |
| Backend restart | Stateless pass; rows and hashes persist, next tick resumes. |
| Pass already running | Force sync returns 409; poller skips the workspace. |

## Security

The package reads and writes only inside resolved roots of local repositories
that the workspace owner registered, under the rules in
[Scanner and file safety](#scanner-and-file-safety). Plan content is untrusted
text: it is stored as task description Markdown and rendered by the existing
sanitized Markdown renderer. File contents are never written to logs; errors
log the repository ID, relative path, and reason code only.

## Feature flag

Runtime flag `features.planFiles` (`KANDEV_FEATURES_PLAN_FILES`), registered in
`runtimeflags/registry.go`, restart required. `profiles.yaml`: `prod`, `dev`,
and `e2e` all false, as for every release toggle; E2E specs enable it through
the `KANDEV_FEATURES_PLAN_FILES` environment variable of their backend
fixture, and an installation enables it in Settings, System, Feature Toggles.
When off, `services.PlanFiles` stays nil, so the poller,
subscriber, and routes are not created, and the web `useFeature("planFiles")`
hides the section.

## Frontend

`apps/web/components/settings/plan-files-section.tsx` renders inside
`app/settings/workspace/workspace-workflows-client.tsx` below the workflow sync
section, using the same card, status-banner, and dialog primitives as
`workflow-sync-section.tsx`. API client:
`apps/web/lib/api/domains/plan-files-api.ts`. It shows the enable switch, board
select with "Create Plans board", the status-to-step mapping table, the
directory list editor, last pass summary with per-file error rows, and Sync
now. On phones the mapping rows stack label over select, and the error list
becomes a vertical list. All copy goes through `t()` in seven locales.

## Observability

Expvar counters, also logged as `plan_files.metric.*`:

- `plan_files_pass_total{outcome}` with `ok`, `partial`, `failed`.
- `plan_files_file_errors_total{reason}` with `parse`, `too_large`,
  `not_regular_file`, `unreadable`, `duplicate_external_id`, `task_service`.
- `plan_files_writeback_total{outcome}` with `written`, `conflict`, `failed`,
  `handoff_skipped`.

No workspace, task, repository, or path value is a label.

## Related decisions

- [ADR 2026-10-04 repository plan files](../../../decisions/2026-10-04-repository-plan-files-source-of-truth.md)
