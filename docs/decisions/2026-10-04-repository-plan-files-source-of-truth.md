# ADR-2026-10-04-repository-plan-files-source-of-truth: Repository plan files are the source of truth for plan tasks

**Status:** accepted
**Date:** 2026-10-04
**Area:** backend

## Context

Teams write implementation plans as Markdown files in their repositories and
want each plan on a Kandev board. One installation already does this with an
external script that reads one repository's plan cards, uses a project-specific
status line, and creates tasks through the REST API. The script cannot serve
other workspaces, and other repositories keep plans in a different layout with
no machine-readable status.

Kandev needs a built-in way to project plan files of every local repository in
a workspace onto a board, while the files remain readable and editable by
agents and other tools that never talk to Kandev.

## Decision

- The plan file in the repository working tree is the source of truth. Kandev
  tasks for plan files are a projection keyed by an external identifier.
- The status, priority, and order live in YAML frontmatter with English keys
  and a closed English status vocabulary (`board: queued`, and so on). Board
  column names stay workflow data, so they can be localized by the owner.
- A file is a plan file only when its frontmatter declares `board`. Files
  without it are ignored, so existing documents never appear by accident.
- Board edits to status, priority, and order write back into the frontmatter of
  the working-tree file, uncommitted, with a compare-and-swap on the file's
  content hash. A conflicting file change wins, and the board follows it on the
  next pass.
- Steps with no mapped status are handoff steps. Moving a plan into one does
  not write the file; the agent that starts there updates the file itself.
- Kandev polls local repositories. It does not commit, push, or create plan
  files.

## Consequences

- One format serves every project, and agents can set a plan's status with a
  one-line file edit that works with or without Kandev running.
- Kandev now writes into user repositories. Writes are limited to frontmatter
  keys of files inside registered local repositories, are atomic, and never
  overwrite a newer file version.
- Existing plan collections need a one-time migration to the frontmatter
  format. An optional `external_id` key lets a migrated file keep the task an
  earlier tool created.
- Uncommitted write-back edits show up in `git status`; committing them is the
  owner's or agent's job.

## Alternatives Considered

- **One-way sync, board edits reverted.** Simpler and conflict-free, but the
  owner could not manage plans from the board, which was the requested
  workflow.
- **A project-specific status line in the body** (for example a localized
  `**Status:**` line). Readable, but needs per-language keywords inside Kandev
  and is harder to edit safely than a frontmatter key.
- **Default every Markdown file in plan directories to queued.** Makes a
  forgotten status visible, but floods the board with historical plans and
  index files; plan creation tooling sets the status instead.
- **Keep an external sync script per workspace.** No Kandev change, but every
  project needs its own service, and board write-back would need direct file
  access from the script anyway.
