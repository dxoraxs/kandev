---
id: "03-dmhive-migration"
title: "Move dmhive to the built-in plan sync"
status: pending
wave: 2
depends_on: ["01-card-facts", "02-summary-section"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-CARD-001
  - REQ-TASKS-PLAN-CARD-002
acceptance_criteria:
  - AC-TASKS-PLAN-CARD-001.2
  - AC-TASKS-PLAN-CARD-001.3
  - AC-TASKS-PLAN-CARD-002.2
system_design:
  - ../../specs/tasks/system-design/plan-file-card-facts.md
---

# Task 03: dmhive migration

## Summary

Convert the dmhive plan cards to frontmatter, configure the built-in sync on
the existing "Планы dmhive" board so it adopts the 64 existing tasks, and
retire the custom synchronizer. Runs against the dmhive repository
(`/Users/valentin/Desktop/my-project/dmhive`) and the running Kandev
instance built with Tasks 01 and 02.

## In scope

1. Inventory before: list non-archived tasks of "Планы dmhive" with
   `external_id`, step, and ID; save it next to the task report.
2. Stop the custom synchronizer: `launchctl bootout` and `launchctl disable`
   for `gui/<uid>/com.dmhive.kandev-sync` (reversible with `enable` and
   `bootstrap`).
3. Convert each `docs/plans/NN[a-z]-*.md` card (not `00-README.md`) with a
   script that adds frontmatter and keeps the body otherwise byte-identical:
   - `board` from `**Доска:**` (очередь queued, в работе in_progress, ждёт
     владельца waiting_owner, ждёт внешнего waiting_external, отложено
     deferred, закрыто done, нет hidden); the `**Доска:**` line is removed.
   - `external_id: plan:<NN>`; `title: "<NN> <H1 text without the NN. prefix>"`.
   - `executor` from `**Исполнитель:**` when it starts with Claude or Codex;
     the human line stays.
   - `priority: high` and `order: <rank>` for cards in the
     `## Активный приоритет после ревью` table of `00-README.md`.
   - `depends_on` as file names resolved from `**Зависимости:**` card numbers.
   - `date` from `**Дата:** ДД.ММ.ГГГГ…`; the line stays as the human note.
4. Configure Kandev for the dmhive workspace: add the missing «Отдать Codex»
   handoff step; `PUT /api/v1/plan-files/config` with the board's existing
   steps (queued Очередь, in_progress В работе, waiting_owner Ждёт вас,
   waiting_external Ждёт внешнего, deferred Отложено, done Закрыто),
   directories `["docs/plans"]`, summary heading `Сейчас`; then sync.
5. Rewrite the handoff step prompts and the dmhive rules (`AGENTS.md` sections
   "Где записано" and "Карточка показывает текущий статус",
   `docs/plans/00-README.md` rules) from the `**Доска:**` line to the
   frontmatter keys.
6. Commit in dmhive with a path-scoped commit containing only this task's
   changes. For files that also hold the owner's uncommitted edits, stage the
   converted HEAD content, not the working file, and verify with
   `git diff --cached`.
7. Ask the owner before deleting `apps/kandev-sync`,
   `deploy/kandev/plans-workflow.yml`, and their tests: they carry the owner's
   uncommitted edits.

## Out of scope

- Converting `docs/superpowers/plans` (implementation plans, not cards).
- Pushing dmhive.

## Acceptance

- After sync, every task from the inventory keeps its ID and maps to the step
  its card says; no new task is created; the two `нет` cards stay
  archived; the pass reports no file errors.
- `launchctl print` shows the job disabled and not running.
- Open cards' task descriptions show the header and the `Сейчас` section.

## Verification

```bash
cd /Users/valentin/Desktop/my-project/dmhive && git diff --cached --stat
curl -s "http://localhost:38429/api/v1/plan-files/config?workspace_id=f55adcb1-03ce-493e-9f1a-9f2d8120b563"
launchctl print "gui/$(id -u)/com.dmhive.kandev-sync"
```

Plus the before/after inventory comparison script output.

## Files likely touched

- dmhive `docs/plans/*.md` (66 cards), `AGENTS.md`, `docs/plans/00-README.md`
- Kandev data: dmhive plan files config, the «Отдать Codex» step, handoff
  step prompts

## Dependencies

Tasks 01 and 02 on the running Kandev build. The display hints package should
land first so the executor and date render on cards.
