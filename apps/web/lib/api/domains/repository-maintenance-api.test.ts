import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MaintenanceTaskError, startRepositoryMaintenanceTask } from "./repository-maintenance-api";

const originalFetch = global.fetch;

function json(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

async function failure(promise: Promise<unknown>): Promise<MaintenanceTaskError> {
  try {
    await promise;
  } catch (err) {
    if (err instanceof MaintenanceTaskError) return err;
    throw err;
  }
  throw new Error("expected the start to be rejected");
}

describe("startRepositoryMaintenanceTask", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    global.fetch = fetchSpy as unknown as typeof fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("POSTs the kind to the repository's maintenance-tasks endpoint", async () => {
    fetchSpy.mockResolvedValueOnce(json({ task_id: "t1", session_id: "s1", existing: false }, 201));
    const res = await startRepositoryMaintenanceTask("repo/1", "repository_cleanup");
    const [url, init] = fetchSpy.mock.calls[0]! as [string, RequestInit];
    expect(url).toContain("/api/v1/repositories/repo%2F1/maintenance-tasks");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toEqual({ kind: "repository_cleanup" });
    expect(res).toEqual({ task_id: "t1", session_id: "s1", existing: false });
  });

  it("returns an existing task, with or without a session", async () => {
    fetchSpy.mockResolvedValueOnce(json({ task_id: "t1", session_id: "", existing: true }));
    await expect(startRepositoryMaintenanceTask("r", "plan_adaptation")).resolves.toEqual({
      task_id: "t1",
      session_id: "",
      existing: true,
    });
  });

  it.each(["no_agent_profile", "no_workflow", "repository_not_local"])(
    "maps a 409 %s to its typed reason",
    async (reason) => {
      fetchSpy.mockResolvedValueOnce(json({ reason }, 409));
      const err = await failure(startRepositoryMaintenanceTask("r", "plan_adaptation"));
      expect(err.reason).toBe(reason);
      expect(err.taskId).toBeUndefined();
    },
  );

  it("maps 404 kind_unavailable and 404 repository not found", async () => {
    fetchSpy.mockResolvedValueOnce(json({ reason: "kind_unavailable" }, 404));
    expect((await failure(startRepositoryMaintenanceTask("r", "plan_adaptation"))).reason).toBe(
      "kind_unavailable",
    );
    fetchSpy.mockResolvedValueOnce(json({ error: "repository not found" }, 404));
    expect((await failure(startRepositoryMaintenanceTask("r", "plan_adaptation"))).reason).toBe(
      "repository_not_found",
    );
  });

  it("maps other failures to failed and keeps the task id of a launch failure", async () => {
    fetchSpy.mockResolvedValueOnce(json({ error: "boom", task_id: "t9" }, 500));
    const err = await failure(startRepositoryMaintenanceTask("r", "plan_adaptation"));
    expect(err.reason).toBe("failed");
    expect(err.taskId).toBe("t9");

    fetchSpy.mockResolvedValueOnce(json({ error: "bad kind" }, 400));
    expect((await failure(startRepositoryMaintenanceTask("r", "plan_adaptation"))).reason).toBe(
      "failed",
    );
  });
});
