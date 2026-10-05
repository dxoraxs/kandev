import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  act,
  cleanup,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";

const getPlanFilesConfig = vi.fn();
const getUnadaptedPlanFiles = vi.fn();
const startRepositoryMaintenanceTask = vi.fn();
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
let mobile = false;
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile: mobile }),
}));
vi.mock("@/components/toast-provider", () => ({ useToast: () => ({ toast }) }));
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ push }) }));

import { usePlanAdaptationOffer } from "@/hooks/domains/settings/use-plan-adaptation-offer";
import { AdaptPlansOfferDialog, type PlanAdaptationOffer } from "./adapt-plans-offer-dialog";

const REPO = "beaver-blocks";
const CONFIRM = "adapt-plans-confirm";
const NOT_NOW = "adapt-plans-not-now";
const OFFER_ID = "adapt-plans-offer";
const OFFER: PlanAdaptationOffer = {
  repositoryId: "r1",
  repositoryName: REPO,
  count: 16,
  boardWillBeCreated: false,
};

beforeEach(() => {
  mobile = false;
  for (const fn of [
    getPlanFilesConfig,
    getUnadaptedPlanFiles,
    startRepositoryMaintenanceTask,
    toast,
    push,
  ]) {
    fn.mockReset();
  }
});

afterEach(cleanup);

describe("AdaptPlansOfferDialog", () => {
  it("renders nothing without an offer", () => {
    render(<AdaptPlansOfferDialog offer={null} onDismiss={vi.fn()} />);
    expect(screen.queryByTestId(OFFER_ID)).toBeNull();
  });

  it("names the repository and count and offers exactly two actions", () => {
    render(<AdaptPlansOfferDialog offer={OFFER} onDismiss={vi.fn()} />);
    const dialog = screen.getByTestId(OFFER_ID);
    expect(dialog.textContent).toContain("Plans found in beaver-blocks");
    expect(dialog.textContent).toContain("16 plan files are not in the board format");
    expect(screen.getByTestId(NOT_NOW).textContent).toBe("Not now");
    expect(screen.getByTestId(CONFIRM).textContent).toBe("Adapt with agent");
    expect(dialog.querySelectorAll("button:not([data-slot='dialog-close'])").length).toBe(2);
    expect(screen.queryByTestId("adapt-plans-offer-board")).toBeNull();
  });

  it("uses the singular form and shows the board helper only when sync is not configured", () => {
    render(
      <AdaptPlansOfferDialog
        offer={{ ...OFFER, count: 1, boardWillBeCreated: true }}
        onDismiss={vi.fn()}
      />,
    );
    expect(screen.getByTestId(OFFER_ID).textContent).toContain(
      "1 plan file is not in the board format",
    );
    expect(screen.getByTestId("adapt-plans-offer-board").textContent).toContain(
      "A Plans board will be created",
    );
  });

  it("Not now dismisses with no request and no navigation", () => {
    const onDismiss = vi.fn();
    render(<AdaptPlansOfferDialog offer={OFFER} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId(NOT_NOW));
    expect(onDismiss).toHaveBeenCalledTimes(1);
    expect(startRepositoryMaintenanceTask).not.toHaveBeenCalled();
    expect(push).not.toHaveBeenCalled();
    expect(toast).not.toHaveBeenCalled();
  });
});

describe("AdaptPlansOfferDialog actions", () => {
  it("Adapt with agent starts the task, opens it and closes the offer", async () => {
    startRepositoryMaintenanceTask.mockResolvedValue({
      task_id: "t1",
      session_id: "s1",
      existing: false,
    });
    const onDismiss = vi.fn();
    render(<AdaptPlansOfferDialog offer={OFFER} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/t/t1"));
    expect(startRepositoryMaintenanceTask).toHaveBeenCalledWith("r1", "plan_adaptation");
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("keeps the offer open when the start fails", async () => {
    startRepositoryMaintenanceTask.mockRejectedValue(new Error("network"));
    const onDismiss = vi.fn();
    render(<AdaptPlansOfferDialog offer={OFFER} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() =>
      expect(toast).toHaveBeenCalledWith(expect.objectContaining({ variant: "error" })),
    );
    expect(onDismiss).not.toHaveBeenCalled();
    expect(push).not.toHaveBeenCalled();
  });

  it("keeps the desktop actions compact and in dialog order", () => {
    render(<AdaptPlansOfferDialog offer={OFFER} onDismiss={vi.fn()} />);
    const dialog = screen.getByTestId(OFFER_ID);
    expect(dialog.getAttribute("data-slot")).toBe("dialog-content");
    for (const id of [NOT_NOW, CONFIRM]) {
      const className = screen.getByTestId(id).className;
      expect(className).toContain("w-auto");
      expect(className).toContain("h-7");
    }
    const buttons = [...dialog.querySelectorAll("[data-testid^='adapt-plans-']")].map((el) =>
      el.getAttribute("data-testid"),
    );
    expect(buttons).toEqual([NOT_NOW, CONFIRM]);
  });

  it("is an inset bottom drawer on phones with stacked full-width 44px actions, primary first", () => {
    mobile = true;
    render(
      <AdaptPlansOfferDialog offer={{ ...OFFER, boardWillBeCreated: true }} onDismiss={vi.fn()} />,
    );
    const sheet = screen.getByTestId(OFFER_ID);
    expect(sheet.getAttribute("data-slot")).toBe("drawer-content");
    expect(sheet.textContent).toContain(`Plans found in ${REPO}`);
    expect(screen.getByTestId("adapt-plans-offer-board")).toBeTruthy();
    for (const id of [NOT_NOW, CONFIRM]) {
      const className = screen.getByTestId(id).className;
      expect(className).toContain("w-full");
      expect(className).toContain("max-md:h-11");
    }
    const order = [...sheet.querySelectorAll("[data-testid^='adapt-plans-']")]
      .map((el) => el.getAttribute("data-testid"))
      .filter((id) => id === CONFIRM || id === NOT_NOW);
    expect(order).toEqual([CONFIRM, NOT_NOW]);
  });

  it("Not now on the phone drawer dismisses without a request", () => {
    mobile = true;
    const onDismiss = vi.fn();
    render(<AdaptPlansOfferDialog offer={OFFER} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId(NOT_NOW));
    expect(onDismiss).toHaveBeenCalledTimes(1);
    expect(startRepositoryMaintenanceTask).not.toHaveBeenCalled();
  });
});

function wrapper(planFiles: boolean) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <StateProvider initialState={{ features: { ...defaultState.features, planFiles } }}>
        {children}
      </StateProvider>
    );
  };
}

describe("usePlanAdaptationOffer", () => {
  it("offers the repository when it has unadapted plans and sync is configured", async () => {
    getUnadaptedPlanFiles.mockResolvedValue([
      { repository_id: "r1", repository_name: REPO, count: 16, directories: [] },
    ]);
    getPlanFilesConfig.mockResolvedValue({ enabled: true });
    const { result } = renderHook(() => usePlanAdaptationOffer("ws-1"), { wrapper: wrapper(true) });
    await act(() => result.current.check({ id: "r1", name: REPO }));
    expect(getUnadaptedPlanFiles).toHaveBeenCalledWith({ workspaceId: "ws-1", repositoryId: "r1" });
    expect(result.current.offer).toEqual({
      repositoryId: "r1",
      repositoryName: REPO,
      count: 16,
      boardWillBeCreated: false,
    });
    act(() => result.current.dismiss());
    expect(result.current.offer).toBeNull();
  });

  it("flags that the board will be created when no config exists", async () => {
    getUnadaptedPlanFiles.mockResolvedValue([
      { repository_id: "r1", repository_name: REPO, count: 3, directories: [] },
    ]);
    getPlanFilesConfig.mockResolvedValue(null);
    const { result } = renderHook(() => usePlanAdaptationOffer("ws-1"), { wrapper: wrapper(true) });
    await act(() => result.current.check({ id: "r1", name: REPO }));
    expect(result.current.offer?.boardWillBeCreated).toBe(true);
  });

  it("offers nothing when the repository has no unadapted plans or the read fails", async () => {
    getUnadaptedPlanFiles.mockResolvedValueOnce([]);
    const { result } = renderHook(() => usePlanAdaptationOffer("ws-1"), { wrapper: wrapper(true) });
    await act(() => result.current.check({ id: "r1", name: "x" }));
    expect(result.current.offer).toBeNull();
    getUnadaptedPlanFiles.mockRejectedValueOnce(new Error("boom"));
    await act(() => result.current.check({ id: "r1", name: "x" }));
    expect(result.current.offer).toBeNull();
  });

  it("reads nothing while the planFiles flag is off", async () => {
    const { result } = renderHook(() => usePlanAdaptationOffer("ws-1"), {
      wrapper: wrapper(false),
    });
    await act(() => result.current.check({ id: "r1", name: "x" }));
    expect(getUnadaptedPlanFiles).not.toHaveBeenCalled();
    expect(result.current.offer).toBeNull();
  });
});
