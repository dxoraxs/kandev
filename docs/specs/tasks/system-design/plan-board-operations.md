---
status: current
system: tasks
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-001
  - REQ-TASKS-PLAN-BOARD-OPS-002
  - REQ-TASKS-PLAN-BOARD-OPS-003
  - REQ-TASKS-PLAN-BOARD-OPS-004
  - REQ-TASKS-PLAN-BOARD-OPS-005
  - REQ-TASKS-PLAN-BOARD-OPS-006
  - REQ-TASKS-PLAN-BOARD-OPS-007
  - REQ-TASKS-PLAN-BOARD-OPS-008
  - REQ-TASKS-PLAN-BOARD-OPS-009
---

# Plan Board Operations System Design

## Purpose and boundaries

This design extends `internal/planfiles`
([Repository plan files](repository-plan-files.md)) with operations that the
board performs on plan files. The package stays a client of task-system
services; it adds no task semantics. Everything here runs only when the
existing `features.planFiles` flag is on and the workspace config is enabled.
No new runtime flag is introduced.

Contracts used but not owned here:

- The `date` key, the `card_display` metadata merge, and the summary section:
  [Plan file card facts](plan-file-card-facts.md). This design adds facts to
  the same merge and must be implemented after it.
- Card rendering: [Kanban card display hints](kanban-card-display-hints.md).
  This design adds the `flags` field to that contract.
- Blocking and the auto-start gate: [Task dependencies](task-dependencies.md)
  (`Service.AddDependency`, `Service.RemoveDependency`,
  `dependencyBlocksAutoStart`).
- Notification providers: `internal/notifications/service`.
- Git execution: `internal/common/subproc` (`NewGitCommand`,
  `RunGitOutputClass`, work class `GitBackground` for passes and
  `GitInteractive` for the commit action).

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-TASKS-PLAN-BOARD-OPS-001` | [Executor steps](#executor-steps) |
| `REQ-TASKS-PLAN-BOARD-OPS-002` | [Owner notes](#owner-notes), [Owner decision](#owner-decision), [Frontend](#frontend) |
| `REQ-TASKS-PLAN-BOARD-OPS-003` | [Date wake-up](#date-wake-up), [Frontend](#frontend) |
| `REQ-TASKS-PLAN-BOARD-OPS-004` | [Dependencies](#dependencies) |
| `REQ-TASKS-PLAN-BOARD-OPS-005` | [Format additions](#format-additions), [Card facts](#card-facts), [Frontend](#frontend) |
| `REQ-TASKS-PLAN-BOARD-OPS-006` | [Plan index](#plan-index) |
| `REQ-TASKS-PLAN-BOARD-OPS-007` | [Plan creation](#plan-creation), [Frontend](#frontend) |
| `REQ-TASKS-PLAN-BOARD-OPS-008` | [Git state](#git-state), [Frontend](#frontend) |
| `REQ-TASKS-PLAN-BOARD-OPS-009` | [Waiting-owner list](#waiting-owner-list), [Frontend](#frontend) |

## Format additions

All in `internal/planfiles/format`, pure and byte-preserving.

- `PlanFile.Tracks []string`: the `tracks` key, a YAML sequence of at most 20
  non-empty scalars. More entries or a non-sequence value produce the parse
  error `invalid_tracks` and an empty list. Path safety is checked by the
  caller through `scan`, not by the parser.
- `CountItems(body string) (done, total int)` in `items.go`: counts list items
  whose marker is `-` or `*` followed by `[ ]`, `[x]`, or `[X]`, with any
  indentation, outside fenced code blocks. It reuses the fence tracking of the
  title fallback.
- `AppendNote(content []byte, heading, line string) ([]byte, error)` in
  `notes.go`: see [Owner notes](#owner-notes).
- `NewPlan(NewPlanInput) []byte` in `newplan.go`: frontmatter with `board`,
  `title`, `priority`, and optional `executor`, then `# <title>` and the body.
  The title and executor are emitted as YAML double-quoted scalars. The output
  must round-trip through `Parse` to the same values; a test asserts it.
- `WorkOrderDone(content []byte) bool`: true when the frontmatter `status`
  scalar equals `done`. Used for tracked directories.

## Data and contracts

### Tables

New columns on `plan_file_configs`:

| Column | Notes |
| --- | --- |
| `executor_steps` TEXT NOT NULL DEFAULT '{}' | JSON object, step ID to executor name. |
| `notes_heading` TEXT NOT NULL DEFAULT '' | Empty reads as `Owner notes`. At most 100 characters. |
| `wake_on_date` INTEGER NOT NULL DEFAULT 1 | |
| `stale_after_days` INTEGER NOT NULL DEFAULT 7 | 0 to 365. |
| `index_file` TEXT NOT NULL DEFAULT '' | Empty is off. |

New column on `plan_file_tasks`:

| Column | Notes |
| --- | --- |
| `synced_depends_on` TEXT NOT NULL DEFAULT '[]' | JSON array of the blocker task IDs the last pass set, sorted. |

Columns are added with `ALTER TABLE ... ADD COLUMN` when `db.TableColumns`
does not list them, the same way `summary_heading` is added. The store
conformance schema, owner actions, and the upgrade fixture manifest are
updated in the same change; `go run ./cmd/sqlguard ./internal` must pass.

### Config API

`Config` and `PutConfigRequest` gain `executor_steps`, `notes_heading`,
`wake_on_date`, `stale_after_days`, and `index_file`. In `PutConfigRequest`
they are pointers: an absent field keeps the stored value, so older clients
do not reset them. `validateMapping` rejects an executor step that is a value
of `status_steps`, is not a step of the workflow, or repeats a name
(`ErrInvalidConfig`, codes `invalid_executor_steps`, `invalid_notes_heading`,
`invalid_stale_after_days`, `invalid_index_file`). `index_file` must match
`[A-Za-z0-9][A-Za-z0-9._-]*\.md`.

### HTTP API

All under `/api/v1/plan-files`, authorized with the existing
`workspaceAuthorizer`.

| Route | Request | Response |
| --- | --- | --- |
| `POST /tasks/:taskId/decision` | `{action: "accept"\|"return", result?: "done"\|"queued", comment?}` | `200 {board}`; `404 not_plan_task`; `409 not_waiting_owner`; `409 file_changed`; `400 invalid_decision` |
| `POST /plans?workspace_id=` | `{repository_id, directory, title, file_name?, priority?, executor?, body?}` | `201 {task_id, repository_id, rel_path}`; `409 file_exists`; `400 invalid_plan`; `404 repository_not_found` |
| `GET /git-status?workspace_id=` | | `200 {repositories: [{repository_id, repository_name, files: [rel_path]}]}` |
| `POST /commit?workspace_id=` | `{repository_id, message?}` | `200 {commit, files}`; `409 repository_busy`; `409 nothing_to_commit`; `422 commit_failed {output}`; `404 repository_not_found`; `400 invalid_commit` |
| `GET /waiting-owner` | | `200 {items: [...], failed_workspaces: [{workspace_id, workspace_name}]}` |

A waiting-owner item is `{workspace_id, workspace_name, task_id, title,
repository_name, rel_path, date, executor, priority}`.

### Task metadata

The pass adds two facts to the `card_display` object through the metadata
compare-and-merge that [Plan file card facts](plan-file-card-facts.md)
introduces (`projectCardFacts`), sending the full map only when a fact
changed:

- `card_display.progress = {done, total}`.
- `card_display.flags = ["stale" | "open_items" | "uncommitted", ...]` in that
  fixed order; the key is omitted when no flag applies.

## Executor steps

`cfg.ExecutorSteps` maps a step ID to a name. A step in the map is a handoff
step in the sense of `staysInHandoff`.

- **Write-back** (`boardKeys`, `writeBoardEdit`): when the destination step is
  an executor step and the parsed file's board is `queued` or `in_progress`,
  the keys to set are `executor: <name>` only. The value is written quoted
  when it contains a character that a plain YAML scalar cannot carry. The row
  is saved with the new `SyncedStepID` as today.
- **Claim conflict:** when the board is `in_progress` and `pf.Executor` is
  non-empty and differs from the step's name (compared after trimming, case
  insensitively), no write happens; the task is moved to
  `cfg.StatusSteps[in_progress]` with the system actor and the row notice is
  set to `plan is already taken by <executor>`. The orchestrator may have
  reacted to the entry before the move back; see
  [Failure and recovery](#failure-and-recovery).
- **Pass** (`desiredStep`): a task in an executor step stays there only while
  the board is `queued` or `in_progress` and `pf.Executor` equals the step's
  name. Otherwise the desired step is the mapped status step. The pass never
  selects an executor step as a destination.

## Owner notes

`AppendNote` finds the notes section: the first line outside code fences that
is `## ` followed by the heading, compared with `strings.TrimSpace` and
`strings.EqualFold`. The section ends before the next `## ` line or at the end
of the file. The note is inserted after the last non-blank line of the
section. When the section is missing, the function appends a blank line (when
the file does not already end with one), `## <heading>`, a blank line, and the
note. The inserted text uses the line ending of the frontmatter's opening
fence. The function returns `ErrInvalidValue` for a note that contains a line
break. A fuzz test asserts that removing the inserted bytes restores the
input.

`composeEdit(content, keys, note)` in `planfiles` applies `SetKeys` and then
`AppendNote` and is the only caller that combines them, so one
compare-and-swap write (`scan.WriteFile` with the row hash) carries both.

## Owner decision

`Service.Decide(ctx, taskID, DecisionRequest)`:

1. Load the row by task ID (`404 not_plan_task`), authorize its workspace, and
   take the workspace lock.
2. Require `task.WorkflowStepID == cfg.StatusSteps[waiting_owner]`
   (`409 not_waiting_owner`).
3. `loadVerified` the file (`409 file_changed` on a hash mismatch, and the row
   notice is left untouched).
4. Normalize the comment (trim, line breaks to spaces, at most 500 characters).
5. `composeEdit` with `board` and the note, write, then move the task to the
   mapped step with the system actor and save the row with the new hash and
   `SyncedStepID`, so the write-back subscriber sees no divergence.

The handler lives in `handlers_decision.go`.

## Date wake-up

A step of `pass.run` after `resolveAll` and before `applyTracked`: for each
collected file with `cfg.WakeOnDate`, board `waiting_external` or `deferred`,
and `pf.Date <= today` (`time.Now().In(time.Local)` formatted `2006-01-02`;
the pass receives a clock for tests), the pass writes `board: waiting_owner`
plus the note through `composeEdit`, re-parses the written bytes, and replaces
the collected file, so the same pass moves the task and saves the new hash.
A failed write is reported as a file error (`wake_failed`) and retried by the
next pass.

After the pass releases the lock, each wake-up is sent to
`DateNotifier.NotifyPlanDateReached(ctx, workspaceID, title, relPath)`, an
interface of `planfiles` implemented by `notifications/service`
(`HandlePlanDateReached`, modeled on `HandleInboxItem`, event constant
`plan_file.date_reached` added to `AvailableEvents`). Errors are logged.
Metric: `plan_files_wake_total`.

## Dependencies

After `applyTracked`, for each tracked plan task the pass resolves `DependsOn`
names to the task IDs of plan files in the same repository and directory. A
name without a plan task yields the file error `unknown_dependency`. The
sorted ID list is compared with `row.SyncedDependsOn`; when equal nothing is
called. Otherwise the pass calls `RemoveDependency` for IDs only in the old
list and `AddDependency` for IDs only in the new list (two methods added to
`TaskAccess` and its fake), then saves the new list. An error from the
dependency service becomes the file error `invalid_dependency`; the row keeps
the entries whose change was refused, so the next pass retries only those. A
name that leaves the directory or is not a `.md` file name, and a
self-reference, are `invalid_dependency` without a call; a plan whose named
file could not be read this pass keeps its dependencies. `not found` on removal is ignored.

## Card facts

Computed in `projection.go` next to `projectCardFacts`:

- **Progress:** `CountItems(pf.Body)` plus tracked items. Tracked items are
  read through `scan` with the plan-file safety rules (inside the repository
  root, no symbolic links, 1 MiB per file, at most 200 `task-*.md` files per
  directory). Errors are file errors (`invalid_track`) and the item counts as
  zero. A `tracks` entry is an error when it is absolute, contains a `..`
  segment, does not exist, is a symbolic link (inside the repository or not),
  resolves outside the repository root, or is a file over 1 MiB. A tracked
  directory with more than 200 `task-*.md` files counts the first 200 and
  reports `truncated`; `task-*.md` entries that are links or not regular files
  are skipped.
- **`open_items`:** board `done` and `done < total`.
- **`stale`:** board `in_progress`, `cfg.StaleAfterDays > 0`, not
  `turnInFlight`, and the newest modification time among the plan file and
  its tracked items is older than the threshold. Modification times come from
  `os.Lstat` in the scanner (`scan.ScannedFile` gains `ModTime`). A tracked
  directory contributes its own modification time as well, so adding or
  removing a work order counts as a touch. The clock is `Service.currentTime`
  (`SetClock`), shared with owner notes and plan creation.
- **`uncommitted`:** see [Git state](#git-state).

## Plan index

`index.go` renders the index from pass state after `reorder`: for each
repository and scanned directory with at least one visible plan file, a
document that starts with the marker line
`<!-- Generated by Kandev plan files. Do not edit. -->`, then `# Plans`, then
one `## <status>` section per non-empty status in closed-set order with a
table `| Plan | Priority | Executor | Date |`. Rows follow the step order the
pass applied. Links are the file name, percent-encoded. Cell text escapes `|`.

`scan.WriteGenerated(root, relPath, content, marker)` writes with the same
containment checks as `scan.WriteFile`, creates the file when absent, replaces
it only when the existing bytes start with the marker, and returns
`ErrNotGenerated` otherwise (file error `index_not_owned`). The pass skips the
write when the bytes are equal. `collect` skips a file whose name equals
`cfg.IndexFile` before parsing, so it is neither a plan file nor unadapted.

## Plan creation

`Service.CreatePlan(ctx, workspaceID, CreatePlanRequest)` validates the
repository (local, in the workspace), the directory (one of
`cfg.Directories`), the file name, and the priority; derives the default file
name (`slug` keeps `[a-z0-9]`, joins runs with `-`, cuts at 80 bytes); and
writes with `scan.CreateFile(root, relPath, content)`, which fails with
`ErrExists` when the path exists and creates the directory when missing. It
then runs `SyncWorkspace` and returns the task ID of the row for the new path.

## Git state

Subpackage `internal/planfiles/gitstate`:

- `Dirty(ctx, root string, dirs []string) (map[string]struct{}, error)` runs
  `git status --porcelain=v1 -z --untracked-files=all -- <dirs>` with literal
  pathspecs and returns repository-relative paths. `ErrNotRepository` when
  `git rev-parse --is-inside-work-tree` fails; the pass treats it as an empty
  set.
- `Busy(ctx, root) bool`: `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD`,
  `rebase-merge`, or `rebase-apply` exists in the git directory
  (`git rev-parse --git-dir`).
- `CommitPaths(ctx, root, paths, message) (sha string, output string, err error)`:
  `git add -- <untracked paths>`, then
  `git commit --only -m <message> -- <paths>`; on failure
  `git reset -q -- <untracked paths>` restores the index. `--only` leaves
  other staged changes staged and out of the commit.

The pass calls `Dirty` once per repository and intersects the result with the
plan files and the index file. `Service.GitStatus` and `Service.CommitPlans`
recompute the set under the workspace lock; the commit action then runs a
pass so the flags clear.

## Waiting-owner list

`Service.WaitingOwner(ctx)` lists workspaces through a new `WorkspaceLister`
dependency (`ListWorkspaces`), keeps those the caller is authorized for and
whose config is enabled, and for each reads the rows whose `synced_step_id`
equals the `waiting_owner` step, then loads the tasks. A workspace whose
config, rows, or tasks fail to load is returned in `failed_workspaces`. Items
are sorted by `card_display.date` ascending, undated last, then by title.

## Frontend

- `lib/kanban/card-display.ts`: `flags` parser (closed set, order preserved,
  unknown values dropped); `KanbanCardHintRow` renders one warning pill per
  flag with `data-testid="kanban-card-flag-<name>"`.
- `hooks/domains/plans/use-plan-board.ts`: loads the workspace plan-files
  config once per workspace and answers whether a workflow is the plan board
  and which step a board status maps to. Plan-specific UI is derived from it
  (`task.workflowId === config.workflow_id`), so no task field is added.
- `lib/kanban/kanban-sort.ts` and `task-order.ts`: sort key `date_asc`; the
  backend allow-list `kanbanSortValues` gains the same value.
- `lib/api/domains/plan-files-api.ts`: clients for the five routes and the new
  config fields. `hooks/domains/plans/`: `usePlanDecision`, `useCreatePlan`,
  `usePlanGitStatus`, `useWaitingOwner`. Components never fetch directly.
- `components/task/plan-decision-bar.tsx`: shown in the task page under the
  top bar when the task is on the plan board in the step mapped to
  `waiting_owner`, whether or not the task has a session. A `not_plan_task`
  response hides the bar for that task. Desktop: inline bar with Accept, Return, and a comment
  dialog. Phone: a trigger in the task's description view that opens a bottom
  drawer.
- `components/kanban/new-plan-dialog.tsx`: opened from a New plan button in
  the kanban header when the active workflow is the plan board; on a phone
  from the board's display options, as a full-height surface.
- `components/settings/plan-files-*`: executor step rows, notes heading,
  wake-up toggle, stale threshold, index file name, and a git block with the
  per-repository count and Commit plan files.
- Page `/plans/waiting` (`SpaRoute` kind `plans-waiting`, modeled on the Needs
  You inbox route), a primary-nav entry with a count badge, and a phone
  navigation-sheet entry; all hidden when `useFeature("planFiles")` is false.
- Copy lives in `planFiles.json` and `kanban.json` in every locale.

## Failure and recovery

- Every file write is a hash compare-and-swap. A lost race reports
  `file_changed` or a file error and never overwrites.
- A crash between a file write and the task move is repaired by the next
  pass, which reads the file and moves the task.
- A claim conflict moves the task back after the step's `on_enter` actions
  may already have run. The move back does not stop a session that the
  orchestrator started; the notice tells the owner. Stopping that session is
  not part of this design.
- A failed `git status` leaves the previous `uncommitted` flags unchanged for
  that repository and is reported as a repository error (`git_status_failed`).
- A failed commit resets only the paths the action added to the index.

## Security

- All new file access goes through `scan` containment checks. Tracked paths,
  the index file, and created files cannot leave the repository root or
  follow symbolic links.
- Git commands receive paths as literal pathspecs after `--`, and the commit
  message as a single `-m` argument. No shell is involved.
- The waiting-owner list authorizes each workspace separately.
- Owner comments are written as one Markdown list line; line breaks are
  removed so a comment cannot add headings or frontmatter.

## Observability

Counters under `plan_files_*`: `plan_files_wake_total`,
`plan_files_decision_total` (label `action`), `plan_files_commit_total`
(label `outcome`: `ok`, `busy`, `failed`), `plan_files_index_total` (label
`outcome`: `written`, `not_owned`). No task, repository, or path is a label.

## Related decisions

- [ADR 2026-10-04 repository plan files](../../../decisions/2026-10-04-repository-plan-files-source-of-truth.md)
- [ADR 2026-10-05 plan board writes notes and commits](../../../decisions/2026-10-05-plan-board-writes-notes-and-commits.md)
