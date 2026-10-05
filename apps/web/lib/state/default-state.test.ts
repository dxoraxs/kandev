import { describe, expect, it } from "vitest";
import { defaultState, mergeInitialState } from "./default-state";
import type { HydrationState } from "./store";

describe("turn hydration state", () => {
  it("defaults and deep-merges loaded session markers", () => {
    const state = mergeInitialState({
      turns: {
        bySession: { "session-1": [] },
        activeBySession: {},
        loadedBySession: {},
      },
    } as unknown as HydrationState);

    expect(defaultState.turns.loadedBySession).toEqual({});
    expect(state.turns.loadedBySession).toEqual({ "session-1": true });
  });
});

describe("quick chat hydration state", () => {
  it("marks an empty boot snapshot ready for the active workspace", () => {
    const state = mergeInitialState({
      workspaces: { items: [], activeId: "workspace-1" },
      quickChat: { sessions: [] },
    } as unknown as HydrationState);

    expect(state.quickChat.selectionReadyByWorkspace).toEqual({ "workspace-1": true });
  });
});

describe("failed Inbox hydration state", () => {
  it("keeps the request revision map when older persisted state omits it", () => {
    const state = mergeInitialState({
      failedInbox: {
        byWorkspaceId: {},
        generationByWorkspaceId: {},
      },
    } as unknown as HydrationState);

    expect(state.failedInbox.readRevisionByWorkspaceId).toEqual({});
  });
});

describe("settings agent hydration", () => {
  it("normalizes fallback fields from boot-hydrated profiles", () => {
    const state = mergeInitialState({
      settingsAgents: {
        items: [
          {
            profiles: [
              {
                id: "explicit-profile",
                fallback_model: "provider/model",
                auto_fallback: false,
              },
              {
                id: "automatic-profile",
                fallback_model: "",
                auto_fallback: true,
              },
            ],
          },
        ],
      },
    } as unknown as HydrationState);

    expect(state.settingsAgents.items[0]?.profiles).toMatchObject([
      {
        id: "explicit-profile",
        fallbackModel: "provider/model",
        autoFallback: false,
      },
      {
        id: "automatic-profile",
        fallbackModel: "",
        autoFallback: true,
      },
    ]);
  });
});

describe("card display hydration state", () => {
  const metadata = { card_display: { executor: { name: "Claude", kind: "agent" } } };
  const executor = { name: "Claude", kind: "agent" };

  it("derives card hints for boot kanban tasks and workflow snapshots", () => {
    const state = mergeInitialState({
      kanban: { tasks: [{ id: "task-1", metadata }] },
      kanbanMulti: { snapshots: { "workflow-1": { tasks: [{ id: "task-2", metadata }] } } },
    } as unknown as HydrationState);

    expect(state.kanban.tasks[0]?.cardDisplay).toEqual({ executor });
    expect(state.kanbanMulti.snapshots["workflow-1"]?.tasks[0]?.cardDisplay).toEqual({ executor });
  });
});
