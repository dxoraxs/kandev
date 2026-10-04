---
id: "01-plan-file-format"
title: "Plan file format"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-FILES-001
  - REQ-TASKS-PLAN-FILES-003
acceptance_criteria:
  - AC-TASKS-PLAN-FILES-001.1
  - AC-TASKS-PLAN-FILES-001.2
  - AC-TASKS-PLAN-FILES-001.3
  - AC-TASKS-PLAN-FILES-001.4
  - AC-TASKS-PLAN-FILES-003.4
system_design:
  - ../../specs/tasks/system-design/repository-plan-files.md
---

# Task 01: Plan file format

## Summary

Add the pure `planfiles/format` package: parse a plan file into typed fields,
body, hash, and parse errors, and edit frontmatter keys without changing any
other byte.

## In scope

- `Parse`, `PlanFile`, board status and priority enums, `ContentHash`.
- Title fallback (frontmatter, first `# ` heading, file stem).
- `SetKeys` with insertion before the closing `---` and
  `ErrUnsupportedValueShape` for block scalars, flow collections, and
  multi-line values.
- CRLF and LF files; a UTF-8 BOM before line 1.

## Out of scope

- File I/O, scanning, and task logic.

## Acceptance

- A file without frontmatter or without `board` returns `ok == false`; every
  invalid value produces a named parse error while valid fields still parse.
- `SetKeys` output equals the input byte for byte outside the edited or
  inserted line, including comments, key order, line endings, and the body.
- Fuzz test over random frontmatter never panics and never changes the body.

## Verification

```bash
cd apps/backend && go test ./internal/planfiles/format/... -run . -count=1
cd apps/backend && go test ./internal/planfiles/format/... -run '^$' -fuzz FuzzSetKeys -fuzztime 30s
cd apps/backend && golangci-lint run ./internal/planfiles/...
```

## Files likely touched

- `apps/backend/internal/planfiles/format/parse.go`
- `apps/backend/internal/planfiles/format/setkeys.go`
- `apps/backend/internal/planfiles/format/parse_test.go`
- `apps/backend/internal/planfiles/format/setkeys_test.go`

## Dependencies

None.

## Risks

- YAML that `yaml.v3` accepts but the line editor cannot locate (indented
  top-level keys, anchors). Refuse with `ErrUnsupportedValueShape` rather than
  guessing.
