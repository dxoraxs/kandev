import { type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";

const getWaitingOwner = vi.fn();
const wsHandlers = new Map<string, () => void>();

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getWaitingOwner: (...a: unknown[]) => getWaitingOwner(...a),
}));
vi.mock("@/lib/ws/connection", () => ({
  getWebSocketClient: () => ({
    on: (type: string, handler: () => void) => {
      wsHandlers.set(type, handler);
      return () => wsHandlers.delete(type);
    },
  }),
}));

import { useWaitingOwner } from "./use-waiting-owner";

const ITEM = {
  workspace_id: "ws-1",
  workspace_name: "kandev",
  task_id: "t-1",
  title: "Plan",
  repository_name: "kandev",
  rel_path: "docs/plans/a.md",
  date: "",
  executor: "",
  priority: "medium",
};

function wrapper(planFiles: boolean) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <StateProvider initialState={{ features: { ...defaultState.features, planFiles } }}>
        {children}
      </StateProvider>
    );
  };
}

function visible(state: DocumentVisibilityState) {
  Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
}

describe("useWaitingOwner", () => {
  beforeEach(() => {
    wsHandlers.clear();
    visible("visible");
    getWaitingOwner.mockReset().mockResolvedValue({ items: [ITEM], failed_workspaces: [] });
  });
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("loads the list and the failed workspaces while the feature is on", async () => {
    getWaitingOwner.mockResolvedValue({
      items: [ITEM],
      failed_workspaces: [{ workspace_id: "ws-2", workspace_name: "dm" }],
    });
    const { result } = renderHook(() => useWaitingOwner(), { wrapper: wrapper(true) });
    expect(result.current.status).toBe("loading");
    await waitFor(() => expect(result.current.status).toBe("ready"));
    expect(result.current.items).toEqual([ITEM]);
    expect(result.current.failedWorkspaces).toEqual([
      { workspace_id: "ws-2", workspace_name: "dm" },
    ]);
  });

  it("reads nothing while the feature is off", async () => {
    const { result } = renderHook(() => useWaitingOwner(), { wrapper: wrapper(false) });
    await act(async () => {
      await Promise.resolve();
    });
    expect(getWaitingOwner).not.toHaveBeenCalled();
    expect(result.current.items).toEqual([]);
  });

  it("reports an error and keeps the last list, then recovers on reload", async () => {
    const { result } = renderHook(() => useWaitingOwner(), { wrapper: wrapper(true) });
    await waitFor(() => expect(result.current.status).toBe("ready"));
    getWaitingOwner.mockRejectedValueOnce(new Error("down"));
    await act(async () => {
      result.current.reload();
    });
    await waitFor(() => expect(result.current.status).toBe("error"));
    expect(result.current.items).toEqual([ITEM]);
    await act(async () => {
      result.current.reload();
    });
    await waitFor(() => expect(result.current.status).toBe("ready"));
  });

  it("refetches when the window regains focus", async () => {
    renderHook(() => useWaitingOwner(), { wrapper: wrapper(true) });
    await waitFor(() => expect(getWaitingOwner).toHaveBeenCalledTimes(1));
    await act(async () => {
      window.dispatchEvent(new Event("focus"));
    });
    await waitFor(() => expect(getWaitingOwner).toHaveBeenCalledTimes(2));
  });

  it("refetches once for a burst of task events", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    renderHook(() => useWaitingOwner(), { wrapper: wrapper(true) });
    await waitFor(() => expect(getWaitingOwner).toHaveBeenCalledTimes(1));
    expect([...wsHandlers.keys()].sort()).toEqual([
      "task.created",
      "task.deleted",
      "task.state_changed",
      "task.updated",
    ]);
    act(() => {
      wsHandlers.get("task.updated")?.();
      wsHandlers.get("task.state_changed")?.();
      wsHandlers.get("task.updated")?.();
    });
    expect(getWaitingOwner).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2500);
    });
    expect(getWaitingOwner).toHaveBeenCalledTimes(2);
  });

  it("ignores a response that arrives after a newer read", async () => {
    let resolveFirst: (v: unknown) => void = () => {};
    getWaitingOwner
      .mockReturnValueOnce(new Promise((r) => (resolveFirst = r)))
      .mockResolvedValueOnce({ items: [{ ...ITEM, title: "Newer" }], failed_workspaces: [] });
    const { result } = renderHook(() => useWaitingOwner(), { wrapper: wrapper(true) });
    await act(async () => {
      result.current.reload();
    });
    await waitFor(() => expect(result.current.items[0]?.title).toBe("Newer"));
    await act(async () => {
      resolveFirst({ items: [{ ...ITEM, title: "Stale" }], failed_workspaces: [] });
    });
    expect(result.current.items[0]?.title).toBe("Newer");
  });
});
