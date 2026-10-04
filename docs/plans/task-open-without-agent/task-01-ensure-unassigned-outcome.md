---
id: "01-ensure-unassigned-outcome"
title: "Ensure unassigned outcome"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-TASK-OPEN-WITHOUT-AGENT-001
acceptance_criteria:
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.1
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.2
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.3
  - AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.4
system_design:
  - ../../specs/tasks/system-design/task-open-without-agent.md
---

# Task 01: Ensure unassigned outcome

## Summary

`Service.EnsureSession` returns `Source: "no_agent_profile"` (success, no
session) for a passive open when no agent profile resolves, without calling
`LaunchSession`.

## In scope

- Short-circuit after `resolveTaskAgentProfile` in
  `apps/backend/internal/orchestrator/session_ensure.go`, gated on
  `LaunchActivationSourceSessionOpen`.
- Update the `Source` field comment on `EnsureSessionResponse`.
- Debug-level log with the task id.

## Out of scope

- Any change for `user_action`, the plugin exact-run adapter, or the HTTP
  ensure endpoint: they keep returning the existing error.
- Web changes (task 02).

## Acceptance

- Passive open of a task with no resolvable profile returns success,
  `source == "no_agent_profile"`, empty `session_id`, and the launch path is
  never invoked (assert no session row is created).
- `user_action` open of the same task still returns an error wrapping
  `ErrNoAgentProfileID`; a passive open with a workspace default still returns
  `created_prepare` or `created_start`.
- WS `session.ensure` returns a normal response (not an error envelope) with
  `source: "no_agent_profile"`.

## Verification

```bash
cd apps/backend && go test ./internal/orchestrator/ -run 'EnsureSession' -count=1
cd apps/backend && go test ./internal/orchestrator/handlers/ -run 'Ensure' -count=1
cd apps/backend && golangci-lint run ./internal/orchestrator/...
```

## Files likely touched

- `apps/backend/internal/orchestrator/session_ensure.go`
- `apps/backend/internal/orchestrator/session_ensure_test.go`
- `apps/backend/internal/orchestrator/handlers/handlers_test.go`

## Dependencies

None.

## Risks

- The queued-launch short-circuit must stay before the new check, and the
  existing-session lookup must stay before it too, or a task with an existing
  session but no profile would stop showing that session.

## Results

- `EnsureSession` returns `source: "no_agent_profile"` for a passive open with
  no resolved profile, before `LaunchSession`.
- Coverage: `session_ensure_no_profile_test.go` (passive outcome with zero
  session rows, explicit `user_action` still `ErrNoAgentProfileID`, positive
  control with a resolved profile creating one session).
- No WS handler test was added: `wsEnsureSession` passes a successful service
  response through unchanged, and constructing a full `orchestrator.Service`
  in the handler package only re-proves the service test.
- `go test ./internal/orchestrator/ -run 'EnsureSession'`: ok;
  `go test ./internal/orchestrator/handlers/ -run 'Ensure'`: ok;
  `golangci-lint run ./internal/orchestrator/...`: 0 issues.
