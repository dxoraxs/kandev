---
status: current
system: tasks
requirements:
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-001
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-002
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-003
  - REQ-TASKS-TASK-DESCRIPTION-VIEW-004
---

# Task Description View System Design

## Purpose and boundaries

This design moves the sessionless content defined by
[task open without agent](task-open-without-agent.md) out of the chat panel
into a dedicated description panel, and reduces the unassigned notice to a
corner element. The backend ensure outcome, the `unassigned` ensure status,
`SessionlessTaskContext`, and `NewSessionDialog` are reused unchanged.

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| `REQ-TASKS-TASK-DESCRIPTION-VIEW-001` | [Description panel](#description-panel), [Description document](#description-document) |
| `REQ-TASKS-TASK-DESCRIPTION-VIEW-002` | [Session tab transition](#session-tab-transition) |
| `REQ-TASKS-TASK-DESCRIPTION-VIEW-003` | [Corner notice](#corner-notice) |
| `REQ-TASKS-TASK-DESCRIPTION-VIEW-004` | [Phone composition](#phone-composition) |

## Description document

`TaskDescriptionDocument` (`apps/web/components/task/sessionless-task-view.tsx`)
renders `useTask(taskId)?.description`
through `MemoizedMarkdown` inside a wrapper carrying the global `markdown-body`
class (`app/globals.css`), which styles headings, lists, block quotes, rules,
tables, and code. The current wrapper lacks that class, which is why headings
and list markers are lost. An empty description keeps the
`task:sessionlessNoDescription` placeholder. The document is the scroll
container of its view (`h-full min-h-0 overflow-y-auto`), so a long
description scrolls to its last line (AC-001.4).

`resolveSessionlessTaskView` (`sessionless-task-view-model.ts`) keeps deciding
the description text and the corner state (`none`, `preparing`, `unassigned`).

## Description panel

A new Dockview panel `task-description`:

- `PANEL_REGISTRY` entry in `lib/state/layout-manager/constants.ts`:
  component `task-description`, title key `task:panelDescription`, default
  tab component. Added to `STRUCTURAL_COMPONENTS`, `KNOWN_PANEL_IDS`,
  `RENDERABLE_COMPONENT_NAMES`, `PANEL_RENDERERS`
  (`dockview-panel-content.tsx`), and the parallel map in
  `dockview-shared.tsx`.
- Its renderer, `TaskDescriptionPanel` (`task-description-panel.tsx`), reads
  the active task. While the task has no session it renders the sessionless
  view (document plus [corner notice](#corner-notice)); otherwise only the
  document.
- `dockview-add-panel-items.tsx` lists it through the store action
  `addDescriptionPanel`, so any task can open it (AC-001.2).

Sessionless layout: the generic `chat` placeholder stays the slot of a
sessionless task, because restore, hand-off, and safety-net paths depend on
it. `TaskChatPanel` renders the description document and corner notice in that
slot and hides the composer, and `resolveChatPanelTitle` names the tab
`task:panelDescription` while `useSessionlessTaskView` reports the task as
sessionless. No "Agent" tab exists until a session does (AC-001.1).

The board preview (`PreviewNoSessionsState` in `preview-session-tabs.tsx`)
renders `TaskDescriptionDocument` and the corner notice in its sessionless
slot; its error branch is unchanged (AC-001.3).

## Session tab transition

`runAutoSessionTabEffect` (`dockview-session-tabs.ts`) records, in
`sessionlessTaskIdRef`, a task observed with a loaded, empty session list.
When the first session panel of that task is added into the placeholder's
group, the effect inserts an inactive `task-description` panel at index 0 of
the same group before it removes the `chat` placeholder. The new session panel
stays active, so "Start agent", a board hand-off, or another client's session
all produce an active Agent tab beside a kept Description tab without a reload
(AC-002.1, AC-002.2). A task that already had a session when the page opened
gets no Description tab. `ensureSessionTabPrecedesNonSessionTabs` treats
`task-description` as a leading tab, so session tabs never jump ahead of it.

Any other host that renders a chat panel without a session (a restored
layout, the tablet layout) goes through the same `TaskChatPanel` branch.

## Corner notice

`UnassignedCornerNotice` replaces the full-width `Alert`:

- Desktop and tablet: a compact neutral card (`border bg-card text-xs`, info
  icon, never `destructive`) floated to the top-right of the description
  document (`float-right ml-4 mb-3 max-w-72`), so the document starts at the
  top and its text wraps around the card instead of being covered. It shows
  the title `task:noAgentProfileConfigured`, a small primary "Start agent"
  button opening `NewSessionDialog`, and an info control (accessible name
  `task:unassignedNoticeDetails`) that discloses
  `task:noAgentProfileConfiguredDetail` and the workspace settings link
  (AC-003.1, AC-003.2). Below the container width where a float leaves too
  little text width, the card takes the full width above the document.
- `preparing`: the same corner shows the existing `GridSpinner` with
  `task:preparingWorkspace2` (AC-003.3).
- No Retry, no raw error text; real ensure errors keep the page-level banner.

## Phone composition

- `buildMobileNavItems` (`components/task/mobile/session-mobile-bottom-nav.tsx`)
  takes a `sessionless` flag. While true, the first item keeps the internal
  panel id `chat` but reads `task:panelDescription` with a document icon.
- `MobileChatPanelContent` (`session-mobile-layout.tsx`) renders
  `TaskDescriptionDocument` and the corner notice without
  `MobileSessionsPicker` and without `TaskChatPanel` while sessionless; when a
  session appears it renders the picker and `TaskChatPanel` as today
  (AC-004.1).
- The phone corner notice is a compact control (icon and short title, at
  least 44 px tall) in the document's top-right corner. It opens a bottom
  `Drawer` holding the explanation, "Start agent", and the settings link as
  stacked full-width 44 px actions (AC-004.2).

## Copy

New keys in all locales: `task:panelDescription` ("Description") and
`task:unassignedNoticeDetails` (accessible name of the info control). Existing
keys are reused for the title, detail, actions, placeholder, and preparing
status.

## Testing

- Unit: `runAutoSessionTabEffect` keeps a Description tab when the first
  session replaces a sessionless placeholder and adds none for a task that
  had a session at open; the add-panel menu opens the Description panel; the
  placeholder title reads Description while sessionless.
- Unit: `buildMobileNavItems` label for `sessionless` true and false.
- Unit: existing `sessionless-task-view-model.test.ts` stays green.
- E2E (`chromium`, `mobile-chrome`): sessionless task opens on Description
  with no Agent tab and no composer; headings and list items render as
  elements; corner notice does not overlap description text; "Start agent"
  adds an active Agent tab while Description remains; long description
  scrolls; phone drawer actions are 44 px.
