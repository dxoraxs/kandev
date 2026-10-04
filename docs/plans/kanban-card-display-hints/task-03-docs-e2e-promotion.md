---
id: "03-docs-e2e-promotion"
title: "Docs, E2E, and spec promotion"
status: planned
wave: 3
depends_on:
  - "02-hint-row"
plan: "plan.md"
requirements:
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-001
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-002
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-003
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-004
acceptance_criteria:
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-001.2
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-002.1
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-003.1
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-004.3
system_design:
  - ../../specs/tasks/system-design/kanban-card-display-hints.md
---

# Task 03: Docs, E2E, and spec promotion

## Summary

Document the `card_display` contract, prove the rendered result end to end
on desktop and phone, and promote the package.

## In scope

- `docs/public/automation-and-mcp.md`: a "Card display hints" section with
  the contract table and a create/update example (`/docs-maintainer`).
- `e2e/tests/kanban/card-display-hints.spec.ts`: a task created with
  `card_display` (date today, progress, executor) shows the three elements
  with their tooltips and no description text; `updateTaskMetadata` with a
  past date switches the tag to the danger tone without a reload; a task
  without hints shows none.
- `e2e/tests/kanban/mobile-card-display-hints.spec.ts`: the same card at
  375 px, the row inside the card bounds, no horizontal page scroll.
- Spec promotion: requirement to `active`, system design to `current`, plan
  to `implemented`, after checking the implementation against both.

## Acceptance

- Both E2E specs pass with the guarded runner.
- `python3 scripts/list-docs.py validate` and
  `python3 scripts/lint-spec-files.py --all` pass after promotion.

## Verification

From `apps/web`:

```bash
pnpm e2e:run -- e2e/tests/kanban/card-display-hints.spec.ts e2e/tests/kanban/mobile-card-display-hints.spec.ts
pnpm e2e:run -- e2e/tests/kanban/large-column-virtualization.spec.ts
```

From the repository root:

```bash
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
```

## Likely files

- `apps/web/e2e/tests/kanban/card-display-hints.spec.ts` (new)
- `apps/web/e2e/tests/kanban/mobile-card-display-hints.spec.ts` (new)
- `docs/public/automation-and-mcp.md`
- the requirement, system design, and plan frontmatter

## Risks

No fixed sleeps in E2E; wait on rendered state. Existing E2E that read a
card's description preview must move to the hover card or description view.
