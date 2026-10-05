---
id: "12-e2e-and-docs"
title: "E2E, public docs, and promotion"
status: done
wave: 12
depends_on: ["03-executor-steps", "04-owner-decisions", "05-git-state-and-commit", "06-create-plan", "07-plan-index", "08-dependencies", "09-progress-and-flags", "10-date-wake-up", "11-waiting-owner-page"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-001
  - REQ-TASKS-PLAN-BOARD-OPS-002
  - REQ-TASKS-PLAN-BOARD-OPS-003
  - REQ-TASKS-PLAN-BOARD-OPS-004
  - REQ-TASKS-PLAN-BOARD-OPS-005
  - REQ-TASKS-PLAN-BOARD-OPS-006
  - REQ-TASKS-PLAN-BOARD-OPS-007
  - REQ-TASKS-PLAN-BOARD-OPS-008
  - REQ-TASKS-PLAN-BOARD-OPS-009
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-002.1
  - AC-TASKS-PLAN-BOARD-OPS-002.3
  - AC-TASKS-PLAN-BOARD-OPS-002.6
  - AC-TASKS-PLAN-BOARD-OPS-005.5
  - AC-TASKS-PLAN-BOARD-OPS-007.1
  - AC-TASKS-PLAN-BOARD-OPS-007.4
  - AC-TASKS-PLAN-BOARD-OPS-007.5
  - AC-TASKS-PLAN-BOARD-OPS-008.1
  - AC-TASKS-PLAN-BOARD-OPS-008.2
  - AC-TASKS-PLAN-BOARD-OPS-008.3
  - AC-TASKS-PLAN-BOARD-OPS-009.1
  - AC-TASKS-PLAN-BOARD-OPS-009.2
  - AC-TASKS-PLAN-BOARD-OPS-009.4
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 12: E2E, public docs, and promotion

## Summary

Prove the user-visible flows in Playwright on desktop and phone, document the capability, and promote the specifications.

## In scope

- `e2e/tests/plans/plan-board-operations.spec.ts` and `mobile-plan-board-operations.spec.ts` with the four flows of the plan; helper additions in `e2e/helpers/plan-files.ts`.
- `docs/public/plan-files.md` (new keys `tracks`, executor columns, decisions, wake-up, index, commit, waiting page), `docs/public/coverage.json`, `README.md` fork section.
- `apps/backend/AGENTS.md` planfiles notes and the root `AGENTS.md` observability list for the new `plan_files_*` counters.
- Promote the requirement to `active`, the design to `current`, the plan to `implemented`, with recorded verification results.

## Out of scope

- New product behavior.

## Acceptance

- Both specs pass with causal waits only (no fixed sleeps).
- `python3 scripts/list-docs.py validate` and `python3 scripts/lint-spec-files.py --all` pass.

## Verification

```bash
make build-web build-backend
(cd apps/web && pnpm e2e:run --no-build tests/plans/plan-board-operations.spec.ts)
(cd apps/web && pnpm e2e:run --no-build --project mobile-chrome tests/plans/mobile-plan-board-operations.spec.ts)
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
```

## Files likely touched

- apps/web/e2e/tests/plans/plan-board-operations.spec.ts, mobile-plan-board-operations.spec.ts (new)
- `apps/web/e2e/helpers/plan-files.ts`
- docs/public/plan-files.md, docs/public/coverage.json
- README.md, AGENTS.md, apps/backend/AGENTS.md
- docs/specs/tasks/requirements/plan-board-operations.md, docs/specs/tasks/system-design/plan-board-operations.md, docs/plans/plan-board-operations/plan.md

## Dependencies

Task 03, Task 04, Task 05, Task 06, Task 07, Task 08, Task 09, Task 10, Task 11

## Risks

- Local E2E runs enforce one worker per shard; do not overlap with another suite running on this machine.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Promotion of the requirement, design, and plan statuses is left to the primary session (scope change); the requirement stays `draft`.

| Command | Final output line |
| --- | --- |
| `make build-web build-backend` | exit 0 |
| `(cd apps/web && pnpm e2e:run --no-build tests/plans/plan-board-operations.spec.ts)`, run 1 and run 2 | `4 passed (17.1s)`, `4 passed (15.3s)` |
| `(cd apps/web && pnpm e2e:run --no-build --project mobile-chrome tests/plans/mobile-plan-board-operations.spec.ts)`, run 1 and run 2 | `3 passed (19.4s)`, `3 passed (19.7s)` |
| `(cd apps/web && pnpm run typecheck)` | `tsc --noEmit`, no errors |
| `(cd apps && pnpm --filter @kandev/web lint)` | `eslint --max-warnings 0`, no findings |
| `(cd apps/web && pnpm run i18n:check)` | `no non-JSX copy - 3674 guarded file(s) checked.` |
| `(cd apps && pnpm --filter @kandev/web test -- lib/toast/sonner.test.ts components lib/kanban hooks/domains/plans app/plans-waiting)` | `Tests  11021 passed \| 4 skipped (11025)` |
| `python3.12 scripts/list-docs.py validate` | `Validated 350 decisions and 1347 specifications.` |
| `python3.12 scripts/lint-spec-files.py --all` | `All specification files passed.` |
| `git diff --check` | no output |

The E2E global setup requires `make -C apps/backend e2e-plugin-package` after any change under `apps/backend` (including `AGENTS.md`). Product defect found by the phone spec and fixed with a component test: the Waiting for owner row was only in the saved-layout phone navigation, so the default navigation sheet had no entry (`components/navigation/app-nav-sections.tsx`).
