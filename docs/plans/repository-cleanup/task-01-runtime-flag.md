---
id: "01-runtime-flag"
title: "Repository cleanup runtime flag"
status: pending
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-WORKSPACES-REPO-CLEANUP-001
acceptance_criteria:
  - AC-WORKSPACES-REPO-CLEANUP-001.8
system_design:
  - ../../specs/workspaces/system-design/repository-cleanup.md
---

# Task 01: Repository cleanup runtime flag

## Summary

Register `features.repositoryCleanup` (`KANDEV_FEATURES_REPOSITORY_CLEANUP`)
through the `/runtime-feature-flags` checklist: registry entry with metadata
(feature, experimental, high risk, restart required), config field, profile
defaults off in `prod`, `dev`, and `e2e`, and the frontend feature key.

## In scope

- `runtimeflags/registry.go` entry and config read/apply.
- `profiles.yaml` defaults.
- Frontend `features` slice key `repositoryCleanup` and `useFeature` support.
- Localized label and description for Settings, System, Feature Toggles.

## Out of scope

- Any consumer of the flag.

## Acceptance

- Registry, profile, and frontend completeness tests pass with the new flag.
- The flag shows in Feature Toggles and defaults to off.

## Verification

```bash
cd apps/backend && go test ./internal/runtimeflags/... ./internal/profiles/...
cd apps && pnpm --filter @kandev/web test -- lib/state/slices/features
cd apps/web && pnpm run i18n:check
make -C apps/backend lint
```

## Files likely touched

- `apps/backend/internal/runtimeflags/registry.go`
- `apps/backend/internal/common/config/` (features struct)
- `profiles.yaml`
- `apps/web/lib/state/slices/features/types.ts` and its tests
- `apps/web/src/locales/*/` (feature toggle copy)

## Dependencies

None. Load `/runtime-feature-flags` before starting.
