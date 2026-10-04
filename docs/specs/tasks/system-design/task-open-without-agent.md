---
status: current
system: tasks
requirements:
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-001
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-002
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-003
---

# Task Open Without Agent System Design

## Purpose and boundaries

The task system owns `session.ensure`, the server-authoritative entry point
that runs when a task is opened, and the agent-profile resolution it applies.
This design adds a typed "no agent profile" outcome to that entry point for
passive opens and a shared sessionless view to the web client.

Adjacent contracts used but not owned here:

- [Prevent agent auto-start on open](../requirements/prevent-agent-autostart-on-open.md):
  the `auto_start` override and the final-step gate are unchanged.
- [Queued session ownership](queued-session-ownership.md): a queued or
  deferred launch still short-circuits a passive open before profile
  resolution.
- [Task launch failure recovery](../requirements/task-launch-failure-recovery.md):
  real ensure failures keep their existing banner and recovery actions.

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-001` | [Backend ensure outcome](#backend-ensure-outcome) |
| `REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-002` | [Sessionless task view](#sessionless-task-view) |
| `REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-003` | [Unassigned notice](#unassigned-notice) |

## Backend ensure outcome

`Service.EnsureSession` (`apps/backend/internal/orchestrator/session_ensure.go`)
resolves the profile with `resolveTaskAgentProfile` after the queued-launch
short-circuit and the existing-session lookup. Today an empty result still
flows into `LaunchSession` with `IntentPrepare`, and the executor returns
`ErrNoAgentProfileID` from `prepareSessionAttempt`, which the WS handler maps
to `INTERNAL_ERROR`.

New rule, applied immediately after `resolveTaskAgentProfile`:

- When the resolved profile is empty and
  `opts.ActivationSource == LaunchActivationSourceSessionOpen`, return
  `EnsureSessionResponse{Success: true, TaskID, Source: "no_agent_profile",
  NewlyCreated: false}` with no `SessionID` and no `State`. `LaunchSession` is
  not called, so no session row, workspace, worktree, or execution is created
  (AC-001.1, AC-001.2). The method logs once at debug level.
- Any other activation source (`user_action`, the plugin exact-run adapter in
  `internal/backendapp/adapters_plugin_execution.go`, and the HTTP
  `POST /api/v1/tasks/:id/sessions/ensure`, which passes no activation source)
  keeps the current path and the current error (AC-001.3).

The outcome is computed on every call and never persisted, so a profile that
resolves later (workspace default, step profile, task metadata) produces a
normal `created_prepare` or `created_start` on the next passive open
(AC-001.4).

The new source value joins the documented set on `EnsureSessionResponse.Source`
(`existing_primary | existing_newest | created_prepare | created_start |
skipped_terminal_pr | queued | existing_queued | no_agent_profile`).

## Data and contracts

- WS `session.ensure` response: `source` gains `"no_agent_profile"`;
  `session_id` and `state` are absent for it. The request shape is unchanged.
- Web type `EnsureSessionResponse` in
  `apps/web/lib/services/session-launch-service.ts` gains the same literal.
- `EnsureTaskSessionStatus` in
  `apps/web/hooks/domains/session/use-ensure-task-session.ts` gains
  `"unassigned"`. `useEnsureTaskSession` sets it when the resolved response's
  `source` is `no_agent_profile`, skips the forced session reload for that
  outcome, and keeps the per-task latch so the open does not loop.
  `retry()` re-runs ensure as today.

## Sessionless task view

`SessionlessTaskView` (`apps/web/components/task/sessionless-task-view.tsx`)
renders a sessionless task: its description, with the ensure status in a
corner (`preparing` spinner or the [unassigned notice](#unassigned-notice);
`idle` and `error` add nothing, the error banner stays page-level, AC-003.6).
An empty description renders the placeholder copy (AC-002.2). Its layout,
mounts, tab composition, and phone form are owned by
[task description view](task-description-view.md).

Ensure status reaches the view through a small React context,
`SessionlessTaskContext` (`{ taskId, workspaceId, status }`), provided by
`TaskPageContent` (`components/task/task-page-content.tsx`) around the layout.

## Unassigned notice

For `status === "unassigned"` the notice offers:

- Title `task:noAgentProfileConfigured`, with the detail
  `task:noAgentProfileConfiguredDetail` behind a disclosure.
- A primary "Start agent" action that opens `NewSessionDialog`
  (`components/task/new-session-dialog.tsx`) with the task id and workspace id.
  The dialog already handles a task with no current session and already shows
  `task:noAgentProfilesConfiguredAddOne` when no profile exists at all
  (AC-003.2).
- A link `task:openWorkspaceSettings` to `/settings/workspaces/{workspaceId}`
  when the workspace id is known (AC-003.3).

`describeEnsureError` (`components/task/ensure-session-error.tsx`) is
unchanged. Its `AGENT_PROFILE_MISSING_HINT` branch still serves other paths
that return `agent_profile_id is required ...` (for example the
`task_ws_handlers.go` start handler) and session-recovery errors routed through
`SessionRecoveryFeedback`; it never matched the ensure path's text, which this
design makes unnecessary for passive opens.

## Failure and recovery

- Profile resolution itself cannot fail; lookup errors already degrade to an
  empty profile, which now yields the unassigned outcome on passive open.
- A task deleted between open and ensure still maps to `NOT_FOUND`.
- Any error after the new short-circuit (`LaunchSession` failures) keeps the
  `INTERNAL_ERROR` path and the existing banner.

## Observability

The unassigned outcome is logged at debug level with the task id only. It adds
no metric; the existing error log lines `failed to prepare session` and
`failed to ensure session` stop firing for passive opens of unassigned tasks,
which removes false error noise from backend logs.

## Testing

- Go: `session_ensure_test.go` cases for passive open with no profile (outcome,
  no launch call), explicit `user_action` with no profile (error unchanged),
  and profile-present passive open (unchanged).
- Go: `handlers` WS test that the response carries `source: no_agent_profile`.
- Web unit: `use-ensure-task-session.test.ts` for the `unassigned` status and
  no forced reload.
- Web unit: `SessionlessTaskView` rendering decisions as a pure helper test.
- E2E (Playwright, desktop and mobile projects): a task in a workspace with no
  default profile opens with its description and the neutral notice, no
  `ensure-session-error-banner`, and "Start agent" opens the dialog.
