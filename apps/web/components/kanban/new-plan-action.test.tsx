import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";
import { invalidatePlanBoardConfig } from "@/hooks/domains/plans/use-plan-board";
import type { TaskListingPage } from "@/lib/task-listing/view-navigation";

const getPlanFilesConfig = vi.fn();

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getPlanFilesConfig: (...a: unknown[]) => getPlanFilesConfig(...a),
}));

import { NewPlanButton, useNewPlanAction } from "./new-plan-action";

const WS = "ws-1";
const BOARD = "wf-plans";

function Probe({ page, workflowId }: { page: TaskListingPage; workflowId: string | null }) {
  const action = useNewPlanAction(WS, page, workflowId);
  return (
    <div>
      <span data-testid="available">{String(action.available)}</span>
      <span data-testid="open">{String(action.open)}</span>
      <button data-testid="open-it" onClick={() => action.setOpen(true)} />
    </div>
  );
}

async function mount(page: TaskListingPage, workflowId: string | null, planFiles = true) {
  render(
    <StateProvider initialState={{ features: { ...defaultState.features, planFiles } }}>
      <Probe page={page} workflowId={workflowId} />
    </StateProvider>,
  );
  await act(async () => {
    await Promise.resolve();
  });
}

const available = () => screen.getByTestId("available").textContent;

beforeEach(() => {
  invalidatePlanBoardConfig(WS);
  getPlanFilesConfig.mockReset().mockResolvedValue({
    workspace_id: WS,
    enabled: true,
    workflow_id: BOARD,
    status_steps: {},
    directories: [],
    executor_steps: {},
  });
});
afterEach(() => cleanup());

describe("useNewPlanAction", () => {
  it("is available on the kanban page of the plan board", async () => {
    await mount("kanban", BOARD);
    expect(available()).toBe("true");
  });

  it("is not available for another workflow, no workflow, or another page", async () => {
    await mount("kanban", "wf-other");
    expect(available()).toBe("false");
    cleanup();
    await mount("kanban", null);
    expect(available()).toBe("false");
    cleanup();
    await mount("tasks", BOARD);
    expect(available()).toBe("false");
  });

  it("is not available when sync is turned off or the feature is off", async () => {
    getPlanFilesConfig.mockResolvedValue({ workspace_id: WS, enabled: false, workflow_id: BOARD });
    await mount("kanban", BOARD);
    expect(available()).toBe("false");
    cleanup();
    invalidatePlanBoardConfig(WS);
    getPlanFilesConfig.mockClear();
    await mount("kanban", BOARD, false);
    expect(available()).toBe("false");
    expect(getPlanFilesConfig).not.toHaveBeenCalled();
  });

  it("opens only while available", async () => {
    await mount("kanban", BOARD);
    expect(screen.getByTestId("open").textContent).toBe("false");
    fireEvent.click(screen.getByTestId("open-it"));
    expect(screen.getByTestId("open").textContent).toBe("true");
    cleanup();
    await mount("kanban", "wf-other");
    fireEvent.click(screen.getByTestId("open-it"));
    expect(screen.getByTestId("open").textContent).toBe("false");
  });
});

describe("NewPlanButton", () => {
  it("shows the label on desktop and calls onClick", () => {
    const onClick = vi.fn();
    render(<NewPlanButton onClick={onClick} />);
    const button = screen.getByTestId("new-plan-button");
    expect(button.textContent).toContain("New plan");
    fireEvent.click(button);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("keeps the name for assistive technology in the compact form", () => {
    render(<NewPlanButton onClick={vi.fn()} compact />);
    const button = screen.getByTestId("new-plan-button");
    expect(button.getAttribute("data-compact")).toBe("true");
    expect(screen.getByRole("button", { name: "New plan" })).toBe(button);
  });
});
