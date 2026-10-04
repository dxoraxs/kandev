import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../client";
import {
  createPlanFilesBoard,
  getPlanFilesConfig,
  getUnadaptedPlanFiles,
  putPlanFilesConfig,
  syncPlanFilesNow,
} from "./plan-files-api";

const originalFetch = global.fetch;
const WS = { workspaceId: "ws 1" };
const WS_PARAM = "workspace_id=ws%201";

function json(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("plan-files-api", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    global.fetch = fetchSpy as unknown as typeof fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("getPlanFilesConfig reads the config with an encoded workspace_id", async () => {
    fetchSpy.mockResolvedValueOnce(json({ workspace_id: "ws 1", enabled: true }));
    const cfg = await getPlanFilesConfig(WS);
    const url = fetchSpy.mock.calls[0]![0] as string;
    expect(url).toContain(`/api/v1/plan-files/config?${WS_PARAM}`);
    expect(cfg?.enabled).toBe(true);
  });

  it("getPlanFilesConfig resolves null on 404 and rethrows other errors", async () => {
    fetchSpy.mockResolvedValueOnce(json({ error: "plan files config not found" }, 404));
    await expect(getPlanFilesConfig(WS)).resolves.toBeNull();
    fetchSpy.mockResolvedValueOnce(json({ error: "boom" }, 500));
    await expect(getPlanFilesConfig(WS)).rejects.toBeInstanceOf(ApiError);
  });

  it("putPlanFilesConfig PUTs the raw status tokens and directories", async () => {
    fetchSpy.mockResolvedValueOnce(json({ enabled: true }));
    const payload = {
      enabled: true,
      workflow_id: "wf-1",
      status_steps: { queued: "s1", in_progress: "s2" },
      directories: ["docs/plans"],
    };
    await putPlanFilesConfig(payload, WS);
    const [url, init] = fetchSpy.mock.calls[0]! as [string, RequestInit];
    expect(url).toContain(`/api/v1/plan-files/config?${WS_PARAM}`);
    expect(init.method).toBe("PUT");
    expect(JSON.parse(init.body as string)).toEqual(payload);
  });

  it("putPlanFilesConfig sends the operation settings and omits absent ones", async () => {
    fetchSpy.mockImplementation(async () => json({ enabled: true }));
    const base = {
      enabled: true,
      workflow_id: "wf-1",
      status_steps: { queued: "s1" },
      directories: ["docs/plans"],
    };
    await putPlanFilesConfig(
      {
        ...base,
        executor_steps: { s9: "Claude" },
        notes_heading: "Owner notes",
        wake_on_date: false,
        stale_after_days: 0,
        index_file: "INDEX.md",
      },
      WS,
    );
    const sent = JSON.parse((fetchSpy.mock.calls[0]![1] as RequestInit).body as string);
    expect(sent).toMatchObject({
      executor_steps: { s9: "Claude" },
      notes_heading: "Owner notes",
      wake_on_date: false,
      stale_after_days: 0,
      index_file: "INDEX.md",
    });
    await putPlanFilesConfig(base, WS);
    const legacy = JSON.parse((fetchSpy.mock.calls[1]![1] as RequestInit).body as string);
    expect(Object.keys(legacy).sort()).toEqual(Object.keys(base).sort());
  });

  it("putPlanFilesConfig surfaces the backend validation message", async () => {
    fetchSpy.mockResolvedValueOnce(
      json({ error: 'invalid plan files config: status "done" needs a step' }, 400),
    );
    await expect(
      putPlanFilesConfig(
        { enabled: true, workflow_id: "w", status_steps: {}, directories: ["a"] },
        WS,
      ),
    ).rejects.toMatchObject({ status: 400, message: expect.stringContaining("needs a step") });
  });

  it("createPlanFilesBoard POSTs to /board", async () => {
    fetchSpy.mockResolvedValueOnce(json({ workflow_id: "wf-9", status_steps: { done: "s9" } }));
    const res = await createPlanFilesBoard(WS);
    const [url, init] = fetchSpy.mock.calls[0]! as [string, RequestInit];
    expect(url).toContain(`/api/v1/plan-files/board?${WS_PARAM}`);
    expect(init.method).toBe("POST");
    expect(res.workflow_id).toBe("wf-9");
  });

  it("syncPlanFilesNow POSTs to /sync and rejects with a 409 ApiError while a pass runs", async () => {
    fetchSpy.mockResolvedValueOnce(
      json({ outcome: "ok", at: "2026-10-04T10:00:00Z", counts: {}, file_errors: [] }),
    );
    const res = await syncPlanFilesNow(WS);
    const [url, init] = fetchSpy.mock.calls[0]! as [string, RequestInit];
    expect(url).toContain(`/api/v1/plan-files/sync?${WS_PARAM}`);
    expect(init.method).toBe("POST");
    expect(res.outcome).toBe("ok");

    fetchSpy.mockResolvedValueOnce(json({ error: "a plan file sync is already running" }, 409));
    await expect(syncPlanFilesNow(WS)).rejects.toMatchObject({ status: 409 });
  });
});

describe("plan-files-api unadapted", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    global.fetch = fetchSpy as unknown as typeof fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("getUnadaptedPlanFiles reads the rows, scoped to a repository when one is given", async () => {
    const rows = [
      { repository_id: "r1", repository_name: "beaver", count: 16, directories: ["docs/plans"] },
    ];
    fetchSpy.mockResolvedValueOnce(json({ repositories: rows }));
    await expect(getUnadaptedPlanFiles(WS)).resolves.toEqual(rows);
    expect(fetchSpy.mock.calls[0]![0] as string).toContain(
      `/api/v1/plan-files/unadapted?${WS_PARAM}`,
    );
    expect(fetchSpy.mock.calls[0]![0] as string).not.toContain("repository_id");

    fetchSpy.mockResolvedValueOnce(json({ repositories: [] }));
    await getUnadaptedPlanFiles({ ...WS, repositoryId: "r 1" });
    expect(fetchSpy.mock.calls[1]![0] as string).toContain("&repository_id=r%201");
  });

  it("getUnadaptedPlanFiles resolves an empty list for a null repositories field", async () => {
    fetchSpy.mockResolvedValueOnce(json({ repositories: null }));
    await expect(getUnadaptedPlanFiles(WS)).resolves.toEqual([]);
  });
});
