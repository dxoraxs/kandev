---
title: "Plan Files"
description: "Show Markdown plan files from your local repositories as tasks on a workflow board, and write board edits back into the files."
---

# Plan Files

Plan files turn implementation plans kept as Markdown files in your repositories into tasks on one workflow board. The file stays the source of truth: Kandev keeps the task current while the file changes and writes status, priority, and order edits from the board back into the file.

This page is a reference for the file format and the sync rules. Plan files are experimental and off by default.

## Quick path

1. Enable `features.planFiles` in **Settings > System > Feature Toggles** and restart Kandev.
2. Open **Settings > Workspaces > _workspace_ > Workflows** and turn on **Plan files**.
3. Click **Create Plans board**, or pick an existing workflow and assign a step to every status.
4. Add a `board` key to the frontmatter of each plan you want on the board, then click **Sync now**.

```md
---
board: queued
priority: high
executor: codex
---

# Add offline mode

Steps of the plan...
```

## Which files are plan files

- Kandev scans every repository of the workspace whose source is a local path. Remote repositories are not scanned.
- The scanned directories default to `docs/plans` and `docs/superpowers/plans`, relative to the repository root. You can change the list in the settings section.
- Only `*.md` files directly inside a scanned directory are read.
- A file is a plan file only when its YAML frontmatter has a `board` key. Other files are ignored and never produce a task or an error.

## Frontmatter keys

| Key           | Required | Value                                                                                       |
| ------------- | -------- | ------------------------------------------------------------------------------------------- |
| `board`       | Yes      | One of the statuses below. Compared case-insensitively after trimming.                       |
| `title`       | No       | Task title. Defaults to the first `# ` heading of the body, then to the file name.          |
| `priority`    | No       | `critical`, `high`, `medium` (default), or `low`.                                           |
| `order`       | No       | A decimal number. Lower values come first within a column.                                  |
| `executor`    | No       | Who works on the plan. Shown on the card as an agent executor until the plan is done.       |
| `date`        | No       | A calendar date in `YYYY-MM-DD` form, such as when to check an answer or a deadline. Shown on the card until the plan is done. An invalid date is reported in the description header and ignored. |
| `depends_on`  | No       | A list of plan file names in the same directory. Shown as links and turned into task dependencies (see [Dependencies](#dependencies)). |
| `tracks`      | No       | A list of at most 20 paths relative to the repository root. Files and directories whose task lists count toward the plan's progress (see [Progress and card flags](#progress-and-card-flags)). |
| `external_id` | No       | Task identifier, at most 255 bytes. Use it to keep a task across renames or adopt an existing task. |

Unknown keys are preserved and ignored. An invalid value does not hide the plan: the task stays where it was (a new task goes to the `queued` step) and the description header shows the parse error.

## Statuses

| Status             | Meaning                                        |
| ------------------ | ---------------------------------------------- |
| `queued`           | Planned, not started.                          |
| `in_progress`      | Someone is working on it.                      |
| `waiting_owner`    | Blocked on the workspace owner.                |
| `waiting_external` | Blocked on someone or something outside.       |
| `deferred`         | Postponed.                                     |
| `done`             | Finished. The card shows no date or executor.  |
| `hidden`           | Not shown. The task is archived.               |

Each status except `hidden` maps to one step of the plan board. Column names are workflow data, so you can name and localize them freely.

## How the board follows the files

- Kandev polls every enabled workspace once a minute, so a created or changed file shows up within 90 seconds, including uncommitted changes. **Sync now** runs a pass immediately.
- Each plan file becomes one task in the mapped step. The description starts with a header that names the repository, the file, the executor, dependency links, and parse errors, followed by the plan body. Bodies longer than 16 KiB are cut, and the header says the full plan is in the file.
- The `date` and `executor` keys become card facts, not title text. The date reads as a waiting date for `waiting_owner` and `waiting_external`, a deferral date for `deferred`, and a due date otherwise. Done plans carry no facts.
- Within a step, plan tasks are ordered by `order`; plans without `order` follow in file path order.
- Deleting a file, moving it out of the scanned directories, or setting `hidden` archives its task. When the file returns, the same task is unarchived.
- Kandev does not move a task while its agent turn is starting or running; the move is applied on a later pass.

## Board edits write back

Moving a plan task to a mapped column writes the new `board` value into the file. Changing the priority writes `priority`, and reordering writes `order` values that reproduce the new order on the next pass.

- A write changes only the keys it sets. The body, other keys, key order, and comments stay byte for byte the same.
- Writes stay uncommitted in the working tree. Commit them yourself, let an agent do it, or use **Commit plan files** in the settings (see [Committing plan files](#committing-plan-files)).
- If the file changed on disk since Kandev last read it, the write is skipped. The board follows the file on the next pass, and the task shows a notice that the board edit was not saved.

### Handoff steps

A plan board step that no status maps to is a handoff step. The built-in Plans template has one, **Hand to agent**, which starts an agent on entry.

- Moving a plan into a handoff step writes nothing to the file. The agent that starts there updates `board` itself.
- While the file says `queued` or `in_progress`, the task stays in the handoff step. Any other status moves it to its mapped step.

### Executor columns

A step that no status maps to can also carry an executor name. In **Settings > Workspaces > _workspace_ > Workflows**, the **Executor columns** list pairs a step with a name of 1 to 40 bytes. A step has at most one name, a name belongs to at most one step, and a step that a status maps to cannot be an executor column.

- Moving a plan into an executor column writes that name to the `executor` key and leaves `board` alone, as long as the file says `queued` or `in_progress`.
- If the file says `in_progress` and `executor` names a different executor, nothing is written, the task moves back to the step of its status, and the task shows a notice that the plan is already taken.
- A sync pass never moves a task into an executor column, and moves a task out of one when the file's `executor` no longer equals the column's name.
- The claim conflict cannot prevent an `on_enter` auto-start of the column from firing before the task moves back. Keep auto-start off on executor columns that can be contested.

## Owner decisions

A plan task in the step mapped to `waiting_owner` shows a decision bar on its task page with **Return** and **Accept** (and **Accept, back to queue** in the menu next to Accept). Other tasks never show it.

- **Accept** takes an optional comment and writes `board: done` (or `board: queued` for **Accept, back to queue**). The note is `- <YYYY-MM-DD> accepted: <comment>`; without a comment the colon and comment are left out.
- **Return** needs a comment and writes `board: queued` with the note `- <YYYY-MM-DD> returned: <comment>`.
- A comment is one line of at most 500 characters; line breaks become spaces.
- The note is appended as the last line of the section headed `## Owner notes` (change the heading with **Notes section heading**). A file without that section gets it at the end. Every other byte of the file stays the same.
- If the file changed on disk after Kandev last read it, the action fails without writing and tells you the file changed. After a successful action the task is already in the step of its new status.

On a phone the same actions sit in the task's description view as full-width buttons, and the comment is entered in a bottom drawer.

## Waiting for owner page

**Waiting for owner** in the navigation (the navigation sheet on a phone, at `/plans/waiting`) lists the plan tasks in the `waiting_owner` step across every workspace you can access that has plan files enabled. Each row shows the workspace, title, repository, date, and executor, ordered by date with undated plans last. The entry shows the number of listed plans and is hidden while `features.planFiles` is off. Selecting a row opens the task. A workspace that cannot be read is named on the page and does not hide the others. On a phone each plan is one full-width row.

## Dates that act

When **Move plans to Waiting for you when their date arrives** is on (the default), a plan whose status is `waiting_external` or `deferred` and whose `date` is today or earlier in the server's local time zone is moved to `waiting_owner` by the next sync pass. The file gets `board: waiting_owner` and the note `- <YYYY-MM-DD> date reached: was <previous status>`. A plan in any other status is never changed, whatever its date.

Each wake-up sends a `plan_file.date_reached` notification, with the plan title, through the notification providers that subscribe to that event. Default notification providers do not subscribe to `plan_file.date_reached` until the owner enables it in **Settings > Notifications**. A failed notification never undoes the wake-up.

The kanban sort picker offers a date sort: tasks of a column ordered by date ascending, undated tasks after dated ones in board order.

## Dependencies

Each `depends_on` entry that names a plan file of the same directory becomes a task dependency of the plan task. After every pass the plan task's dependencies on other plan tasks equal the resolvable entries; dependencies on tasks that are not plan tasks are left alone.

- A dependency resolves only when the predecessor task is completed. A blocked plan task that enters a step with agent auto-start does not start an agent.
- An entry that names no plan file, or a set of entries that task dependencies reject (a cycle or a self-reference), is reported as a file error for that plan and leaves its existing dependencies unchanged.

## Progress and card flags

Progress counts Markdown task-list items (`- [ ]`, `* [ ]` open; `[x]`, `[X]` done) outside fenced code blocks: the items of the plan body, of each file listed in `tracks`, and one item per `task-*.md` file directly inside a tracked directory (done when its frontmatter `status` is `done`). A `tracks` entry that leaves the repository, is a symbolic link, or does not exist is reported as a file error and skipped. The card shows `done/total` while the plan is not done.

Cards also show warning tags after the other hints:

| Tag           | When                                                                                                                         |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `Open items`  | The status is `done` and a counted item is still open.                                                                      |
| `Stale`       | The status is `in_progress`, no agent turn is starting or running, and neither the file nor a tracked item changed within **Mark in-progress plans as stale after** days (default 7, 0 turns it off). |
| `Uncommitted` | The repository is a git working tree and the plan file is modified, staged, or untracked. A deleted plan file is not listed as uncommitted. |

## Plan index

When **Index file name** is set (a `*.md` file name without a path, for example `INDEX.md`), every scanned directory that holds a plan file gets that file, listing each visible plan once, grouped by status in board order, with the title as a link, the priority, the executor, and the date. The file starts with a marker line; Kandev overwrites only a file that starts with the marker or does not exist, and reports any other file of that name without touching it. The index has no timestamp, is not rewritten when its content would not change, and is never counted as a plan file.

## Creating a plan from the board

On the plan board, **New plan** (in the board display options on a phone) asks for a local repository, one of the scanned directories, a title, an optional file name, a priority, an optional executor, and an optional body. The file name defaults to a slug of the title's ASCII letters and digits (`plan-<YYYYMMDD-HHMM>` when there are none), must match `[A-Za-z0-9][A-Za-z0-9._-]*\.md`, and is at most 120 bytes. The file starts with `board: queued`, the title, the priority, and the executor, followed by a level-one heading and the body. An existing file is never replaced. The card is on the board when the action reports success. On a phone the form is a full-height drawer with a sticky **Create** button.

## Committing plan files

In **Settings > Workspaces > _workspace_ > Workflows**, the **Uncommitted plan files** block lists each local repository with the number of plan files and plan indexes that are modified, staged, or untracked. When the number is not zero, **Commit plan files** opens a dialog with the file list and an editable message (default `docs(plans): update plan files`).

The action creates one commit on the current branch that holds exactly those files, even if other changes were already staged. It never pushes. During a merge, rebase, or cherry-pick, or when the commit fails (for example a hook rejects it), the working tree and index stay as they were and the dialog shows the reason with the last lines of the git output.

## Adopting existing tasks

When a workspace already tracks plans as tasks, set `external_id` in each plan file to the identifier of its existing task. Kandev adopts that task, including an archived one, moves it onto the plan board, and keeps its history and sessions. Without `external_id`, the identifier is derived from the repository and the file path.

If two plan files resolve to the same identifier, neither is synced, and both are reported as a conflict.

## Adapting existing plans

Plan files whose header is not in the board format do not appear on the board. The **Plan files** section lists each local repository that has some, with the number of files and an **Adapt with agent** button. The same count shows in the status line as "not adapted". Adding a repository that has such files also offers the adaptation once; choose **Not now** to skip it.

**Adapt with agent** creates a task in the workspace's default agent profile and opens it. The agent reads the repository history, sets each plan's status, and commits only the plan files, working in the repository's checkout. Starting it again while that task is still active opens the same task instead of creating another.

- The workspace needs a default agent profile and at least one workflow besides the plan board. Otherwise the button reports what is missing.
- If plan sync is not set up yet, starting the adaptation also creates the Plans board and turns sync on.
- Only repositories on this machine are listed.

## Limits and errors

- Only regular files inside the repository root are read and written. A symlinked plan file, or a scanned directory that resolves outside the root, is skipped and reported.
- Files larger than 1 MiB are skipped. A directory with more than 1,000 Markdown files is read up to that count and reported as truncated.
- A missing repository path, an unreadable file, or a failed write is reported for that repository or file and does not stop the rest of the pass.
- The settings section shows the time and outcome of the last pass, counts of created, updated, moved, archived, and failed tasks, and one row per failed file.

Turning sync off stops scanning and writing and leaves existing plan tasks unchanged. Kandev commits plan files only when you use **Commit plan files**, never pushes, and creates plan files only through **New plan** and the generated index.
