---
id: "02-cleanup-kind-and-prompt"
title: "Repository cleanup launcher kind and prompt"
status: pending
wave: 2
depends_on: ["01-runtime-flag", "../plan-file-adaptation/02-maintenance-launcher"]
plan: "plan.md"
requirements:
  - REQ-WORKSPACES-REPO-CLEANUP-001
  - REQ-WORKSPACES-REPO-CLEANUP-002
  - REQ-WORKSPACES-REPO-CLEANUP-003
acceptance_criteria:
  - AC-WORKSPACES-REPO-CLEANUP-001.3
  - AC-WORKSPACES-REPO-CLEANUP-001.4
  - AC-WORKSPACES-REPO-CLEANUP-001.5
  - AC-WORKSPACES-REPO-CLEANUP-001.8
  - AC-WORKSPACES-REPO-CLEANUP-002.1
  - AC-WORKSPACES-REPO-CLEANUP-002.2
  - AC-WORKSPACES-REPO-CLEANUP-002.3
  - AC-WORKSPACES-REPO-CLEANUP-002.4
  - AC-WORKSPACES-REPO-CLEANUP-002.5
  - AC-WORKSPACES-REPO-CLEANUP-003.1
  - AC-WORKSPACES-REPO-CLEANUP-003.2
  - AC-WORKSPACES-REPO-CLEANUP-003.3
  - AC-WORKSPACES-REPO-CLEANUP-003.4
system_design:
  - ../../specs/workspaces/system-design/repository-cleanup.md
---

# Task 02: Cleanup launcher kind and prompt

## Summary

Add the `repository_cleanup` kind to the maintenance launcher: available only
when the flag is on, builds the protected list from Kandev worktree records,
renders `config/prompts/repository-cleanup.md`, and titles the task
`Clean up repository: <repo>`.

## In scope

- Kind registration with availability from `cfg.Features.RepositoryCleanup`.
- Protected list from `GetWorktreesByRepositoryID` filtered to non-archived
  tasks; a read failure rejects the request.
- Prompt template with the seven phases from the system design.
- Rendering test asserting each phase, the forbidden operations, the merged-only
  remote deletion command, and both protected lists (including tag stripping).

## Out of scope

- Launcher changes beyond registering the kind; frontend.

## Acceptance

- Handler tests: 201 with the cleanup title and metadata when the flag is on;
  404 `kind_unavailable` when off; 200 existing on a second request; protected
  list contains a non-archived task's worktree and excludes an archived one.
- Prompt test passes for every rule listed in the system design.

## Verification

```bash
cd apps/backend && go test ./internal/task/handlers/... -run 'Maintenance.*Cleanup|Cleanup'
cd apps/backend && go test ./config/prompts/...
make -C apps/backend lint
```

## Files likely touched

- `apps/backend/internal/task/handlers/maintenance_task_handlers.go`
- `apps/backend/internal/task/service/maintenance_tasks.go`
- `apps/backend/config/prompts/repository-cleanup.md` (new)
- Tests next to each file.

## Dependencies

Task 01 and plan-file-adaptation Task 02 (launcher).
