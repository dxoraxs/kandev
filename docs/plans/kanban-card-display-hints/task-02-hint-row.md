---
id: "02-hint-row"
title: "Hint row components"
status: planned
wave: 2
depends_on:
  - "01-parser-mapping-and-body"
plan: "plan.md"
requirements:
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-003
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-004
acceptance_criteria:
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-003.1
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-003.2
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-003.3
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-004.1
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-004.2
  - AC-TASKS-KANBAN-CARD-DISPLAY-HINTS-004.3
system_design:
  - ../../specs/tasks/system-design/kanban-card-display-hints.md
---

# Task 02: Hint row components

## Summary

Render the date tag, progress chip, and executor badge from
`task.cardDisplay` in a row under the card title.

## In scope

- `components/kanban-card-display-hints.tsx`: `DateTag`, `ProgressChip`,
  `ExecutorBadge`, `KanbanCardHintRow`, styled per the design's tone
  formulas; mounted in `KanbanCardBody` after the title row.
- `components/kanban-card-display-hints.test.tsx`: each element's text,
  tone class, accessible name, year suffix, `person` icon, and the row's
  absence without hints. Use a fixed `today` through fake timers.
- Locale keys `kanban:cardDateWaiting`, `cardDateDue`, `cardDateDeferred`,
  `cardExecutor`, `cardProgress_one`, `cardProgress_other` in English, the
  seven translated locales (zh-hant via `pnpm run i18n:zh-hant`), and the
  pseudo-locale; the new file appended to `i18nGuardFiles`.

Out of scope: docs and E2E (03).

## ASCII UI preview

[UI-01](plan.md#ui-01-desktop-card-on-a-plan-board) and
[UI-02](plan.md#ui-02-phone-card):

```text
| Удаление Meta охватывает новый     ↑     |
| канал Instagram                          |
| (⏳ 5 окт) (☑ 1/3)                  (C)  |
```

## Acceptance

- Each element renders only from its valid field, with the tone and
  accessible name the requirements define.
- Hovering an element does not start a drag or open the task.

## Verification

From `apps/web`:

```bash
pnpm vitest run components/kanban-card-display-hints.test.tsx components/kanban-card-content.test.tsx
pnpm run typecheck
pnpm run i18n:check
```

From `apps`: `pnpm --filter @kandev/web lint`.

## Likely files

- `apps/web/components/kanban-card-display-hints.tsx` (new) and its test
- `apps/web/components/kanban-card-content.tsx`
- `apps/web/src/locales/*/kanban.json`
- `apps/web/eslint.i18n.options.mjs`

## Risks

Do not compare a translated string with `===`; format dates with the active
i18n language, never at module scope.
