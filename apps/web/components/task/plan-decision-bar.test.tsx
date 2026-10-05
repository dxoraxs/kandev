import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";
import { ApiError } from "@/lib/api/client";
import { invalidatePlanBoardConfig } from "@/hooks/domains/plans/use-plan-board";

const getPlanFilesConfig = vi.fn();
const decidePlan = vi.fn();

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getPlanFilesConfig: (...a: unknown[]) => getPlanFilesConfig(...a),
  decidePlan: (...a: unknown[]) => decidePlan(...a),
}));

import { PlanDecisionBar, type PlanDecisionBarProps } from "./plan-decision-bar";

const WS = "ws-1";
const BAR = "plan-decision-bar";
const RETURN = "plan-decision-return";
const ACCEPT = "plan-decision-accept";
const CONFIRM = "plan-decision-confirm";
const COMMENT = "plan-decision-comment";
const ERROR = "plan-decision-error";
const TASK = "task-1";
const DRAWER_RETURN = "plan-decision-drawer-return";
const BOARD = "wf-plans";
const WAITING = "step-waiting";
const QUEUED = "step-queued";

function config(overrides: Record<string, unknown> = {}) {
  return {
    workspace_id: WS,
    enabled: true,
    workflow_id: BOARD,
    status_steps: { queued: QUEUED, waiting_owner: WAITING, done: "step-done" },
    ...overrides,
  };
}

const PROPS: PlanDecisionBarProps = {
  taskId: TASK,
  workspaceId: WS,
  workflowId: BOARD,
  workflowStepId: WAITING,
  presentation: "desktop",
};

function mount(props: Partial<PlanDecisionBarProps> = {}, planFiles = true) {
  return render(
    <StateProvider initialState={{ features: { ...defaultState.features, planFiles } }}>
      <PlanDecisionBar {...PROPS} {...props} />
    </StateProvider>,
  );
}

async function settle() {
  await act(async () => {
    await Promise.resolve();
  });
}

function setup() {
  beforeEach(() => {
    invalidatePlanBoardConfig(WS);
    getPlanFilesConfig.mockReset().mockResolvedValue(config());
    decidePlan.mockReset().mockResolvedValue({ board: "queued" });
  });
  afterEach(() => cleanup());
}

async function openConfirm(testId: string) {
  fireEvent.click(await screen.findByTestId(testId));
  fireEvent.click(screen.getByTestId(CONFIRM));
}

async function expectAbsent(reads = 1) {
  await waitFor(() => expect(getPlanFilesConfig).toHaveBeenCalledTimes(reads));
  await settle();
  expect(screen.queryByTestId(BAR)).toBeNull();
}

describe("PlanDecisionBar visibility", () => {
  setup();

  it("shows for a plan-board task in the waiting-owner step", async () => {
    mount();
    expect(await screen.findByTestId(BAR)).toBeTruthy();
    expect(screen.getByText("This plan waits for your decision.")).toBeTruthy();
    expect(getPlanFilesConfig).toHaveBeenCalledTimes(1);
  });

  it("is absent in another step of the plan board", async () => {
    mount({ workflowStepId: QUEUED });
    await expectAbsent();
  });

  it("is absent for a task of another workflow", async () => {
    mount({ workflowId: "wf-other" });
    await expectAbsent();
  });

  it("is absent when the workspace has no plan-files config", async () => {
    getPlanFilesConfig.mockResolvedValue(null);
    mount();
    await expectAbsent();
  });

  it("is absent when sync is turned off", async () => {
    getPlanFilesConfig.mockResolvedValue(config({ enabled: false }));
    mount();
    await expectAbsent();
  });

  it("is absent and reads nothing while the planFiles feature is off", async () => {
    mount({}, false);
    await settle();
    expect(getPlanFilesConfig).not.toHaveBeenCalled();
    expect(screen.queryByTestId(BAR)).toBeNull();
  });

  it("reads the config once per workspace across mounts", async () => {
    mount();
    await screen.findByTestId(BAR);
    cleanup();
    mount();
    await screen.findByTestId(BAR);
    expect(getPlanFilesConfig).toHaveBeenCalledTimes(1);
  });
});

describe("PlanDecisionBar desktop decisions", () => {
  setup();

  it("keeps Return disabled until the comment has text", async () => {
    mount();
    fireEvent.click(await screen.findByTestId(RETURN));
    const confirm = screen.getByTestId(CONFIRM) as HTMLButtonElement;
    expect(confirm.disabled).toBe(true);
    fireEvent.change(screen.getByTestId(COMMENT), { target: { value: "   " } });
    expect(confirm.disabled).toBe(true);
    fireEvent.change(screen.getByTestId(COMMENT), { target: { value: "redo" } });
    expect(confirm.disabled).toBe(false);
  });

  it("submits Return with the comment", async () => {
    mount();
    fireEvent.click(await screen.findByTestId(RETURN));
    fireEvent.change(screen.getByTestId(COMMENT), { target: { value: "fix the totals" } });
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() =>
      expect(decidePlan).toHaveBeenCalledWith(TASK, {
        action: "return",
        comment: "fix the totals",
      }),
    );
  });

  it("submits Accept and close without a comment", async () => {
    mount();
    await openConfirm(ACCEPT);
    await waitFor(() =>
      expect(decidePlan).toHaveBeenCalledWith(TASK, { action: "accept", result: "done" }),
    );
  });

  it("submits Accept, back to queue from the split menu", async () => {
    mount();
    const menu = await screen.findByTestId("plan-decision-accept-menu");
    fireEvent.pointerDown(menu, { button: 0, ctrlKey: false });
    fireEvent.click(menu);
    fireEvent.click(await screen.findByTestId("plan-decision-accept-queue"));
    fireEvent.change(await screen.findByTestId(COMMENT), { target: { value: "go on" } });
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() =>
      expect(decidePlan).toHaveBeenCalledWith(TASK, {
        action: "accept",
        result: "queued",
        comment: "go on",
      }),
    );
  });

  it("hides the bar once the decision is recorded", async () => {
    mount();
    await openConfirm(ACCEPT);
    await waitFor(() => expect(screen.queryByTestId(BAR)).toBeNull());
  });
});

describe("PlanDecisionBar errors and phone", () => {
  setup();

  it("shows the file-changed copy and keeps the bar when the file moved on", async () => {
    decidePlan.mockRejectedValue(new ApiError("changed", 409, { code: "file_changed" }));
    mount();
    await openConfirm(ACCEPT);
    expect((await screen.findByTestId(ERROR)).textContent).toBe(
      "The plan file changed since this page last read it. Sync the board and try again.",
    );
    expect(screen.getByTestId(BAR)).toBeTruthy();
  });

  it("hides the bar when the server says the task is not a plan task", async () => {
    decidePlan.mockRejectedValue(new ApiError("none", 404, { code: "not_plan_task" }));
    mount();
    await openConfirm(ACCEPT);
    await waitFor(() => expect(screen.queryByTestId(BAR)).toBeNull());
  });

  it("shows the generic copy for any other failure", async () => {
    decidePlan.mockRejectedValue(new ApiError("boom", 500, { error: "boom" }));
    mount();
    await openConfirm(ACCEPT);
    expect((await screen.findByTestId(ERROR)).textContent).toBe(
      "The decision could not be saved. Try again.",
    );
  });

  it("renders a phone trigger whose drawer buttons are at least 44px high", async () => {
    mount({ presentation: "phone" });
    const trigger = await screen.findByTestId("plan-decision-trigger");
    expect(trigger.className).toContain("min-h-11");
    expect(trigger.className).toContain("w-full");
    expect(screen.queryByTestId(RETURN)).toBeNull();
    fireEvent.click(trigger);
    await screen.findByTestId("plan-decision-drawer");
    for (const id of [
      "plan-decision-drawer-accept",
      "plan-decision-drawer-accept-queue",
      DRAWER_RETURN,
    ]) {
      const button = screen.getByTestId(id);
      expect(button.className).toContain("min-h-11");
      expect(button.className).toContain("w-full");
    }
    expect((screen.getByTestId(DRAWER_RETURN) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByTestId(COMMENT), { target: { value: "redo" } });
    fireEvent.click(screen.getByTestId(DRAWER_RETURN));
    await waitFor(() =>
      expect(decidePlan).toHaveBeenCalledWith(TASK, { action: "return", comment: "redo" }),
    );
  });
});
