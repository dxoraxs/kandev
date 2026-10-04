---
id: "02-repository-scanner"
title: "Safe repository scanner"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-FILES-002
  - REQ-TASKS-PLAN-FILES-006
acceptance_criteria:
  - AC-TASKS-PLAN-FILES-002.1
  - AC-TASKS-PLAN-FILES-006.1
  - AC-TASKS-PLAN-FILES-006.2
  - AC-TASKS-PLAN-FILES-006.4
system_design:
  - ../../specs/tasks/system-design/repository-plan-files.md
---

# Task 02: Safe repository scanner

## Summary

Add `planfiles/scan`: list and read `*.md` files directly inside configured
directories of one repository root, and write a file atomically with a hash
precondition, all inside the resolved root.

## In scope

- Directory validation (relative, no `..`), missing-directory skip,
  sorted listing, `maxFilesPerDirectory` truncation warning.
- `os.Lstat` regular-file check, `maxPlanFileBytes` limit, containment check
  after `EvalSymlinks`.
- `WriteFile` with expected-hash check, temp file in the same directory,
  `fsync`, rename, mode preserved; `ErrHashMismatch`.
- Typed `FileError` reasons matching the design's closed set.

## Out of scope

- Parsing and task logic.

## Acceptance

- Symlinked files, symlinked directories pointing outside the root, oversized
  files, and the 1,001st file are reported, never read.
- `WriteFile` refuses when the on-disk hash differs and leaves the file
  unchanged; on success no temp file remains.

## Verification

```bash
cd apps/backend && go test ./internal/planfiles/scan/... -count=1
cd apps/backend && golangci-lint run ./internal/planfiles/...
```

## Files likely touched

- `apps/backend/internal/planfiles/scan/scan.go`
- `apps/backend/internal/planfiles/scan/write.go`
- `apps/backend/internal/planfiles/scan/scan_test.go`
- `apps/backend/internal/planfiles/scan/write_test.go`

## Dependencies

None.

## Risks

- macOS `/var` to `/private/var` symlink in temp directories: compare
  resolved paths on both sides.
