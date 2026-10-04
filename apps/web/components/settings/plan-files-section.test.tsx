import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";
import { ApiError } from "@/lib/api/client";

const getPlanFilesConfig = vi.fn();
const putPlanFilesConfig = vi.fn();
const createPlanFilesBoard = vi.fn();
const getUnadaptedPlanFiles = vi.fn();
const syncPlanFilesNow = vi.fn();
const listWorkflows = vi.fn();
const listWorkflowSteps = vi.fn();

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getPlanFilesConfig: (...a: unknown[]) => getPlanFilesConfig(...a),
  putPlanFilesConfig: (...a: unknown[]) => putPlanFilesConfig(...a),
  createPlanFilesBoard: (...a: unknown[]) => createPlanFilesBoard(...a),
  syncPlanFilesNow: (...a: unknown[]) => syncPlanFilesNow(...a),
  getUnadaptedPlanFiles: (...a: unknown[]) => getUnadaptedPlanFiles(...a),
}));
vi.mock("@/lib/api/domains/kanban-api", () => ({
  listWorkflows: (...a: unknown[]) => listWorkflows(...a),
}));
vi.mock("@/lib/api/domains/workflow-api", () => ({
  listWorkflowSteps: (...a: unknown[]) => listWorkflowSteps(...a),
}));

vi.mock("@/components/toast-provider", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ push: vi.fn() }) }));

import { PlanFilesSection } from "./plan-files-section";

const ENABLED = "plan-files-enabled";
const BOARD = "plan-files-board";
const SAVE = "plan-files-save";
const SYNC_NOW = "plan-files-sync-now";
const STATUSES = ["queued", "in_progress", "waiting_owner", "waiting_external", "deferred", "done"];
const FULL_MAP = Object.fromEntries(STATUSES.map((s, i) => [s, `step-${i}`]));

function config(overrides: Record<string, unknown> = {}) {
  return {
    workspace_id: "ws-1",
    enabled: true,
    workflow_id: "wf-1",
    status_steps: FULL_MAP,
    directories: ["docs/plans", "docs/superpowers/plans"],
    last_pass_at: "2026-10-04T13:42:00Z",
    last_pass_ok: false,
    last_counts: {
      created: 2,
      updated: 1,
      moved: 0,
      archived: 0,
      unarchived: 0,
      failed: 1,
      unadapted: 0,
    },
    last_file_errors: [
      {
        repository_id: "r1",
        repository_name: "city_companion",
        rel_path: "docs/superpowers/plans/x.md",
        reason: "parse",
      },
    ],
    created_at: "2026-10-01T00:00:00Z",
    updated_at: "2026-10-01T00:00:00Z",
    ...overrides,
  };
}

const STEPS = STATUSES.map((_, i) => ({ id: `step-${i}`, name: `Step ${i}`, position: i }));

function renderSection(planFiles = true) {
  return render(
    <StateProvider initialState={{ features: { ...defaultState.features, planFiles } }}>
      <PlanFilesSection workspaceId="ws-1" />
    </StateProvider>,
  );
}

function select(testId: string) {
  return screen.getByTestId(testId) as HTMLSelectElement;
}

beforeEach(() => {
  for (const fn of [
    getPlanFilesConfig,
    putPlanFilesConfig,
    createPlanFilesBoard,
    syncPlanFilesNow,
    getUnadaptedPlanFiles,
    listWorkflows,
    listWorkflowSteps,
  ]) {
    fn.mockReset();
  }
  getUnadaptedPlanFiles.mockResolvedValue([]);
  listWorkflows.mockResolvedValue({
    workflows: [
      { id: "wf-1", name: "Plans", style: "kanban" },
      { id: "wf-2", name: "Other", style: "kanban" },
      { id: "wf-office", name: "Office", style: "office" },
    ],
    total: 3,
  });
  listWorkflowSteps.mockResolvedValue({ steps: STEPS, total: STEPS.length });
});

afterEach(cleanup);

describe("PlanFilesSection", () => {
  it("renders nothing and fetches nothing while the planFiles flag is off (AC-005.6)", () => {
    const { container } = renderSection(false);
    expect(container.firstChild).toBeNull();
    expect(getPlanFilesConfig).not.toHaveBeenCalled();
  });

  it("shows only the header and switch when sync is disabled (AC-005.1)", async () => {
    getPlanFilesConfig.mockResolvedValue(config({ enabled: false }));
    renderSection();
    await screen.findByTestId("plan-files-section");
    await waitFor(() =>
      expect((screen.getByTestId(ENABLED) as HTMLButtonElement).disabled).toBe(false),
    );
    expect(screen.queryByTestId(BOARD)).toBeNull();
    expect(screen.queryByTestId(SYNC_NOW)).toBeNull();
  });

  it("keeps Save disabled until every status except hidden has a step (AC-005.2)", async () => {
    getPlanFilesConfig.mockResolvedValue(config());
    renderSection();
    const board = await waitFor(() => {
      const el = select(BOARD);
      expect(el.value).toBe("wf-1");
      return el;
    });
    expect((screen.getByTestId(SAVE) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.queryByTestId("plan-files-step-hidden")).toBeNull();

    fireEvent.change(board, { target: { value: "wf-2" } });
    await waitFor(() => expect(select("plan-files-step-queued").options.length).toBe(7));
    for (const status of STATUSES.slice(0, -1)) {
      fireEvent.change(select(`plan-files-step-${status}`), {
        target: { value: `step-${STATUSES.indexOf(status)}` },
      });
    }
    expect((screen.getByTestId(SAVE) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByTestId("plan-files-incomplete")).toBeTruthy();

    fireEvent.change(select("plan-files-step-done"), { target: { value: "step-5" } });
    expect((screen.getByTestId(SAVE) as HTMLButtonElement).disabled).toBe(false);

    putPlanFilesConfig.mockResolvedValue(config({ workflow_id: "wf-2" }));
    fireEvent.click(screen.getByTestId(SAVE));
    await waitFor(() => expect(putPlanFilesConfig).toHaveBeenCalledTimes(1));
    const [body, opts] = putPlanFilesConfig.mock.calls[0]!;
    expect(body).toEqual({
      enabled: true,
      workflow_id: "wf-2",
      status_steps: FULL_MAP,
      directories: ["docs/plans", "docs/superpowers/plans"],
    });
    expect(opts).toEqual({ workspaceId: "ws-1" });
  });

  it("Create Plans board fills the mapping and selects the new board (AC-005.2)", async () => {
    getPlanFilesConfig.mockResolvedValue(null);
    createPlanFilesBoard.mockResolvedValue({ workflow_id: "wf-new", status_steps: FULL_MAP });
    listWorkflows.mockResolvedValueOnce({ workflows: [], total: 0 }).mockResolvedValue({
      workflows: [{ id: "wf-new", name: "Plans", style: "kanban" }],
      total: 1,
    });
    renderSection();
    await waitFor(() =>
      expect((screen.getByTestId(ENABLED) as HTMLButtonElement).disabled).toBe(false),
    );
    fireEvent.click(screen.getByTestId(ENABLED));
    fireEvent.click(await screen.findByTestId("plan-files-create-board"));
    await waitFor(() => expect(select(BOARD).value).toBe("wf-new"));
    await waitFor(() => expect(select("plan-files-step-done").value).toBe("step-5"));
    expect(select("plan-files-step-queued").value).toBe("step-0");
    expect((screen.getByTestId(SAVE) as HTMLButtonElement).disabled).toBe(false);
  });

  it("lets the owner add and remove directories, defaults first (AC-005.1)", async () => {
    getPlanFilesConfig.mockResolvedValue(null);
    renderSection();
    await waitFor(() =>
      expect((screen.getByTestId(ENABLED) as HTMLButtonElement).disabled).toBe(false),
    );
    fireEvent.click(screen.getByTestId(ENABLED));
    const list = await screen.findByTestId("plan-files-directories");
    expect(list.textContent).toContain("docs/plans");
    expect(list.textContent).toContain("docs/superpowers/plans");
    fireEvent.change(screen.getByPlaceholderText("Directory, for example docs/plans"), {
      target: { value: "notes/plans" },
    });
    fireEvent.click(screen.getByTestId("plan-files-add-directory"));
    expect(list.textContent).toContain("notes/plans");
    fireEvent.click(screen.getByRole("button", { name: "Remove docs/plans" }));
    expect(list.textContent).not.toContain("docs/plans notes");
    expect(screen.queryByRole("button", { name: "Remove docs/plans" })).toBeNull();
  });
});

describe("PlanFilesSection status and sync", () => {
  it("shows last pass time, counts and per-file errors (AC-005.3)", async () => {
    getPlanFilesConfig.mockResolvedValue(config());
    renderSection();
    const status = await screen.findByTestId("plan-files-status");
    expect(screen.getByTestId("plan-files-counts").textContent).toBe(
      "2 created · 1 updated · 0 moved · 0 archived · 1 failed",
    );
    const errors = screen.getByTestId("plan-files-file-errors");
    expect(errors.textContent).toContain("city_companion: docs/superpowers/plans/x.md");
    expect(errors.textContent).toContain("Could not read the plan header");
    expect(status.getAttribute("data-state")).toBe("failed");
  });

  it("Sync now calls the endpoint and refreshes status (AC-005.3)", async () => {
    getPlanFilesConfig.mockResolvedValue(config());
    syncPlanFilesNow.mockResolvedValue({ outcome: "ok", at: "x", counts: {}, file_errors: [] });
    renderSection();
    const button = await screen.findByTestId(SYNC_NOW);
    await waitFor(() => expect((button as HTMLButtonElement).disabled).toBe(false));
    getPlanFilesConfig.mockResolvedValue(
      config({
        last_pass_ok: true,
        last_file_errors: [],
        last_counts: {
          created: 5,
          updated: 0,
          moved: 0,
          archived: 0,
          unarchived: 0,
          failed: 0,
          unadapted: 0,
        },
      }),
    );
    fireEvent.click(button);
    await waitFor(() => expect(syncPlanFilesNow).toHaveBeenCalledWith({ workspaceId: "ws-1" }));
    await waitFor(() =>
      expect(screen.getByTestId("plan-files-counts").textContent).toContain("5 created"),
    );
  });

  it("shows a 409 as a pass-already-running message, not an error (AC-005.3)", async () => {
    getPlanFilesConfig.mockResolvedValue(config());
    syncPlanFilesNow.mockRejectedValue(
      new ApiError("a plan file sync is already running", 409, null),
    );
    renderSection();
    const button = await screen.findByTestId(SYNC_NOW);
    await waitFor(() => expect((button as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(button);
    const notice = await screen.findByTestId("plan-files-sync-running");
    expect(notice.textContent).toContain("already running");
    expect(screen.queryByTestId("plan-files-error")).toBeNull();
  });
});

describe("PlanFilesSection errors and phone layout", () => {
  it("shows API errors inline when saving fails (AC-005.2)", async () => {
    getPlanFilesConfig.mockResolvedValue(config());
    putPlanFilesConfig.mockRejectedValue(
      new ApiError('invalid plan files config: status "done" needs a step', 400, null),
    );
    renderSection();
    await waitFor(() => expect(select(BOARD).value).toBe("wf-1"));
    fireEvent.click(screen.getByRole("button", { name: "Remove docs/superpowers/plans" }));
    fireEvent.click(screen.getByTestId(SAVE));
    const error = await screen.findByTestId("plan-files-error");
    expect(error.textContent).toContain("needs a step");
  });

  it("stacks actions full width on phones with every desktop action present (AC-005.5)", async () => {
    getPlanFilesConfig.mockResolvedValue(config());
    renderSection();
    const sync = await screen.findByTestId(SYNC_NOW);
    for (const id of ["plan-files-create-board", "plan-files-add-directory"]) {
      expect(screen.getByTestId(id).className).toContain("w-full");
    }
    for (const el of [sync, screen.getByTestId(SAVE)]) {
      expect(el.className).toContain("w-full");
      expect(el.parentElement?.className).toContain("flex-col");
    }
    expect(screen.getByTestId(BOARD).className).toContain("w-full");
  });
});
