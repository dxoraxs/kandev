---
id: "01-parser-mapping-and-body"
title: "Parser, mapping, and title-first body"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-001
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-002
acceptance_criteria:
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-001.1
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-001.2
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-001.3
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-002.2
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-002.3
system_design:
  - ../../specs/tasks/system-design/kanban-card-display-hints.md
---

# Task 01: Parser, mapping, and title-first body

## Summary

Add the pure `card_display` parser and date-tone function, map the result
onto kanban tasks, and switch the card body to a two-line title without the
description preview.

## In scope

- `lib/kanban/card-display.ts` with `CardDisplayHints`,
  `cardDisplayFromMetadata`, `dateTagTone`, per the design's Contract and
  Parser sections; `lib/kanban/card-display.test.ts` covering every
  validation row and the today, tomorrow, past, and deferred tone cases.
- `cardDisplay` in `toKanbanTask`, the store task type, and the card `Task`
  type; parity assertion in `lib/kanban/map-task.test.ts` for HTTP and
  WebSocket shapes.
- `KanbanCardBody`: remove the description paragraph; title row aligned to
  the start. `CardTitle`: `line-clamp-2 break-words`. Update
  `kanban-card-content.test.tsx` and `kanban-card-title.test.tsx`.

Out of scope: the hint row components (02), docs and E2E (03).

## ASCII UI preview

See [UI-01](plan.md#ui-01-desktop-card-on-a-plan-board): this work order
delivers the title and the absent description line; the hint row is empty
until 02.

## Acceptance

- A malformed `card_display` yields `undefined` or only its valid fields;
  no exception for any JSON value.
- A card renders no description text and a title clamped at two lines with
  the hover disclosure still available when clipped.

## Verification

From `apps/web`:

```bash
pnpm vitest run lib/kanban/card-display.test.ts lib/kanban/map-task.test.ts components/kanban-card-content.test.tsx components/kanban-card-title.test.tsx
pnpm run typecheck
```

## Likely files

- `apps/web/lib/kanban/card-display.ts` (new), `card-display.test.ts` (new)
- `apps/web/lib/kanban/map-task.ts`, `map-task.test.ts`
- `apps/web/lib/state/slices/kanban/types.ts`
- `apps/web/components/kanban-card-types.ts`
- `apps/web/components/kanban-card-content.tsx`, `kanban-card-title.tsx` and their tests

## Risks

Existing tests that assert the description preview on cards must be updated,
not deleted; search `components` and `e2e` for `line-clamp-1` description
assertions and `task.description` card expectations.
