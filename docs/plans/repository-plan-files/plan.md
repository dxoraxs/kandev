---
created: 2026-10-04
status: implemented
requirements:
  - REQ-TASKS-PLAN-FILES-001
  - REQ-TASKS-PLAN-FILES-002
  - REQ-TASKS-PLAN-FILES-003
  - REQ-TASKS-PLAN-FILES-004
  - REQ-TASKS-PLAN-FILES-005
  - REQ-TASKS-PLAN-FILES-006
system_design:
  - ../../specs/tasks/system-design/repository-plan-files.md
legacy_specs: []
---

# Implementation Plan: Repository Plan Files

## Overview

Build `internal/planfiles` bottom-up: the pure format layer and the safe
scanner first, then persistence, the flag, the template, and the config API,
then the sync pass, then write-back, then the settings UI, and finally E2E and
public docs. Each layer is testable before the next one exists, and the
feature stays dark behind `features.planFiles` until the last work order.

Related decision:
[ADR 2026-10-04 repository plan files](../../decisions/2026-10-04-repository-plan-files-source-of-truth.md).

## Scope

### In scope

- Frontmatter parsing and byte-preserving key edits.
- Scanning local repositories with containment, size, and count limits.
- Config and per-task tables, runtime flag, built-in Plans template, HTTP API.
- Sync pass with projection, adoption, archive/unarchive, ordering, and the
  running-turn and handoff guards; 60-second poller.
- Write-back of status, priority, and order with hash compare-and-swap.
- Workspace settings section with mapping, directories, status, Sync now.
- E2E on desktop and phone; public documentation.

### Out of scope

- Migrating any existing plan collection to the frontmatter format.
- Agent guidance that sets `board` when plans are created or finished.
- Remote repositories, commits, filesystem notifications.

## Technical approach

### Format and scanner (Task 01, Task 02)

- `apps/backend/internal/planfiles/format`: `Parse(content []byte) (PlanFile,
  bool)`, `SetKeys(content []byte, values map[string]string) ([]byte, error)`,
  `ErrUnsupportedValueShape`, `ContentHash`. Uses `gopkg.in/yaml.v3` for reading
  only; edits are line-level inside the frontmatter block.
- `apps/backend/internal/planfiles/scan`: `ScanRepository(root string, dirs
  []string) ([]ScannedFile, []FileError)` and `WriteFile(root, rel string,
  content []byte, expectedHash string) error` with the containment, lstat,
  `maxPlanFileBytes = 1 << 20`, and `maxFilesPerDirectory = 1000` rules.

### Persistence, flag, template, config API (Task 03)

- `planfiles/store.go`: `plan_file_configs`, `plan_file_tasks`; registration in
  `internal/persistence/requiredstores/catalog.go`, store conformance owner
  actions, the workspace-deletion table registry, and
  `internal/backendapp/e2e_reset.go`.
- `runtimeflags/registry.go` entry `features.planFiles`, config field in
  `common/config/config.go`, `profiles.yaml` defaults, web
  `defaultFeatureFlags`.
- `apps/backend/config/workflows/plans.yml` and the template-step-to-status
  table in `planfiles/template.go`.
- `planfiles/service_config.go`, `planfiles/handlers.go`: config GET/PUT,
  board creation, workspace authorizer; prefix in
  `integrationWorkspacePrefixes`; service wiring in `backendapp/services.go`
  guarded by the flag.

### Sync pass (Task 04)

- `planfiles/service_sync.go`, `planfiles/projection.go`, `planfiles/poller.go`.
- Uses `task/service.Service` `CreateTask`, `GetTaskByExternalID`,
  `UpdateTask`, `MoveTaskWithOptions` (actor `system`), `ReorderStepTasks`
  (`admitted` band), and `HandoffService` archive/unarchive.
- Poller started in `backendapp/main.go` beside the workflow-sync poller with
  `addRuntimeCleanup`.
- `POST /api/v1/plan-files/sync` and pass status in the config response.

### Write-back (Task 05)

- `planfiles/writeback.go`: subscriber on `events.TaskMoved`,
  `events.TaskReordered`, `events.TaskUpdated`; divergence from `synced_*`;
  order computation; CAS write; notice; metrics.

### Settings UI (Task 06)

- `apps/web/lib/api/domains/plan-files-api.ts`,
  `apps/web/components/settings/plan-files-section.tsx` and small children
  (`plan-files-mapping.tsx`, `plan-files-status.tsx`), mounted in
  `apps/web/app/settings/workspace/workspace-workflows-client.tsx`.
- Locale keys in all seven catalogs plus English and pseudo.

## ASCII UI preview

`UI-01: Plan files section`, entry: Settings, Workspace, Workflows tab, below
Workflow sync. Structure is required; spacing and wording are illustrative.

Desktop, enabled, with one file error:

```text
+-------------------------------------------------------------------+
| Plan files                                        [x] Enabled     |
| Show plan files from this workspace's local repositories as tasks |
|                                                                   |
| Board      [ Plans                         v ]  [Create Plans board]|
|                                                                   |
| Status            Step                                            |
| queued            [ Queue              v ]                        |
| in_progress       [ In progress        v ]                        |
| waiting_owner     [ Waiting for you    v ]                        |
| waiting_external  [ Waiting external   v ]                        |
| deferred          [ Deferred           v ]                        |
| done              [ Done               v ]                        |
|                                                                   |
| Directories   docs/plans  (x)   docs/superpowers/plans  (x)  [+ Add]|
|                                                                   |
| Last sync 13:42  ok   created 2  updated 1  moved 0  archived 0   |
|   ! city_companion  docs/superpowers/plans/x.md   board value     |
|                                                    [Sync now] [Save]|
+-------------------------------------------------------------------+
```

Phone, same state (native settings stack, full-width controls):

```text
+---------------------------+
| < Workflows               |
| Plan files        [x]     |
| Board                     |
| [ Plans               v ] |
| [ Create Plans board    ] |
| queued                    |
| [ Queue               v ] |
| in_progress               |
| [ In progress         v ] |
| ...                       |
| Directories               |
| docs/plans            (x) |
| docs/superpowers/plans(x) |
| [ + Add directory       ] |
| Last sync 13:42 ok        |
| created 2 updated 1 ...   |
| ! x.md: board value       |
| [ Sync now              ] |
| [ Save                  ] |
+---------------------------+
```

States: disabled shows only the header and switch; no config yet shows the
switch and Create Plans board; a pass error shows the existing status-banner
error style. Maps to AC-TASKS-PLAN-FILES-005.1 through 005.6.

## Tests

| Criteria | Evidence |
| --- | --- |
| 001.1 to 001.4 | `planfiles/format/parse_test.go` |
| 003.4 | `planfiles/format/setkeys_test.go` (byte equality outside edited lines, comments kept, unsupported shapes refused) |
| 006.1 to 006.4 | `planfiles/scan/scan_test.go` (symlink file and directory, oversized, 1,001 files, missing root) |
| 005.2, 005.4 | `planfiles/service_config_test.go`, `planfiles/handlers_test.go` (authorization 404) |
| 002.1 to 002.8, 004.1 to 004.3 | `planfiles/service_sync_test.go` against a real SQLite task service and temp repositories |
| 003.1 to 003.3, 003.5, 003.6 | `planfiles/writeback_test.go` |
| 005.1, 005.3, 005.6 | `apps/web/components/settings/plan-files-section.test.tsx`, `apps/web/lib/api/domains/plan-files-api.test.ts` |

## E2E tests

| Flow | Criteria | File and project |
| --- | --- | --- |
| Enable, create Plans board, file appears, edit file, task updates | 002.2, 002.3, 005.1, 005.3 | `apps/web/e2e/tests/settings/plan-files.spec.ts`, `chromium` |
| Drag task to Done, file gets `board: done` | 003.1 | same file |
| Phone settings layout and Sync now | 005.5 | `apps/web/e2e/tests/settings/mobile-plan-files.spec.ts`, `mobile-chrome` |

## Work orders

- [x] [Task 01: Plan file format](task-01-plan-file-format.md)
- [x] [Task 02: Safe repository scanner](task-02-repository-scanner.md)
- [x] [Task 03: Store, flag, template, and config API](task-03-store-flag-config-api.md)
- [x] [Task 04: Sync pass and poller](task-04-sync-pass.md)
- [x] [Task 05: Board write-back](task-05-write-back.md)
- [x] [Task 06: Plan files settings section](task-06-settings-section.md)
- [x] [Task 07: E2E and public documentation](task-07-e2e-and-docs.md)

Waves: 1 = Task 01, Task 02 (parallel-safe); 2 = Task 03; 3 = Task 04;
4 = Task 05, Task 06 (parallel-safe); 5 = Task 07.

## Verification results

- `go test -trimpath ./internal/planfiles/... -count=1 -race`: ok (planfiles, format, scan); goleak clean.
- `go test -trimpath ./internal/backendapp/ -count=1` (with resolved `TMPDIR`): ok.
- `go vet` and `gofmt -l` on `internal/planfiles` and `internal/backendapp`: clean. `golangci-lint` was not available locally; CI enforces it.
- Web: plan-files section and API tests 16 passed; `typecheck`, `lint`, `i18n:check` clean.
- E2E: `plan-files.spec.ts` (chromium) and `mobile-plan-files.spec.ts` (mobile-chrome) passed.
- Public docs: `validate-public-docs` (48 pages) and its tests (62) passed.

Accepted deviations, recorded in the system design: write-back handlers only
enqueue work (synchronous bus delivery during a pass would deadlock); pass-side
order write-back only for an untouched step; file error reason `write_failed`.

## Risks

- Write-back touches user repositories. CAS on the content hash and the
  byte-preserving editor are the guards; `setkeys_test.go` must prove byte
  equality outside the edited lines.
- Event loops between sync moves and the write-back subscriber. The `synced_*`
  divergence rule is the guard; `writeback_test.go` asserts that a full pass
  produces zero writes.
- Plan bodies can reach hundreds of kilobytes and become task descriptions;
  board list payloads must not include descriptions (confirm in Task 04).
- The workspace-deletion and startup registries described in specs may not
  match code one to one; Task 03 registers the tables wherever the current code
  enforces completeness.
- E2E needs a local repository fixture with files on disk; Task 07 builds it in
  the test's temp directory.
