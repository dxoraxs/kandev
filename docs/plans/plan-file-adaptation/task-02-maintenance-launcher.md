---
id: "02-maintenance-launcher"
title: "Repository maintenance task launcher and plan adaptation kind"
status: done
wave: 2
depends_on: ["01-unadapted-detection"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-ADAPT-003
  - REQ-TASKS-PLAN-ADAPT-004
acceptance_criteria:
  - AC-TASKS-PLAN-ADAPT-003.1
  - AC-TASKS-PLAN-ADAPT-003.2
  - AC-TASKS-PLAN-ADAPT-003.3
  - AC-TASKS-PLAN-ADAPT-003.4
  - AC-TASKS-PLAN-ADAPT-003.5
  - AC-TASKS-PLAN-ADAPT-004.1
  - AC-TASKS-PLAN-ADAPT-004.2
  - AC-TASKS-PLAN-ADAPT-004.3
  - AC-TASKS-PLAN-ADAPT-004.4
system_design:
  - ../../specs/tasks/system-design/plan-file-adaptation.md
---

# Task 02: Repository maintenance task launcher

## Summary

Add `POST /api/v1/repositories/:id/maintenance-tasks`, which creates a
non-ephemeral task on the repository's main checkout with the local executor
and a built-in prompt, and launches the agent. Implement the `plan_adaptation`
kind with the `PlanFilesSetup` hook and the adaptation prompt. Design the kind
table so the cleanup plan adds `repository_cleanup` without changing the
launcher.

## In scope

- Handler, request/response types, typed `reason` values, route registration.
- Kind registry: availability check, preparation hook, prompt name, variables,
  title. `plan_adaptation` available only when `PlanFilesSetup` is non-nil.
- Active-task guard by metadata (`repository_maintenance_kind`,
  `repository_maintenance_repository_id`) with a per-repository mutex.
- Resolution: default agent profile, first visible non-plan-board workflow and
  its start step, base branch from `RepositoryCurrentBranch`.
- `CreateTask` + `LaunchSession` modeled on `httpStartQuickChat`, but with
  `exec-local` and a workflow.
- `config/prompts/plan-file-adaptation.md` and a rendering test that asserts
  every rule of `AC-TASKS-PLAN-ADAPT-004.1` to `004.4`.
- `repository_maintenance_task_total{kind,outcome}` metric.
- Wiring `planfiles.Service` as `PlanFilesSetup` in `backendapp`.

## Out of scope

- The cleanup kind, its flag, and its prompt.
- Frontend.

## Acceptance

- Handler tests cover 201 create, 200 existing, and each 409/404 reason, and
  assert executor `exec-local`, base branch equal to the current branch,
  workflow and step, metadata, and that `EnsureBoard` runs only for
  `plan_adaptation`.
- Two concurrent requests for one repository create exactly one task.
- The prompt test renders with tag-bearing repository names and shows them
  stripped.

## Verification

```bash
cd apps/backend && go test ./internal/task/handlers/... -run 'Maintenance'
cd apps/backend && go test ./internal/task/service/... -run 'Maintenance'
cd apps/backend && go test ./config/prompts/... ./internal/sysprompt/...
make -C apps/backend lint
```

## Files likely touched

- `apps/backend/internal/task/handlers/repository_handlers.go` (route)
- `apps/backend/internal/task/handlers/maintenance_task_handlers.go` (new)
- `apps/backend/internal/task/service/maintenance_tasks.go` (new: guard, resolution)
- `apps/backend/config/prompts/plan-file-adaptation.md` (new)
- `apps/backend/internal/backendapp/helpers.go` (wiring)
- Tests next to each file.

## Dependencies

Task 01 (`EnsureBoard`, `UnadaptedCounts` for `{plan_directories}`).
