# ADR-2026-10-05-repository-maintenance-agent-tasks: Repository maintenance runs as an agent task on the main checkout

**Status:** accepted
**Date:** 2026-10-05
**Area:** backend

## Context

Two repository-level actions need judgment that a fixed algorithm cannot
provide:

- Adapting an existing plan collection to the plan file format. The status of
  each plan has to be inferred from evidence such as status banners, merge
  history, and the presence of the code a plan describes. Checkboxes are not
  reliable evidence: completed plans often have none ticked.
- Cleaning up a repository: merging every branch and worktree into the default
  branch, resolving conflicts, removing merged branches locally and on the
  remote, and pushing.

Both actions change the repository the owner is working in, not a task
worktree, and both are rare, owner-initiated, and repository-scoped.

## Decision

- Each action is a Kandev task started from a button. The task description is
  a built-in prompt rendered by the backend from an embedded template in
  `apps/backend/config/prompts/`; repository-derived values are passed through
  `sysprompt.StripTags` before substitution.
- The task runs on the repository's main checkout with the system local
  executor (`exec-local`) and the repository's current branch as the base
  branch, so task launch performs no checkout and no discard. Only repositories
  with source type `local` qualify.
- The agent profile is the workspace default agent profile. Without one, the
  action is refused with a typed reason; the backend never guesses a profile.
- At most one active maintenance task of any kind exists per repository, so plan
  adaptation and cleanup never run at once on the same checkout. A second
  request, whatever its kind, returns the existing task instead of creating
  another.
- Irreversible remote effects (remote branch deletion, push) happen only after
  the owner confirms a summary inside the task conversation. The prompt owns
  that contract; the backend does not grant or revoke git permissions.
- Tasks are ordinary, non-ephemeral tasks, so the owner sees progress, the
  conversation, and the diff with the existing task UI.

## Consequences

- No new git automation code in the backend: behavior lives in reviewed prompt
  templates, which are covered by Go tests that assert the rendered rules.
- Running on the main checkout shares it with the owner. The confirmation
  dialog states this, and the one-active-task guard prevents two maintenance
  agents from racing on the same checkout.
- Quality and cost depend on the agent model; the owner reviews the outcome in
  the task conversation and git history.
- The confirmation before irreversible steps is enforced by the agent, not by
  a technical barrier. A future change can add a backend-enforced gate if
  prompt enforcement proves insufficient.

## Alternatives Considered

- **Deterministic backend conversion with heuristics.** Instant and free, but
  the heuristics misclassify completed plans whose checkboxes were never ticked,
  so the owner must correct most rows by hand.
- **Backend git automation for cleanup.** Predictable for fast-forward merges,
  but cannot resolve conflicts or judge whether a merge is safe; it would need
  its own conflict UI.
- **Ephemeral quick-chat style task.** Hidden from the board, so a long cleanup
  awaiting confirmation would be easy to lose.
- **Run in a fresh worktree.** Isolates the owner's checkout, but cleanup must
  operate on the real local branches and worktrees, and plan adaptation must
  land on the branch the plan sync scans.
