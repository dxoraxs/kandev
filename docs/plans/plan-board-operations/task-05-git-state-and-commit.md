---
id: "05-git-state-and-commit"
title: "Git state and commit"
status: done
wave: 5
depends_on: ["02-config-and-settings"]
plan: "plan.md"
requirements:
  - REQ-TASKS-PLAN-BOARD-OPS-008
acceptance_criteria:
  - AC-TASKS-PLAN-BOARD-OPS-008.2
  - AC-TASKS-PLAN-BOARD-OPS-008.3
  - AC-TASKS-PLAN-BOARD-OPS-008.4
system_design:
  - ../../specs/tasks/system-design/plan-board-operations.md
---
# Task 05: Git state and commit

## Summary

Add the `gitstate` subpackage, the status and commit endpoints, and the git block of the settings section. The `uncommitted` card flag itself is delivered by task 09.

## In scope

- `gitstate.Dirty`, `Busy`, `CommitPaths` on `subproc.NewGitCommand` with literal pathspecs.
- `Service.GitStatus`, `Service.CommitPlans`, `GET /git-status`, `POST /commit`, metric `plan_files_commit_total`.
- `pass` keeps the per-repository dirty set for later consumers (a field on the pass, unused by projection yet).
- Web: clients, `usePlanGitStatus`, the git block and commit dialog of UI-02, copy in every locale.

## Out of scope

- Pushing; committing files that are not plan files or the plan index.
- The card flag (task 09).

## Acceptance

- With one modified plan file, one untracked plan file, an unrelated modified file, and an unrelated staged file, the commit contains exactly the two plan files; the unrelated file stays modified and the staged file stays staged.
- A repository with `MERGE_HEAD` returns 409 `repository_busy`; a rejecting pre-commit hook returns 422 `commit_failed` with output, and `git status --porcelain` is identical before and after.
- A local repository path that is not a git working tree yields an empty file list and no error.

## ASCII UI preview

UI-02: git block. Full preview: [plan.md](plan.md#ascii-ui-preview). Covers AC-008.2.

```text
Uncommitted plan files
  dmhive        3 files      [ Commit plan files ]
  kandev        0 files
```

## Verification

```bash
(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)
make -C apps/backend lint
(cd apps && pnpm --filter @kandev/web test -- components/settings/plan-files-git.test.tsx)
(cd apps && pnpm --filter @kandev/web test -- lib/api/domains/plan-files-api.test.ts)
(cd apps/web && pnpm run typecheck)
(cd apps/web && pnpm run i18n:check)
(cd apps && pnpm --filter @kandev/web lint)
```

## Files likely touched

- apps/backend/internal/planfiles/gitstate/gitstate.go, gitstate_test.go (new)
- apps/backend/internal/planfiles/git_ops.go, git_ops_test.go, handlers_ops.go (new)
- apps/backend/internal/planfiles/sync_pass.go, metrics.go
- `apps/web/lib/api/domains/plan-files-api.ts`
- apps/web/hooks/domains/plans/use-plan-git-status.ts (new)
- apps/web/components/settings/plan-files-git.tsx, plan-files-git.test.tsx (new)
- `apps/web/src/locales/*/planFiles.json`

## Dependencies

Task 02

## Risks

- Tests create real repositories in `t.TempDir()`; set `user.name`, `user.email`, and `commit.gpgsign=false` locally in each so they do not depend on the machine config.
- `git status -z` output for renames has two paths per record; parse it by record, not by line.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/tasks/requirements/plan-board-operations.md) and [system design](../../specs/tasks/system-design/plan-board-operations.md), the sections mapped to the requirements above.
- Existing patterns: `internal/planfiles` tests and harnesses (`sync_harness_test.go`, `writeback_harness_test.go`, `fake_tasks_test.go`); web patterns in `components/settings/plan-files-*.tsx`.

## Results

Backend: `gitstate.Dirty`, `Busy`, `CommitPaths` (all through `subproc.NewGitCommand` with
`RunGit*Class`, `--literal-pathspecs`, `--` before paths), `Service.GitStatus`,
`Service.CommitPlans`, `GET /git-status`, `POST /commit`, metric `plan_files_commit_total`,
and `pass.dirty` filled once per repository per pass. Web: API clients, `usePlanGitStatus`,
the git block and commit dialog (drawer on phones) in the settings section, copy in every locale.

| Command | Result |
| --- | --- |
| `(cd apps/backend && go test ./internal/planfiles/... -count=1 -race)` | `ok  github.com/kandev/kandev/internal/planfiles/scan  1.501s` (all four packages ok) |
| `make -C apps/backend lint` | `0 issues.` |
| `(cd apps && pnpm --filter @kandev/web test -- components/settings/plan-files-git.test.tsx)` | `Tests  10 passed (10)` |
| `(cd apps && pnpm --filter @kandev/web test -- lib/api/domains/plan-files-api.test.ts)` | `Tests  15 passed (15)` |
| `(cd apps/web && pnpm run typecheck)` | `tsc --noEmit`, no errors |
| `(cd apps/web && pnpm run i18n:check)` | `no non-JSX copy - 3662 guarded file(s) checked.` (all checks pass) |
| `(cd apps && pnpm --filter @kandev/web lint)` | `eslint --max-warnings 0`, no findings |

Also run: `gofmt -l apps/backend/internal/planfiles` (no output),
`(cd apps/backend && golangci-lint run ./internal/planfiles/...)` (`0 issues.`),
`pnpm run i18n:ratchet` (`19 added + 20 modified file(s) clean`),
`go test ./internal/common/subproc/` (`ok`, includes the raw git guard), and the web suites
`components/settings`, `hooks/domains/plans`, `plan-files-api.test.ts` (238 files, 1589 tests passed).
