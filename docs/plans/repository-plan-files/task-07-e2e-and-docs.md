---
id: "07-e2e-and-docs"
title: "E2E and public documentation"
status: done
wave: 5
depends_on: ["05-write-back", "06-settings-section"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-FILES-002
  - REQ-TASKS-PLAN-FILES-003
  - REQ-TASKS-PLAN-FILES-005
acceptance_criteria:
  - AC-TASKS-PLAN-FILES-002.2
  - AC-TASKS-PLAN-FILES-002.3
  - AC-TASKS-PLAN-FILES-003.1
  - AC-TASKS-PLAN-FILES-005.1
  - AC-TASKS-PLAN-FILES-005.3
  - AC-TASKS-PLAN-FILES-005.5
system_design:
  - ../../specs/tasks/system-design/repository-plan-files.md
---

# Task 07: E2E and public documentation

## Summary

Prove the end-to-end flow in a browser on desktop and phone, and document the
plan file format and settings for users. Use `/e2e` and `/docs-maintainer`.

## In scope

- Backend fixture started with `KANDEV_FEATURES_PLAN_FILES=true` for these specs.
- Fixture: a local repository in the test temp directory with a plan file,
  registered in the e2e workspace.
- Desktop flow: enable, create Plans board, Sync now, task appears; edit file,
  Sync now, title updates; drag to Done, file contains `board: done`.
- Phone flow: settings layout and Sync now.
- `docs/public/plan-files.md` (format table, statuses, write-back rules,
  handoff steps, adoption via `external_id`, limits) and navigation entry.

## Out of scope

- Migration guides for specific projects.

## Acceptance

- Both specs pass with one worker per shard.
- The public page documents every frontmatter key and status from the
  requirement, and `docs/public` checks pass.

## Verification

```bash
cd apps/web && pnpm e2e:run --host --shards 1 --project chromium tests/settings/plan-files.spec.ts
cd apps/web && pnpm e2e:run --host --shards 1 --project mobile-chrome tests/settings/mobile-plan-files.spec.ts
```

## Files likely touched

- `apps/web/e2e/tests/settings/plan-files.spec.ts`
- `apps/web/e2e/tests/settings/mobile-plan-files.spec.ts`
- `apps/web/e2e/fixtures/` (local repository helper, if none fits)
- `docs/public/plan-files.md`, `docs/public/meta.json`

## Dependencies

Tasks 05 and 06.

## Risks

- Drag-and-drop on the board is timing sensitive; reuse the existing kanban
  drag helper.
