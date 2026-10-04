import { describe, expect, it } from "vitest";
import { produce } from "immer";
import type { Draft } from "immer";
import { hydrateState } from "./hydrator";
import { defaultState } from "@/lib/state/default-state";
import type { AppState } from "@/lib/state/store";

const METADATA = {
  card_display: { date: "2026-10-05", progress: { done: 1, total: 3 } },
};
const EXPECTED = {
  date: { iso: "2026-10-05", kind: "due" },
  progress: { done: 1, total: 3 },
};

function hydrate(state: unknown): AppState {
  return produce(structuredClone(defaultState) as AppState, (draft: Draft<AppState>) => {
    hydrateState(draft, state as Partial<AppState>);
  });
}

describe("hydrateState card display hints", () => {
  it("derives cardDisplay for boot-payload kanban tasks", () => {
    const result = hydrate({
      kanban: { tasks: [{ id: "t1", updatedAt: "2026-10-01T00:00:00Z", metadata: METADATA }] },
    });
    expect(result.kanban.tasks[0].cardDisplay).toEqual(EXPECTED);
  });

  it("derives cardDisplay for boot-payload multi-workflow snapshot tasks", () => {
    const result = hydrate({
      kanbanMulti: {
        snapshots: {
          wf1: {
            workflowId: "wf1",
            workflowName: "W",
            steps: [],
            tasks: [{ id: "t1", updatedAt: "2026-10-01T00:00:00Z", metadata: METADATA }],
          },
        },
      },
    });
    expect(result.kanbanMulti.snapshots.wf1.tasks[0].cardDisplay).toEqual(EXPECTED);
  });

  it("leaves cardDisplay undefined when metadata carries no valid hints", () => {
    const result = hydrate({
      kanban: { tasks: [{ id: "t1", updatedAt: "2026-10-01T00:00:00Z", metadata: {} }] },
    });
    expect(result.kanban.tasks[0].cardDisplay).toBeUndefined();
  });
});
