---
created: 2026-10-05
status: planned
requirements:
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-001
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-002
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-003
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-004
system_design:
  - ../../specs/tasks/system-design/kanban-card-display-hints.md
legacy_specs: []
---

# Implementation Plan: Kanban Card Display Hints

## Overview

Kanban cards become title-first: the title wraps to two lines and the
description preview is removed. A documented `metadata.card_display` object
lets external writers show a date tag, an executor badge, and a progress
chip.

Requirements: [kanban-card-display-hints](../../specs/tasks/requirements/kanban-card-display-hints.md).
Design: [kanban-card-display-hints](../../specs/tasks/system-design/kanban-card-display-hints.md).

## Scope

- Pure parser and tone function, store mapping, card types.
- Card body change and the hint row with three elements, localized.
- Public contract documentation, desktop and phone E2E, spec promotion.

Excluded: backend changes, UI editing of hints, date sorting or filtering,
and the synchronizer that writes the hints. The dmhive plan synchronizer
(`dmhive/apps/kandev-sync`) is updated in its own repository after this
package ships: plain title from the `# NN. Title` heading with no executor
prefix or number, `card_display` from the `Дата`, `Доска`, `Исполнитель`
and `Осталось` lines, predecessors through `PUT /api/v1/tasks/:id/dependencies`
(the closed step must complete the task for a dependency to resolve), and
no date line at the start of the description.

## Work orders

| # | Work order | Wave | Depends on |
| --- | --- | --- | --- |
| 01 | [Parser, mapping, and title-first body](task-01-parser-mapping-and-body.md) | 1 | none |
| 02 | [Hint row components](task-02-hint-row.md) | 2 | 01 |
| 03 | [Docs, E2E, and spec promotion](task-03-docs-e2e-promotion.md) | 3 | 02 |

The work orders are sequential: 02 renders the type 01 adds, and 03 tests the
rendered result.

## ASCII UI preview

### UI-01: Desktop card on a plan board

Entry: a board whose tasks carry `card_display`. State: default.

Before (from the rendered board):

```text
+------------------------------------------+
| [dxoraxs/dmhive]                    ...  |
| [Claude] 06a Удаление...  ↑              |
| Ждёт 5 октября > Файл:...    <- muted     |
+------------------------------------------+
```

After:

```text
+------------------------------------------+
| [dxoraxs/dmhive]                    ...  |
| Удаление Meta охватывает новый     ↑     |
| канал Instagram                          |
| (⏳ 5 окт) (☑ 1/3)                  (C)  |
+------------------------------------------+
  (⏳ 5 окт): date tag, warning tone today/tomorrow, danger when past
  (C): executor badge, person icon for kind "person"
  blocked badge from task dependencies keeps its current row below
```

Structural requirements: title up to two lines, no description line, hint
row under the title with the executor badge at the right edge. Spacing and
glyphs are illustrative; icons are Tabler icons.

### UI-02: Phone card

Same composition at 375 px; the hint row does not wrap and stays inside the
card. No hover on touch: the tooltip text is also the accessible name.

```text
+-------------------------------+
| Удаление Meta охватывает   ↑  |
| новый канал Instagram         |
| (⏳ 5 окт) (☑ 1/3)       (C)  |
+-------------------------------+
```

Mapping: UI-01 and UI-02 cover AC-001.1, AC-001.2, AC-003.1, AC-003.2,
AC-004.1, AC-004.2, AC-004.3; checked in work order 03.

## Verification strategy

Run from `apps/web`:

```bash
pnpm vitest run lib/kanban/card-display.test.ts lib/kanban/map-task.test.ts components/kanban-card-content.test.tsx components/kanban-card-title.test.tsx components/kanban-card-display-hints.test.tsx
pnpm run typecheck
pnpm run i18n:check
pnpm e2e:run -- e2e/tests/kanban/card-display-hints.spec.ts e2e/tests/kanban/mobile-card-display-hints.spec.ts
```

Lint from `apps`: `pnpm --filter @kandev/web lint`.

## Risks

- Removing the description line changes every board, not only synchronized
  ones. Accepted by the owner: the hover card and the description view keep
  the text.
- Two-line titles make cards taller; long columns show fewer cards per
  screen. Virtualized columns measure rows, so no fixed-height assumption
  breaks; checked by the existing large-column E2E.
- The date tone is computed at render and does not refresh at midnight.
