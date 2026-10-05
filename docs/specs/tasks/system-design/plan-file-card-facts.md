---
status: draft
system: tasks
requirements:
  - REQ-TASKS-PLAN-CARD-001
  - REQ-TASKS-PLAN-CARD-002
---

# Plan File Card Facts System Design

## Purpose and boundaries

This design extends the plan files sync of
[Repository plan files](repository-plan-files.md) with a `date` frontmatter
key, card facts written into task metadata, and an optional summary section.
It changes the projection of a plan file into a task and one configuration
field. It does not render anything on the card.

Contracts used but not owned here:

- Plan file format, scanner, sync pass, write-back, and config API:
  [Repository plan files](repository-plan-files.md).
- The `card_display` metadata object and its rendering:
  [Kanban card display hints](kanban-card-display-hints.md).
- Task updates: `taskservice.UpdateTask`, which replaces the whole metadata
  map when `Metadata` is set.

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-TASKS-PLAN-CARD-001` | [Date key](#date-key), [Card facts projection](#card-facts-projection), [Metadata write](#metadata-write) |
| `REQ-TASKS-PLAN-CARD-002` | [Summary heading](#summary-heading), [Summary projection](#summary-projection) |

## Components and responsibilities

| Component | Responsibility |
| --- | --- |
| `planfiles/format` | Parses and validates `date`. |
| `planfiles` projection | Derives card facts; drops the executor title prefix; selects the summary section. |
| `planfiles` sync pass | Compares and writes `card_display` with the other projected fields. |
| `planfiles` config | Stores `summary_heading`. |
| Web Plan files section | Edits the summary heading. |

## Data and contracts

### Date key

`format.PlanFile` gains `Date string` (normalized `YYYY-MM-DD`). Parsing uses
`time.Parse("2006-01-02", …)`; an invalid value adds a parse error with code
`invalid_date` and leaves `Date` empty. Write-back preserves the key like any
other unknown-to-writeback key.

### Card facts projection

`projectCardFacts(pf) map[string]any` returns nil or a map with:

| Key | Value | Condition |
| --- | --- | --- |
| `date` | `pf.Date` | `Date` set and board not `done` |
| `date_kind` | `waiting` (`waiting_owner`, `waiting_external`), `deferred` (`deferred`), `due` (other) | with `date` |
| `executor` | `{"name": pf.Executor, "kind": "agent"}` | `Executor` set and board not `done` |

`projectTitle` stops adding `[<executor>] `.

### Metadata write

`updateFields` compares `task.Metadata["card_display"]` with the projected map
using a canonical JSON comparison. When they differ, it copies
`task.Metadata`, sets or deletes `card_display`, and sends the full copy as
`UpdateTaskRequest.Metadata` together with any title, description, or priority
change. Task creation passes the same map in the create request. Equal facts
send nothing, so an unchanged file still produces no update and no event.

### Summary heading

`Config.SummaryHeading string`, stored in the existing plan files config row
(new column `summary_heading TEXT NOT NULL DEFAULT ''`, added at store
initialization with `ALTER TABLE … ADD COLUMN` when `db.TableColumns` does
not list it, as the GitLab store does). The config API accepts `summary_heading`; values are trimmed and
limited to 100 characters (`invalid_summary_heading` otherwise).

### Summary projection

`projectDescription` receives the heading. A line scanner that tracks code
fences finds level-two headings (`## `). When a heading matches after
`strings.TrimSpace` and `strings.EqualFold`, the body becomes the text before
the first level-two heading plus the matched section up to the next level-two
heading. The header gains the line `> Summary: the full plan is in the file.`
The existing 16 KiB cap applies to the selected text.

## Control flow

Sync pass, per plan file: parse (with `date`) → project title, description
(summary selection), priority, card facts → compare with the task → one
`UpdateTask` with the changed fields. Creation follows the same projection.

## Failure and recovery

| Situation | Behavior |
| --- | --- |
| Invalid `date` | Parse error in the header; no date fact; the task stays visible. |
| Summary heading not found | Full body, as before. |
| Metadata update fails | The pass reports the file error as for other update failures and retries next pass. |
| A concurrent writer changes other metadata keys between read and write | The pass reads the task in the same pass before writing; a lost concurrent key is restored by that writer's next write. Accepted: plan tasks have no other metadata writers in Kandev. |

## Frontend

The Plan files section gains a text input "Summary section heading" under the
directories list, 28px on desktop and 44px on phones, saved with the existing
Save action. Copy goes through `t()` in English, pseudo, and the seven
translated catalogs.

## Observability

No new metric. Parse errors with `invalid_date` appear in the existing file
error rows.

## Related decisions

- [ADR 2026-10-04 repository plan files source of truth](../../../decisions/2026-10-04-repository-plan-files-source-of-truth.md)
