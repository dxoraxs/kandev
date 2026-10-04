---
id: "02-config-and-settings"
title: "Config, storage, and settings fields"
status: pending
wave: 2
depends_on: ["01-format-additions"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-001
  - REQ-TASKS-PLAN-BOARD-OPS-003
  - REQ-TASKS-PLAN-BOARD-OPS-005
  - REQ-TASKS-PLAN-BOARD-OPS-006
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-001.1
  - AC-TASKS-PLAN-BOARD-OPS-003.1
  - AC-TASKS-PLAN-BOARD-OPS-005.4
  - AC-TASKS-PLAN-BOARD-OPS-006.1
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 02: Config, storage, and settings fields

## Summary

Persist and expose the five new workspace settings (executor steps, notes heading, wake-up, stale threshold, index file) and the `synced_depends_on` row column, with validation and the settings controls. No sync behavior uses them yet.

## In scope

- Add-column migration for both tables; `Config`, `PutConfigRequest` (pointer fields), scan and upsert SQL, `TaskRow.SyncedDependsOn`.
- Validation codes `invalid_executor_steps`, `invalid_notes_heading`, `invalid_stale_after_days`, `invalid_index_file`.
- Required-stores and store-conformance registrations, upgrade fixture manifest.
- Web: API types, `use-plan-files` draft fields, and the controls of UI-02 except the git block; copy in every locale.

## Out of scope

- Any use of the settings by the pass or write-back.
- The git block of UI-02 (task 05).

## Acceptance

- A `PUT /config` body without the new fields leaves stored values unchanged; a database created before this change gains the columns with defaults on startup.
- Saving an executor step that is a mapped status step, an unknown step, or a duplicate name returns 400 with `invalid_executor_steps`.
- The settings section edits and saves all five settings on desktop and phone widths in the component test.

## ASCII UI preview

UI-02: Plan files settings additions (without the git block). Full preview: [plan.md](plan.md#ascii-ui-preview). Covers AC-001.1, 003.1, 005.4, 006.1.

```text
Executor columns
  [ Hand to Claude      v ]  [ Claude        ]  [ x ]
  [ + Add executor column ]

Notes section heading   [ Owner notes        ]
[x] Move plans to "Waiting for owner" when their date arrives
Mark plans in progress as stale after  [ 7 ] days (0 = never)
Index file name         [ INDEX.md           ]  (empty = no index)
```

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
(cd apps/backend && go run ./cmd/sqlguard ./internal)
(cd apps/backend && go test -race ./internal/persistence/storeconformance ./internal/persistence/requiredstores -count=1)
(cd apps && pnpm --filter @kandev/web test -- components/settings/plan-files-section.test.tsx)
(cd apps && pnpm --filter @kandev/web test -- lib/api/domains/plan-files-api.test.ts)
(cd apps/web && pnpm run typecheck)
(cd apps/web && pnpm run i18n:check)
(cd apps && pnpm --filter @kandev/web lint)
```

## Files likely touched

- apps/backend/internal/planfiles/store.go, models.go, service_config.go, service_config_ops_test.go, store_test.go
- apps/backend/internal/persistence/storeconformance/adapters.go, owner_actions.go, testdata/upgrades/v0.93.0/manifest.json
- `apps/web/lib/api/domains/plan-files-api.ts`
- `apps/web/hooks/domains/settings/use-plan-files.ts`
- apps/web/components/settings/plan-files-section.tsx, plan-files-operations.tsx (new)
- `apps/web/src/locales/*/planFiles.json`

## Dependencies

Task 01

## Risks

- Unified plan sync task 02 adds `summary_heading` with the same add-column approach; if it is already on the branch, reuse its helper instead of adding a second one.
- `plan-files-section.tsx` is near the 200-line component guideline; put the new controls in a new component file.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Pending.
