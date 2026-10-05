---
id: "01-format-additions"
title: "Format additions"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-002
  - REQ-TASKS-PLAN-BOARD-OPS-005
  - REQ-TASKS-PLAN-BOARD-OPS-007
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-002.4
  - AC-TASKS-PLAN-BOARD-OPS-005.1
  - AC-TASKS-PLAN-BOARD-OPS-005.2
  - AC-TASKS-PLAN-BOARD-OPS-007.2
  - AC-TASKS-PLAN-BOARD-OPS-007.3
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 01: Format additions

## Summary

Add the pure format functions every later slice needs: the `tracks` key, Markdown item counting, byte-preserving note appending, new-plan rendering, and the work-order status probe. No behavior changes for existing plan files.

## In scope

- `PlanFile.Tracks` with the `invalid_tracks` parse error (at most 20 entries).
- `CountItems`, fence-aware, in `format/items.go`.
- `AppendNote` in `format/notes.go` with the section rules of the system design, including line-ending preservation and the missing-section case.
- `NewPlan` in `format/newplan.go` and `WorkOrderDone`.
- A fuzz test for `AppendNote` and a round-trip test for `NewPlan` through `Parse`.

## Out of scope

- Any caller of these functions.
- The `date` key (unified plan sync task 01).

## Acceptance

- `AppendNote` output minus the inserted bytes equals the input for every test and fuzz corpus entry, for LF and CRLF files, with and without an existing section.
- `CountItems` ignores items inside fenced code blocks and counts `-` and `*` markers at any indentation.
- `Parse(NewPlan(x))` returns the same title, priority, executor, and body for titles containing `:`, `#`, quotes, and non-ASCII text.

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/format/... -count=1 -race)
(cd apps/backend && go test ./internal/planfiles/format/ -run FuzzAppendNote -fuzz FuzzAppendNote -fuzztime 20s)
make -C apps/backend lint
```

## Files likely touched

- `apps/backend/internal/planfiles/format/parse.go`
- apps/backend/internal/planfiles/format/items.go, items_test.go
- apps/backend/internal/planfiles/format/notes.go, notes_test.go, notes_fuzz_test.go
- apps/backend/internal/planfiles/format/newplan.go, newplan_test.go
- `apps/backend/internal/planfiles/format/parse_tracks_test.go`

## Dependencies

None

## Risks

- `parse.go` is also changed by unified plan sync task 01 (`Date`); keep the `tracks` change to its own decode function to make the later merge trivial.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

- `(cd apps/backend && go test ./internal/planfiles/format/... -count=1 -race)`: `ok  	github.com/kandev/kandev/internal/planfiles/format	1.736s`
- `(cd apps/backend && go test ./internal/planfiles/format/ -run FuzzAppendNote -fuzz FuzzAppendNote -fuzztime 20s)`: `ok  	github.com/kandev/kandev/internal/planfiles/format	21.699s` (the fuzzer found a lone trailing CR on the last line; fixed, and its corpus entry is kept under `format/testdata/fuzz/FuzzAppendNote`)
- `make -C apps/backend lint`: `0 issues.`
- Extra: `(cd apps/backend && go test ./internal/planfiles/... -count=1)`: all packages `ok`.
