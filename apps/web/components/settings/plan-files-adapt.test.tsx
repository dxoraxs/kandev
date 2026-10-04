import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";

const getPlanFilesConfig = vi.fn();
const getUnadaptedPlanFiles = vi.fn();
const startRepositoryMaintenanceTask = vi.fn();
const listWorkflows = vi.fn();
const listWorkflowSteps = vi.fn();
const toast = vi.fn();
const push = vi.fn();

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getPlanFilesConfig: (...a: unknown[]) => getPlanFilesConfig(...a),
  getUnadaptedPlanFiles: (...a: unknown[]) => getUnadaptedPlanFiles(...a),
}));
vi.mock("@/lib/api/domains/repository-maintenance-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/repository-maintenance-api")>()),
  startRepositoryMaintenanceTask: (...a: unknown[]) => startRepositoryMaintenanceTask(...a),
}));
vi.mock("@/lib/api/domains/kanban-api", () => ({
  listWorkflows: (...a: unknown[]) => listWorkflows(...a),
}));
vi.mock("@/lib/api/domains/workflow-api", () => ({
  listWorkflowSteps: (...a: unknown[]) => listWorkflowSteps(...a),
}));
vi.mock("@/components/toast-provider", () => ({ useToast: () => ({ toast }) }));
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ push }) }));

import { MaintenanceTaskError } from "@/lib/api/domains/repository-maintenance-api";
import { PlanFilesSection } from "./plan-files-section";

const ROW = {
  repository_id: "r1",
  repository_name: "beaver-blocks",
  count: 16,
  directories: ["docs/plans"],
};
const ADAPT = "plan-files-adapt";

function config(overrides: Record<string, unknown> = {}) {
  return {
    workspace_id: "ws-1",
    enabled: true,
    workflow_id: "wf-1",
    status_steps: {},
    directories: ["docs/plans"],
    last_pass_at: "2026-10-04T13:42:00Z",
    last_pass_ok: true,
    last_counts: {
      created: 21,
      updated: 2,
      moved: 0,
      archived: 0,
      unarchived: 0,
      failed: 0,
      unadapted: 16,
    },
    last_file_errors: [],
    created_at: "2026-10-01T00:00:00Z",
    updated_at: "2026-10-01T00:00:00Z",
    ...overrides,
  };
}

function renderSection(planFiles = true) {
  return render(
    <StateProvider initialState={{ features: { ...defaultState.features, planFiles } }}>
      <PlanFilesSection workspaceId="ws-1" />
    </StateProvider>,
  );
}

beforeEach(() => {
  for (const fn of [
    getPlanFilesConfig,
    getUnadaptedPlanFiles,
    startRepositoryMaintenanceTask,
    listWorkflows,
    listWorkflowSteps,
    toast,
    push,
  ]) {
    fn.mockReset();
  }
  listWorkflows.mockResolvedValue({ workflows: [], total: 0 });
  listWorkflowSteps.mockResolvedValue({ steps: [], total: 0 });
  getPlanFilesConfig.mockResolvedValue(config());
  getUnadaptedPlanFiles.mockResolvedValue([ROW]);
});

afterEach(cleanup);

describe("PlanFilesSection unadapted rows", () => {
  it("lists a repository with its count and Adapt with agent", async () => {
    renderSection();
    const row = await screen.findByTestId("plan-files-unadapted-row");
    expect(row.textContent).toContain("beaver-blocks");
    expect(screen.getByTestId("plan-files-unadapted-count").textContent).toBe(
      "16 files without board status",
    );
    expect(screen.getByTestId(ADAPT).textContent).toBe("Adapt with agent");
    expect(screen.queryByText("A Plans board will be created and sync turned on.")).toBeNull();
  });

  it("renders the rows with sync disabled and says the board will be created without a config", async () => {
    getPlanFilesConfig.mockResolvedValue(null);
    renderSection();
    await screen.findByTestId("plan-files-unadapted-row");
    expect(screen.queryByTestId("plan-files-board")).toBeNull();
    expect(screen.getByText("A Plans board will be created and sync turned on.")).toBeTruthy();
  });

  it("shows the not adapted count in the status line and hides it at zero", async () => {
    renderSection();
    await waitFor(() =>
      expect(screen.getByTestId("plan-files-counts").textContent).toBe(
        "21 created · 2 updated · 0 moved · 0 archived · 0 failed · 16 not adapted",
      ),
    );
    cleanup();
    getPlanFilesConfig.mockResolvedValue(
      config({ last_counts: { ...config().last_counts, unadapted: 0 } }),
    );
    renderSection();
    await waitFor(() =>
      expect(screen.getByTestId("plan-files-counts").textContent).not.toContain("not adapted"),
    );
  });

  it("renders no row and reads nothing while the planFiles flag is off", () => {
    const { container } = renderSection(false);
    expect(container.firstChild).toBeNull();
    expect(getUnadaptedPlanFiles).not.toHaveBeenCalled();
  });

  it("renders no row when the read fails or nothing is unadapted", async () => {
    getUnadaptedPlanFiles.mockRejectedValue(new Error("boom"));
    renderSection();
    await screen.findByTestId("plan-files-section");
    await waitFor(() => expect(getUnadaptedPlanFiles).toHaveBeenCalled());
    expect(screen.queryByTestId("plan-files-unadapted")).toBeNull();
  });
});

describe("PlanFilesSection Adapt with agent", () => {
  it("starts plan_adaptation and opens the created task", async () => {
    startRepositoryMaintenanceTask.mockResolvedValue({
      task_id: "t1",
      session_id: "s1",
      existing: false,
    });
    renderSection();
    fireEvent.click(await screen.findByTestId(ADAPT));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/t/t1"));
    expect(startRepositoryMaintenanceTask).toHaveBeenCalledWith("r1", "plan_adaptation");
    expect(toast).not.toHaveBeenCalled();
  });

  it("opens an existing task with an empty session id and an info toast", async () => {
    startRepositoryMaintenanceTask.mockResolvedValue({
      task_id: "t7",
      session_id: "",
      existing: true,
    });
    renderSection();
    fireEvent.click(await screen.findByTestId(ADAPT));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/t/t7"));
    expect(toast).toHaveBeenCalledWith(
      expect.objectContaining({
        variant: "default",
        description: expect.stringContaining("Opening"),
      }),
    );
  });

  it("disables the action while a start is pending so a second click does nothing", async () => {
    let resolve: (value: unknown) => void = () => {};
    startRepositoryMaintenanceTask.mockReturnValue(new Promise((r) => (resolve = r)));
    renderSection();
    const button = (await screen.findByTestId(ADAPT)) as HTMLButtonElement;
    fireEvent.click(button);
    await waitFor(() => expect(button.disabled).toBe(true));
    fireEvent.click(button);
    expect(startRepositoryMaintenanceTask).toHaveBeenCalledTimes(1);
    resolve({ task_id: "t1", session_id: "s1", existing: false });
    await waitFor(() => expect(push).toHaveBeenCalledTimes(1));
  });

  it.each([
    ["no_agent_profile", "Set a default agent profile"],
    ["no_workflow", "no workflow"],
    ["repository_not_local", "on this machine"],
    ["kind_unavailable", "not available"],
    ["repository_not_found", "no longer exists"],
    ["failed", "Could not start"],
  ] as const)("shows localized copy for %s and does not navigate", async (reason, copy) => {
    startRepositoryMaintenanceTask.mockRejectedValue(
      new MaintenanceTaskError(reason, "backend text must never be shown"),
    );
    renderSection();
    fireEvent.click(await screen.findByTestId(ADAPT));
    await waitFor(() => expect(toast).toHaveBeenCalled());
    const arg = toast.mock.calls[0]![0] as { description: string; variant: string };
    expect(arg.variant).toBe("error");
    expect(arg.description).toContain(copy);
    expect(arg.description).not.toContain("backend text");
    expect(push).not.toHaveBeenCalled();
  });

  it("opens a task created before a launch failure and still reports the error", async () => {
    startRepositoryMaintenanceTask.mockRejectedValue(
      new MaintenanceTaskError("failed", "launch failed", "t9"),
    );
    renderSection();
    fireEvent.click(await screen.findByTestId(ADAPT));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/t/t9"));
    expect(toast).toHaveBeenCalledWith(expect.objectContaining({ variant: "error" }));
  });

  it("uses full-width 44px phone actions that shrink to auto width on desktop", async () => {
    renderSection();
    const className = (await screen.findByTestId(ADAPT)).className;
    expect(className).toContain("w-full");
    expect(className).toContain("md:w-auto");
    expect(className).toContain("max-md:h-11");
    expect(className).toContain("h-7");
    expect(screen.getByTestId("plan-files-unadapted-row").className).toContain("flex-col");
  });
});
