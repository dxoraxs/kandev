import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";

const startRepositoryMaintenanceTask = vi.fn();
const toast = vi.fn();
const push = vi.fn();

vi.mock("@/lib/api/domains/repository-maintenance-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/repository-maintenance-api")>()),
  startRepositoryMaintenanceTask: (...a: unknown[]) => startRepositoryMaintenanceTask(...a),
}));
let mobile = false;
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile: mobile, isFinePointer: !mobile }),
}));
vi.mock("@/components/toast-provider", () => ({ useToast: () => ({ toast }) }));
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ push }) }));

import { MaintenanceTaskError } from "@/lib/api/domains/repository-maintenance-api";
import { RepositoryCleanupDialog } from "./repository-cleanup-dialog";

const DIALOG = "repository-cleanup-dialog";
const CONFIRM = "repository-cleanup-confirm";
const CANCEL = "repository-cleanup-cancel";

beforeEach(() => {
  mobile = false;
  for (const fn of [startRepositoryMaintenanceTask, toast, push]) fn.mockReset();
});
afterEach(cleanup);

function renderDialog(onOpenChange = vi.fn(), open = true) {
  render(
    <RepositoryCleanupDialog
      repositoryId="r1"
      repositoryName="city_companion"
      open={open}
      onOpenChange={onOpenChange}
    />,
  );
  return onOpenChange;
}

describe("RepositoryCleanupDialog content", () => {
  it("renders nothing while closed", () => {
    renderDialog(vi.fn(), false);
    expect(screen.queryByTestId(DIALOG)).toBeNull();
  });

  it("names the repository, lists the four steps, the checkout and the protected note", () => {
    renderDialog();
    const dialog = screen.getByTestId(DIALOG);
    expect(dialog.textContent).toContain("Clean up city_companion?");
    expect(dialog.textContent).toContain("main checkout");
    expect(dialog.querySelectorAll("ol > li").length).toBe(4);
    expect(dialog.textContent).toContain("Commit uncommitted worktree changes as WIP");
    expect(dialog.textContent).toContain("Merge every branch and worktree into the default branch");
    expect(dialog.textContent).toContain("wait for your confirmation");
    expect(dialog.textContent).toContain("delete merged branches locally and on the remote");
    expect(dialog.textContent).toContain("active Kandev tasks are not touched");
  });

  it("has exactly two actions", () => {
    renderDialog();
    expect(screen.getByTestId(CANCEL).textContent).toBe("Cancel");
    expect(screen.getByTestId(CONFIRM).textContent).toBe("Start cleanup");
    const dialog = screen.getByTestId(DIALOG);
    expect(dialog.querySelectorAll("button:not([data-slot='dialog-close'])").length).toBe(2);
  });

  it("uses a bottom drawer with stacked full-width actions, primary first, on a phone", () => {
    mobile = true;
    renderDialog();
    const sheet = screen.getByTestId(DIALOG);
    expect(sheet.getAttribute("data-slot")).toBe("drawer-content");
    const confirm = screen.getByTestId(CONFIRM);
    const cancel = screen.getByTestId(CANCEL);
    expect(confirm.className).toContain("w-full");
    expect(cancel.className).toContain("w-full");
    expect(confirm.className).toContain("max-md:h-11");
    // eslint-disable-next-line no-bitwise
    expect(confirm.compareDocumentPosition(cancel) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});

describe("RepositoryCleanupDialog actions", () => {
  it("Cancel closes with no request and no navigation", () => {
    const onOpenChange = renderDialog();
    fireEvent.click(screen.getByTestId(CANCEL));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(startRepositoryMaintenanceTask).not.toHaveBeenCalled();
    expect(push).not.toHaveBeenCalled();
  });

  it("Start cleanup starts the cleanup kind, opens the task and closes", async () => {
    startRepositoryMaintenanceTask.mockResolvedValue({
      task_id: "t1",
      session_id: "s1",
      existing: false,
    });
    const onOpenChange = renderDialog();
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/t/t1"));
    expect(startRepositoryMaintenanceTask).toHaveBeenCalledWith("r1", "repository_cleanup");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("opens the existing task with an info toast", async () => {
    startRepositoryMaintenanceTask.mockResolvedValue({
      task_id: "t9",
      session_id: "",
      existing: true,
    });
    renderDialog();
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/t/t9"));
    expect(toast).toHaveBeenCalledWith(
      expect.objectContaining({ description: "A task for this is already running. Opening it." }),
    );
  });

  it("stays open with the localized reason when the launcher rejects", async () => {
    startRepositoryMaintenanceTask.mockRejectedValue(
      new MaintenanceTaskError("no_agent_profile", "x"),
    );
    const onOpenChange = renderDialog();
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() =>
      expect(toast).toHaveBeenCalledWith(
        expect.objectContaining({
          description: "Set a default agent profile for this workspace first.",
          variant: "error",
        }),
      ),
    );
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(push).not.toHaveBeenCalled();
  });

  it("a second click while starting sends one request", async () => {
    let resolve: (v: unknown) => void = () => {};
    startRepositoryMaintenanceTask.mockReturnValue(new Promise((r) => (resolve = r)));
    renderDialog();
    const confirm = screen.getByTestId(CONFIRM);
    fireEvent.click(confirm);
    fireEvent.click(confirm);
    expect(startRepositoryMaintenanceTask).toHaveBeenCalledTimes(1);
    resolve({ task_id: "t1", session_id: "s", existing: false });
    await waitFor(() => expect(push).toHaveBeenCalled());
  });
});
