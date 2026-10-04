---
id: "04-e2e-and-spec-reconciliation"
title: "E2E and spec reconciliation"
status: done
wave: 3
depends_on:
  - "02-description-panel-and-agent-tab"
  - "03-phone-composition"
plan: "plan.md"
requirements:
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-001
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-002
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-003
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-004
acceptance_criteria:
  - AC-TASKS-TASK-DESCRIPTION-VIEW-001.1
  - AC-TASKS-TASK-DESCRIPTION-VIEW-001.4
  - AC-TASKS-TASK-DESCRIPTION-VIEW-002.1
  - AC-TASKS-TASK-DESCRIPTION-VIEW-002.2
  - AC-TASKS-TASK-DESCRIPTION-VIEW-003.1
  - AC-TASKS-TASK-DESCRIPTION-VIEW-004.1
  - AC-TASKS-TASK-DESCRIPTION-VIEW-004.2
system_design:
  - ../../specs/tasks/system-design/task-description-view.md
---

# Task 04: E2E and spec reconciliation

## Summary

Update the two task-open-without-agent specs to the new composition and
reconcile the specifications once the behavior ships.

## In scope

- Desktop spec: Description tab active, no Agent tab, no composer; a `h2` and
  an `li` exist in the document; the notice's box does not intersect any
  description paragraph box; "Start agent" adds an active Agent tab while
  Description remains; long description scrolls to the Start agent control
  and the last paragraph.
- Phone spec: first nav item reads Description, no session picker, corner
  control and drawer actions are at least 44 px, no horizontal overflow.
- Update `@covers` tags to the new criteria.
- Remove the superseded criteria from
  `docs/specs/tasks/requirements/task-open-without-agent.md`, update its
  design, promote this requirement to `active` and design to `current`, set
  the plan to `implemented`.

## Acceptance

- Both specs pass on their projects in two consecutive runs.
- Spec validators pass.

## Verification

```bash
cd apps/web && pnpm e2e:run --host --shards 1 --project chromium tests/task/task-open-without-agent.spec.ts
cd apps/web && pnpm e2e:run --host --shards 1 --project mobile-chrome tests/task/mobile-task-open-without-agent.spec.ts
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
```

## Files likely touched

- `apps/web/e2e/tests/task/task-open-without-agent.spec.ts`
- `apps/web/e2e/tests/task/mobile-task-open-without-agent.spec.ts`
- `docs/specs/tasks/requirements/task-open-without-agent.md`,
  `docs/specs/tasks/system-design/task-open-without-agent.md`

## Dependencies

02 and 03.

## Risks

- Overlap assertions must compare the notice box with paragraph boxes, not
  with the whole document box, which always contains the float.

## Results

- Desktop spec: Description tab active, no session tab, no composer; `h2`,
  `li`, and `strong` render as elements; the notice's box intersects no text
  line box of the description; "Start agent" adds one active session tab while
  Description stays; the long description scrolls to its last paragraph with
  no horizontal overflow.
- Phone spec: first nav item reads Description, no sessions pill or composer,
  44 px corner control, drawer actions at least 44 px, no horizontal overflow.
- `pnpm e2e:run --host --shards 1 --project chromium tests/task/task-open-without-agent.spec.ts`:
  3 passed, twice. `... --project mobile-chrome tests/task/mobile-task-open-without-agent.spec.ts`:
  1 passed, twice.
- Superseded criteria removed from `task-open-without-agent.md`, its design
  trimmed to the ensure contract; `@covers` tags moved to the new criteria.
  This requirement is `active`, its design `current`, the plan `implemented`.
- `list-docs.py validate` and `lint-spec-files.py --all` pass.
