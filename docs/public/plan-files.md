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
| `executor`    | No       | Who works on the plan. Shown as a `[executor]` title prefix until the plan is done.         |
| `depends_on`  | No       | A list of plan file names in the same directory. Shown as links; does not block launch.     |
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
| `done`             | Finished. The executor prefix is dropped.      |
| `hidden`           | Not shown. The task is archived.               |

Each status except `hidden` maps to one step of the plan board. Column names are workflow data, so you can name and localize them freely.

## How the board follows the files

- Kandev polls every enabled workspace once a minute, so a created or changed file shows up within 90 seconds, including uncommitted changes. **Sync now** runs a pass immediately.
- Each plan file becomes one task in the mapped step. The description starts with a header that names the repository, the file, the executor, dependency links, and parse errors, followed by the plan body. Bodies longer than 16 KiB are cut, and the header says the full plan is in the file.
- Within a step, plan tasks are ordered by `order`; plans without `order` follow in file path order.
- Deleting a file, moving it out of the scanned directories, or setting `hidden` archives its task. When the file returns, the same task is unarchived.
- Kandev does not move a task while its agent turn is starting or running; the move is applied on a later pass.

## Board edits write back

Moving a plan task to a mapped column writes the new `board` value into the file. Changing the priority writes `priority`, and reordering writes `order` values that reproduce the new order on the next pass.

- A write changes only the keys it sets. The body, other keys, key order, and comments stay byte for byte the same.
- Writes stay uncommitted in the working tree. Commit them yourself or let an agent do it.
- If the file changed on disk since Kandev last read it, the write is skipped. The board follows the file on the next pass, and the task shows a notice that the board edit was not saved.

### Handoff steps

A plan board step that no status maps to is a handoff step. The built-in Plans template has one, **Hand to agent**, which starts an agent on entry.

- Moving a plan into a handoff step writes nothing to the file. The agent that starts there updates `board` itself.
- While the file says `queued` or `in_progress`, the task stays in the handoff step. Any other status moves it to its mapped step.

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

Turning sync off stops scanning and writing and leaves existing plan tasks unchanged. Kandev never commits, pushes, or creates plan files.
