---
status: current
system: tasks
requirements:
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-001
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-002
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-003
  - REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-004
---

# Kanban Card Display Hints System Design

## Purpose and boundaries

This design changes the kanban card body to a title-first layout and adds a
hint row rendered from a documented task metadata object. It is a frontend
change: task metadata is already accepted on create and update
(`metadata` on the task HTTP handlers), persisted, and carried by both the
HTTP task DTO and the `task.updated` WebSocket payload into the kanban store
(`toKanbanTask`, parity-tested in `lib/kanban/map-task.test.ts`). The backend
does not validate the object; the frontend parser is the validation boundary,
which keeps a malformed value from ever reaching rendering.

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-001` | [Card body](#card-body) |
| `REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-002` | [Contract](#contract), [Parser and mapping](#parser-and-mapping) |
| `REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-003` | [Date tag](#date-tag) |
| `REQ-TASKS-KANBAN-CARD-DISPLAY-HINTS-004` | [Executor badge and progress chip](#executor-badge-and-progress-chip), [Hint row](#hint-row) |

## Contract

`metadata.card_display` is an object. Every field is optional:

```json
{
  "card_display": {
    "date": "2026-10-05",
    "date_kind": "waiting",
    "executor": { "name": "Claude", "kind": "agent" },
    "progress": { "done": 1, "total": 3 }
  }
}
```

| Field | Valid value |
| --- | --- |
| `date` | `YYYY-MM-DD` that is a real calendar date. |
| `date_kind` | `waiting`, `due`, or `deferred`. Absent or other values read as `due`. |
| `executor.name` | String, trimmed, 1 to 40 characters after trimming. |
| `executor.kind` | `agent` or `person`. Absent or other values read as `agent`. |
| `progress.done`, `progress.total` | Integers, `0 <= done <= total`, `total > 0`. |

Each field is validated independently (AC-002.2). Writers merge the object
into the existing metadata; the task API stores metadata as given, so a
writer that replaces metadata must carry other keys forward. The public
contract is documented in `docs/public/automation-and-mcp.md` next to the
task API.

## Parser and mapping

`lib/kanban/card-display.ts` (new, pure):

- `type CardDisplayHints = { date?: { iso: string; kind: DateKind };
  executor?: { name: string; kind: "agent" | "person" };
  progress?: { done: number; total: number } }`.
- `cardDisplayFromMetadata(metadata: unknown): CardDisplayHints | undefined`
  returns `undefined` when no field is valid.
- `dateTagTone(iso, kind, today: Date): "neutral" | "warning" | "danger"`
  compares calendar days built with `new Date(y, m - 1, d)` (local time, no
  UTC shift). `deferred` maps `danger` to `warning` (AC-003.2).

`toKanbanTask` (`lib/kanban/map-task.ts`) adds
`cardDisplay: cardDisplayFromMetadata(source.metadata)`, following the
`isPRReviewFromMetadata` pattern. `cardDisplay` is added to the store task
type (`lib/state/slices/kanban/types.ts`) and the card `Task` type
(`components/kanban-card-types.ts`). Parsing once in the mapper keeps the
card render free of validation and makes both payload shapes agree.

## Card body

`KanbanCardBody` (`components/kanban-card-content.tsx`):

- Removes the description `<p>` (AC-001.2). `CardTitle` keeps passing the
  description to `TaskTitleHoverCard`.
- `CardTitle` (`components/kanban-card-title.tsx`) changes `line-clamp-1` to
  `line-clamp-2` with `break-words` (AC-001.1). `useIsTitleTruncated`
  already measures the vertical axis, so the hover disclosure still opens
  for a clipped title. The title row aligns items to the start so the
  priority and PR icons sit on the first line (AC-001.3).
- Renders `<KanbanCardHintRow task={task} />` between the title row and
  `KanbanCardRelationship`.

`KanbanCardBody` is shared by the board card, the drag preview
(`kanban-card-preview.tsx`) and the plugin slot wrapper, so all three change
together.

## Date tag

`DateTag` in `components/kanban-card-display-hints.tsx` (new):

- Pill: `h-5 rounded-full px-1.5 text-[11px] font-medium` with the tinted
  tone formula used by `BlockedBadge`: neutral `bg-muted text-foreground`,
  warning `bg-amber-500/15 text-amber-700 dark:text-amber-300`, danger
  `bg-red-500/15 text-red-600 dark:text-red-400`.
- Icons (Tabler): `waiting` `IconHourglass`, `due` `IconCalendarDue`,
  `deferred` `IconPlayerPause`.
- Short text: `Intl.DateTimeFormat(i18n.language, { day: "numeric",
  month: "short" })`, plus `year: "numeric"` when the year differs from
  today's (AC-003.1).
- Full text: `kanban:cardDateWaiting`, `kanban:cardDateDue`,
  `kanban:cardDateDeferred` with a `{{date}}` placeholder formatted with
  `dateStyle: "long"`. Used as `aria-label` and `title` (AC-003.3).
- `today` is read at render. A card does not re-render at midnight; the next
  store update or reload refreshes the tone. Accepted.

## Executor badge and progress chip

`ExecutorBadge`: 18 px circle. Kind `agent` shows the first grapheme of the
name uppercased (`Array.from(name)[0]`) on `bg-primary/15 text-primary`;
kind `person` shows `IconUser` on `bg-muted`. Accessible name and title:
`kanban:cardExecutor` with `{{name}}` (AC-004.1).

`ProgressChip`: `IconListCheck` plus `done/total` in the pill formula;
success tone `bg-emerald-500/15 text-emerald-700 dark:text-emerald-300`
when complete. Accessible name and title: `kanban:cardProgress` with `count`
= `total` and `done` (`_one`/`_other` plural forms) (AC-004.2).

## Hint row

`KanbanCardHintRow` renders nothing without `task.cardDisplay`. Otherwise a
`mt-1 flex items-center gap-1 min-w-0` row: date tag, progress chip, then the
executor badge pushed right with `ml-auto`. Elements are non-interactive
spans with `title`; they carry no pointer handlers, so a hover never starts a
drag or opens the task, and the card click target is unchanged (AC-004.3).
The row does not wrap; the date tag and chip have fixed content, so at
375 px the row fits beside the badge.

## Localization

New keys in the `kanban` namespace in all seven locales plus English and the
pseudo-locale: `cardDateWaiting`, `cardDateDue`, `cardDateDeferred`,
`cardExecutor`, `cardProgress_one`, `cardProgress_other`. Traditional Chinese
through `pnpm run i18n:zh-hant`. The new component file is appended to
`i18nGuardFiles`.

## Failure modes

| Case | Behavior |
| --- | --- |
| `card_display` is not an object | No hints. |
| One field invalid | That element is absent; others render. |
| `executor.name` longer than 40 characters | Field invalid, no badge. |
| Writer replaces metadata without `card_display` | Hints disappear on the next update. |

## Verification

- Unit: `lib/kanban/card-display.test.ts` (every validation row, tones at
  the today, tomorrow, and past boundaries, deferred mapping), mapper parity
  in `lib/kanban/map-task.test.ts`, component tests in
  `components/kanban-card-display-hints.test.tsx` and the existing
  `kanban-card-content` tests updated for the removed description.
- E2E: `e2e/tests/kanban/card-display-hints.spec.ts` and
  `e2e/tests/kanban/mobile-card-display-hints.spec.ts` create tasks with
  `card_display` through the API client and assert rendering, tooltips, the
  absent description, and live update after a metadata change.
