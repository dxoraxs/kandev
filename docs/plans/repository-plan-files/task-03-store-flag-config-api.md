---
id: "03-store-flag-config-api"
title: "Store, flag, template, and config API"
status: done
wave: 2
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-FILES-005
acceptance_criteria:
  - AC-TASKS-PLAN-FILES-005.2
  - AC-TASKS-PLAN-FILES-005.4
  - AC-TASKS-PLAN-FILES-005.6
system_design:
  - ../../specs/tasks/system-design/repository-plan-files.md
---

# Task 03: Store, flag, template, and config API

## Summary

Persist plan-file configuration and per-task sync rows, add the
`features.planFiles` runtime flag and the built-in Plans workflow template,
and expose config read, write, and board creation over HTTP with workspace
authorization. Follow `/runtime-feature-flags` for the flag.

## In scope

- `plan_file_configs` and `plan_file_tasks` schema for SQLite and Postgres,
  store CRUD, and registration in `persistence/requiredstores/catalog.go`,
  store conformance owner actions, the workspace-deletion registry, and
  `backendapp/e2e_reset.go`.
- Flag per `/runtime-feature-flags`: registry entry and `runtimeflags/config.go`
  wiring, config field, `profiles.yaml` (`prod`, `dev`, `e2e` all false), web
  `defaultFeatureFlags`.
- `apps/backend/config/workflows/plans.yml` and `planfiles/template.go`.
- `GET/PUT /api/v1/plan-files/config`, `POST /api/v1/plan-files/board`,
  service authorizer, `integrationWorkspacePrefixes` entry, wiring in
  `backendapp/services.go` under the flag.

## Out of scope

- Sync pass, poller, write-back, UI.

## Acceptance

- PUT rejects a foreign workflow, a step from another workflow, a missing
  status, and an absolute or `..` directory; a non-owner gets 404.
- Board creation returns a workflow whose mapping covers every visible status.
- With the flag off, the routes are not registered and the service is nil.

## Verification

```bash
cd apps/backend && go test ./internal/planfiles/... ./internal/runtimeflags/... ./internal/profiles/... ./internal/persistence/... ./config/... -count=1
cd apps/backend && golangci-lint run ./internal/planfiles/... ./internal/backendapp/...
```

## Files likely touched

- `apps/backend/internal/planfiles/store.go`, `store_test.go`
- `apps/backend/internal/planfiles/service_config.go`, `service_config_test.go`
- `apps/backend/internal/planfiles/handlers.go`, `handlers_test.go`
- `apps/backend/internal/planfiles/template.go`
- `apps/backend/config/workflows/plans.yml`
- `apps/backend/internal/runtimeflags/registry.go`
- `apps/backend/internal/common/config/config.go`
- `apps/backend/internal/profiles/profiles.yaml`
- `apps/backend/internal/persistence/requiredstores/catalog.go`
- `apps/backend/internal/persistence/storeconformance/owner_actions.go`
- `apps/backend/internal/backendapp/services.go`, `helpers.go`, `e2e_reset.go`
- `apps/web/lib/state/slices/features/types.ts`

## Dependencies

None in code; wave 2 keeps package layout decisions after Tasks 01 and 02.

## Risks

- Completeness tests in persistence and runtime flags fail on any missed
  registration; run the listed packages, not only `planfiles`.
