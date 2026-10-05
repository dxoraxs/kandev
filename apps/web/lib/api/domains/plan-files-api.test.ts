import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../client";
import {
  commitPlanFiles,
  createPlanFilesBoard,
  createPlan,
  planCreateErrorCode,
  decidePlan,
  getPlanGitStatus,
  planCommitError,
  planDecisionErrorCode,
  getPlanFilesConfig,
  getUnadaptedPlanFiles,
  putPlanFilesConfig,
  syncPlanFilesNow,
} from "./plan-files-api";

const originalFetch = global.fetch;
const WS = { workspaceId: "ws 1" };
const PLANS_DIR = "docs/plans";
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
      directories: [PLANS_DIR],
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
      directories: [PLANS_DIR],
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
      { repository_id: "r1", repository_name: "beaver", count: 16, directories: [PLANS_DIR] },
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

describe("plan-files-api decision", () => {
  beforeEach(() => {
    global.fetch = vi.fn() as unknown as typeof fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("decidePlan POSTs the decision to the task route without a workspace parameter", async () => {
    const fetchSpy = vi.fn().mockResolvedValueOnce(json({ board: "queued" }));
    global.fetch = fetchSpy as unknown as typeof fetch;
    const res = await decidePlan("task 1", { action: "return", comment: "fix it" });
    const [url, init] = fetchSpy.mock.calls[0]! as [string, RequestInit];
    expect(url).toContain("/api/v1/plan-files/tasks/task%201/decision");
    expect(url).not.toContain("workspace_id");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toEqual({ action: "return", comment: "fix it" });
    expect(res.board).toBe("queued");
  });

  it("planDecisionErrorCode reads the typed code of a rejected decision", async () => {
    const fetchSpy = vi
      .fn()
      .mockResolvedValueOnce(json({ error: "changed", code: "file_changed" }, 409))
      .mockResolvedValueOnce(json({ error: "boom" }, 500));
    global.fetch = fetchSpy as unknown as typeof fetch;
    const changed = await decidePlan("t", { action: "accept" }).catch((e) => e);
    expect(planDecisionErrorCode(changed)).toBe("file_changed");
    const other = await decidePlan("t", { action: "accept" }).catch((e) => e);
    expect(planDecisionErrorCode(other)).toBeNull();
    expect(planDecisionErrorCode(new Error("x"))).toBeNull();
  });
});

describe("plan-files-api git", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    global.fetch = fetchSpy as unknown as typeof fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("getPlanGitStatus reads the repositories with an encoded workspace_id", async () => {
    const rows = [{ repository_id: "r1", repository_name: "dmhive", files: ["docs/plans/a.md"] }];
    fetchSpy.mockResolvedValueOnce(json({ repositories: rows }));
    await expect(getPlanGitStatus(WS)).resolves.toEqual(rows);
    expect(fetchSpy.mock.calls[0]![0] as string).toContain(
      `/api/v1/plan-files/git-status?${WS_PARAM}`,
    );
  });

  it("getPlanGitStatus resolves an empty list for a null repositories field", async () => {
    fetchSpy.mockResolvedValueOnce(json({ repositories: null }));
    await expect(getPlanGitStatus(WS)).resolves.toEqual([]);
  });

  it("commitPlanFiles POSTs the repository and message to /commit", async () => {
    fetchSpy.mockResolvedValueOnce(json({ commit: "abc123", files: ["docs/plans/a.md"] }));
    const res = await commitPlanFiles({ repository_id: "r1", message: "plans: tidy" }, WS);
    const [url, init] = fetchSpy.mock.calls[0]! as [string, RequestInit];
    expect(url).toContain(`/api/v1/plan-files/commit?${WS_PARAM}`);
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toEqual({
      repository_id: "r1",
      message: "plans: tidy",
    });
    expect(res.commit).toBe("abc123");
  });

  it("planCommitError reads the typed code and output of a rejected commit", async () => {
    fetchSpy
      .mockResolvedValueOnce(json({ error: "busy", code: "repository_busy" }, 409))
      .mockResolvedValueOnce(
        json({ error: "failed", code: "commit_failed", output: "hook said no" }, 422),
      )
      .mockResolvedValueOnce(json({ error: "boom" }, 500));
    const busy = await commitPlanFiles({ repository_id: "r" }, WS).catch((e) => e);
    expect(planCommitError(busy)).toEqual({ code: "repository_busy", output: "" });
    const failed = await commitPlanFiles({ repository_id: "r" }, WS).catch((e) => e);
    expect(planCommitError(failed)).toEqual({ code: "commit_failed", output: "hook said no" });
    const other = await commitPlanFiles({ repository_id: "r" }, WS).catch((e) => e);
    expect(planCommitError(other)).toBeNull();
    expect(planCommitError(new Error("x"))).toBeNull();
  });
});

describe("plan-files-api create", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    global.fetch = fetchSpy as unknown as typeof fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("createPlan POSTs the plan to /plans with an encoded workspace_id", async () => {
    const created = { task_id: "t1", repository_id: "r1", rel_path: "docs/plans/fix.md" };
    fetchSpy.mockResolvedValueOnce(json(created, 201));
    const body = {
      repository_id: "r1",
      directory: PLANS_DIR,
      title: "Fix",
      priority: "high" as const,
    };
    await expect(createPlan(body, WS)).resolves.toEqual(created);
    const [url, init] = fetchSpy.mock.calls[0]! as [string, RequestInit];
    expect(url).toContain(`/api/v1/plan-files/plans?${WS_PARAM}`);
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toEqual(body);
  });

  it("planCreateErrorCode reads the typed code of a rejected create", async () => {
    fetchSpy
      .mockResolvedValueOnce(json({ error: "exists", code: "file_exists" }, 409))
      .mockResolvedValueOnce(json({ error: "bad", code: "invalid_plan" }, 400))
      .mockResolvedValueOnce(json({ error: "gone", code: "repository_not_found" }, 404))
      .mockResolvedValueOnce(json({ error: "boom" }, 500));
    const create = () =>
      createPlan({ repository_id: "r", directory: "d", title: "t" }, WS).catch((e) => e);
    expect(planCreateErrorCode(await create())).toBe("file_exists");
    expect(planCreateErrorCode(await create())).toBe("invalid_plan");
    expect(planCreateErrorCode(await create())).toBe("repository_not_found");
    expect(planCreateErrorCode(await create())).toBeNull();
    expect(planCreateErrorCode(new Error("x"))).toBeNull();
  });
});
