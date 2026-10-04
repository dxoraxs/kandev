---
id: "01-unadapted-detection"
title: "Unadapted plan file detection"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-ADAPT-001
  - REQ-TASKS-PLAN-ADAPT-003
acceptance_criteria:
  - AC-TASKS-PLAN-ADAPT-001.1
  - AC-TASKS-PLAN-ADAPT-001.3
  - AC-TASKS-PLAN-ADAPT-001.4
  - AC-TASKS-PLAN-ADAPT-003.2
system_design:
  - ../../specs/tasks/system-design/plan-file-adaptation.md
---

# Task 01: Unadapted plan file detection

## Summary

Count Markdown files in scanned plan directories that lack a `board`
frontmatter key, expose the count in the pass status and through a read
endpoint that works without a sync config, and add `EnsureBoard` for the
adaptation launcher.

## In scope

- `PassCounts.Unadapted` incremented in `pass.collect` where `format.Parse`
  returns `ok == false`; per-repository tally kept on the pass.
- `Service.UnadaptedCounts(ctx, workspaceID, repositoryID string)`: scans local
  repositories with config directories or `DefaultDirectories()`, returns
  `[]UnadaptedRepo{RepositoryID, RepositoryName, Count, Directories}` for counts
  above zero; workspace authorization as in `GetConfig`.
- `GET /api/v1/plan-files/unadapted` handler and route.
- `Service.EnsureBoard(ctx, workspaceID) (created bool, err error)`.

## Out of scope

- Any change to plan file parsing, file error rows, or pass outcome.
- Frontend.

## Acceptance

- A repository with three files without `board` and one plan file reports
  `unadapted: 3` in `last_counts` and no file error row; the outcome is unchanged.
- The endpoint returns counts for a workspace with no config, using default
  directories, and omits remote repositories and repositories with zero.
- `EnsureBoard` creates a board and an enabled config once; a second call and a
  workspace with an existing disabled config change nothing.

## Verification

```bash
cd apps/backend && go test ./internal/planfiles/... -run 'Unadapted|EnsureBoard|Pass'
cd apps/backend && go vet ./internal/planfiles/...
make -C apps/backend lint
```

## Files likely touched

- `apps/backend/internal/planfiles/models.go`
- `apps/backend/internal/planfiles/sync_pass.go`, `sync_types.go`
- `apps/backend/internal/planfiles/service_config.go` (or new `service_unadapted.go`)
- `apps/backend/internal/planfiles/handlers.go`
- Tests next to each file.

## Dependencies

None.
