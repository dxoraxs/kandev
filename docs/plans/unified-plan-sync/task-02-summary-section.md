---
id: "02-summary-section"
title: "Plan file summary section"
status: pending
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-CARD-002
acceptance_criteria:
  - AC-TASKS-PLAN-CARD-002.1
  - AC-TASKS-PLAN-CARD-002.2
  - AC-TASKS-PLAN-CARD-002.3
system_design:
  - ../../specs/tasks/system-design/plan-file-card-facts.md
---

# Task 02: Summary section

## Summary

Add the workspace summary heading to the plan files configuration (store
column, API, validation), select the plan header plus the matching section
when building task descriptions, and add the field to the Plan files settings
section. Follow `/mobile-parity`.

## In scope

- `Config.SummaryHeading`, column added at store init, API field,
  `invalid_summary_heading`.
- Summary selection in `projectDescription` with code-fence awareness and the
  header line from the system design.
- Web: field in `plan-files-section.tsx` per UI-01, API type, i18n in English,
  pseudo, and the seven translated catalogs, no em dash.
- Extend `e2e/tests/settings/plan-files.spec.ts` and
  `mobile-plan-files.spec.ts`: set the heading, save, reload, value kept; on
  phone the input's rendered height is at least 44px.

## Out of scope

- Card facts (Task 01).

## Acceptance

- Go tests: matching heading (case and whitespace), heading inside a code
  fence ignored, no match keeps the full body, 101-character value rejected,
  column added to an existing table.
- Desktop and phone E2E pass.

## ASCII UI preview

See [UI-01 in the plan](plan.md#ascii-ui-preview).

```text
| Summary section heading   [ Сейчас                ] |
```

Phone: label above a full-width 44px input.

## Verification

```bash
cd apps/backend && go test ./internal/planfiles/...
make -C apps/backend lint
cd apps && pnpm --filter @kandev/web test -- components/settings/plan-files-section.test.tsx
cd apps/web && pnpm run typecheck
cd apps/web && pnpm run i18n:check
cd apps && pnpm --filter @kandev/web lint
make build-web build-backend
cd apps/web && pnpm e2e:run -- tests/settings/plan-files.spec.ts tests/settings/mobile-plan-files.spec.ts
```

## Files likely touched

- `apps/backend/internal/planfiles/models.go`, `store.go`, `service_config.go`, `projection.go`
- `apps/web/components/settings/plan-files-section.tsx`, `lib/api/domains/plan-files-api.ts`
- `apps/web/src/locales/*/settings.json`
- The two E2E specs

## Dependencies

None. Run before or after Task 01, not in parallel.
