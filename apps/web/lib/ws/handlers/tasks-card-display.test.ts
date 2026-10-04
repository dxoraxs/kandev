import { describe, it, expect, vi } from "vitest";
import type { StoreApi } from "zustand";
import type { AppState } from "@/lib/state/store";
import { registerTasksHandlers } from "./tasks";

// A task.updated payload that carries metadata must refresh the parsed card
// display hints of an already cached task.

const WORKFLOW_ID = "wf1";
const TASK_ID = "t1";
const STEP_ID = "step1";

function makeStore(initial: Partial<AppState> = {}) {
  let state = {
    kanban: { workflowId: WORKFLOW_ID, steps: [], tasks: [] },
    kanbanMulti: { snapshots: {}, isLoading: false },
    tasks: {
      activeTaskId: null,
      activeSessionId: null,
      pinnedSessionId: null,
      lastSessionByTaskId: {},
    },
    taskSessionsByTask: { itemsByTaskId: {}, loadedByTaskId: {}, loadingByTaskId: {} },
    environmentIdBySessionId: {},
    setActiveSession: vi.fn(),
    setActiveSessionAuto: vi.fn(),
    removeTaskFromSidebarPrefs: vi.fn(),
    setTaskDeletedNotification: vi.fn(),
    ...initial,
  } as unknown as AppState;

  return {
    getState: () => state,
    setState: (updater: AppState | ((s: AppState) => AppState)) => {
      const next =
        typeof updater === "function" ? (updater as (s: AppState) => AppState)(state) : updater;
      state = { ...state, ...next };
    },
    subscribe: () => () => {},
    destroy: vi.fn(),
    getInitialState: vi.fn(),
  } as unknown as StoreApi<AppState> & { getState: () => AppState };
}

function makeMessage(payload: Record<string, unknown>) {
  return {
    id: "msg-1",
    type: "notification" as const,
    action: "task.updated" as const,
    payload,
  } as Parameters<NonNullable<ReturnType<typeof registerTasksHandlers>["task.updated"]>>[0];
}

function basePayload(overrides: Record<string, unknown> = {}) {
  return {
    task_id: TASK_ID,
    workflow_id: WORKFLOW_ID,
    workflow_step_id: STEP_ID,
    title: "Test",
    description: "",
    state: "TODO",
    is_ephemeral: false,
    ...overrides,
  };
}

describe("task.updated handler: card display hints", () => {
  it("re-derives cardDisplay from the payload metadata for a cached task", () => {
    const store = makeStore({
      kanban: {
        workflowId: WORKFLOW_ID,
        steps: [],
        tasks: [
          {
            id: TASK_ID,
            workflowId: WORKFLOW_ID,
            workflowStepId: STEP_ID,
            title: "Test",
            position: 0,
            metadata: { card_display: { date: "2030-01-01" } },
            cardDisplay: { date: { iso: "2030-01-01", kind: "due" } },
          },
        ],
      },
    });

    registerTasksHandlers(store)["task.updated"]!(
      makeMessage(basePayload({ metadata: { card_display: { date: "2020-01-01" } } })),
    );

    const task = store.getState().kanban.tasks.find((t) => t.id === TASK_ID);
    expect(task?.cardDisplay?.date?.iso).toBe("2020-01-01");
  });
});
