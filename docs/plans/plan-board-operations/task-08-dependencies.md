---
id: "08-dependencies"
title: "Plan dependencies"
status: pending
wave: 8
depends_on: ["02-config-and-settings"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-004
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-004.1
  - AC-TASKS-PLAN-BOARD-OPS-004.2
  - AC-TASKS-PLAN-BOARD-OPS-004.3
  - AC-TASKS-PLAN-BOARD-OPS-004.4
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 08: Plan dependencies

## Summary

Turn resolvable `depends_on` entries into task dependencies so blocked plans show the blocked badge and are not auto-started.

## In scope

- `TaskAccess.AddDependency` and `RemoveDependency` (and the fake); `dependencies.go` with the diff against `SyncedDependsOn`.
- File errors `unknown_dependency` and `invalid_dependency`.
- An integration test in `backendapp` that a blocked plan task moved into an auto-start step does not start an agent.

## Out of scope

- Editing `depends_on` from the board.
- Dependencies across directories or repositories.

## Acceptance

- After a pass, a plan with `depends_on: [a.md, b.md]` depends on exactly those two plan tasks; removing `b.md` from the list removes that dependency on the next pass and keeps a dependency a user added on a non-plan task.
- A dependency cycle between two plan files is reported for the file whose edge was rejected, and the pass completes.
- A pass with unchanged `depends_on` makes zero dependency calls (asserted on the fake call log).

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
(cd apps/backend && go test ./internal/backendapp/ -run PlanFiles -count=1)
```

## Files likely touched

- apps/backend/internal/planfiles/dependencies.go, dependencies_test.go (new)
- apps/backend/internal/planfiles/sync_types.go, sync_pass.go, fake_tasks_test.go
- apps/backend/internal/backendapp/planfiles_dependencies_integration_test.go (new)
- docs/specs/tasks/requirements/repository-plan-files.md (AC-002.4 wording already updated)

## Dependencies

Task 02

## Risks

- A dependency resolves only when the predecessor task is COMPLETED; a custom plan board whose done step does not complete tasks keeps dependents blocked. Document it in the public docs (task 12).

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Pending.
