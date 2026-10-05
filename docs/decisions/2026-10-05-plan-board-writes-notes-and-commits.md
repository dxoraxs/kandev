# ADR-2026-10-05-plan-board-writes-notes-and-commits: The plan board appends notes to plan files and commits them on request

**Status:** accepted
**Date:** 2026-10-05
**Area:** backend

## Context

[ADR 2026-10-04](2026-10-04-repository-plan-files-source-of-truth.md) limits
Kandev's writes to frontmatter keys of plan files and leaves them uncommitted.
Owners now want to answer a plan from the board (accept, return with a
comment), have dated plans come back on their date, keep a generated index,
create plan files, and commit plan edits. Each of these needs a write that
the first decision did not allow.

## Decision

- Kandev may append single-line, dated notes to one named section of a plan
  file's body. It never edits or removes existing body text. A note and its
  frontmatter change are one compare-and-swap write.
- Kandev may create a plan file and a generated index file inside a scanned
  directory. It overwrites only index files that carry its marker line and
  never replaces an existing plan file.
- Time can change a file: a `waiting_external` or `deferred` plan whose `date`
  arrives is rewritten to `waiting_owner`. The file stays the source of truth
  because the change is recorded in it, with a note that names the previous
  status.
- Kandev creates a git commit only on an explicit owner action, containing
  only plan files and plan indexes, on the current branch, and never pushes.

## Consequences

- The board can carry a full review loop without the owner editing Markdown,
  and agents read the owner's answer in the file.
- Kandev writes more than frontmatter. The write surface stays narrow:
  append-only notes, marker-owned index files, new files that did not exist.
- A pass can now modify files without a board edit (wake-up, index). Both are
  idempotent, and the index contains nothing that changes between passes.
- Commit hooks of the repository run inside the backend process's
  environment; a failing hook is reported and leaves the index unchanged.

## Alternatives Considered

- **Keep decisions in task comments only.** No new write surface, but an
  agent working from the file would not see the answer, and the file would
  stop being the source of truth for review state.
- **Move a dated plan on the board without changing the file.** The board and
  the file would disagree until someone edits the file, which the first ADR
  rules out.
- **Commit automatically after every write-back.** Keeps the tree clean but
  creates commits the owner did not ask for and interleaves with agents' work
  on the same branch.
