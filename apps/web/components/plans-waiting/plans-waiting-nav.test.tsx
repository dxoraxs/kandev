import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { TooltipProvider } from "@kandev/ui/tooltip";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";

const getWaitingOwner = vi.fn();

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getWaitingOwner: (...a: unknown[]) => getWaitingOwner(...a),
}));
vi.mock("@/lib/ws/connection", () => ({ getWebSocketClient: () => null }));
vi.mock("@/lib/routing/client-router", () => ({ usePathname: () => "/", useRouter: () => ({}) }));

import { PlansWaitingMobileRow, PlansWaitingSidebarItem } from "./plans-waiting-nav";

function plan(id: string) {
  return {
    workspace_id: "ws-1",
    workspace_name: "kandev",
    task_id: id,
    title: id,
    repository_name: "kandev",
    rel_path: `docs/plans/${id}.md`,
    date: "",
    executor: "",
    priority: "medium",
  };
}

function mount(node: React.ReactNode, planFiles: boolean) {
  return render(
    <StateProvider initialState={{ features: { ...defaultState.features, planFiles } }}>
      <TooltipProvider>{node}</TooltipProvider>
    </StateProvider>,
  );
}

async function settle() {
  await act(async () => {
    await Promise.resolve();
  });
}

describe("PlansWaitingSidebarItem", () => {
  beforeEach(() => {
    getWaitingOwner
      .mockReset()
      .mockResolvedValue({ items: [plan("a"), plan("b")], failed_workspaces: [] });
  });
  afterEach(() => cleanup());

  it("links to the page with the number of waiting plans", async () => {
    mount(<PlansWaitingSidebarItem collapsed={false} />, true);
    const link = await screen.findByTestId("sidebar-plans-waiting");
    expect(link.getAttribute("href")).toBe("/plans/waiting");
    expect(link.getAttribute("aria-label")).toBe("Waiting for owner");
    await waitFor(() => expect(link.textContent).toContain("2"));
  });

  it("shows no badge when nothing waits", async () => {
    getWaitingOwner.mockResolvedValue({ items: [], failed_workspaces: [] });
    mount(<PlansWaitingSidebarItem collapsed={false} />, true);
    const link = await screen.findByTestId("sidebar-plans-waiting");
    await settle();
    expect(link.textContent).not.toMatch(/\d/);
  });

  it("is absent and reads nothing while the planFiles flag is off", async () => {
    mount(<PlansWaitingSidebarItem collapsed={false} />, false);
    await settle();
    expect(screen.queryByTestId("sidebar-plans-waiting")).toBeNull();
    expect(getWaitingOwner).not.toHaveBeenCalled();
  });
});

describe("PlansWaitingMobileRow", () => {
  beforeEach(() => {
    getWaitingOwner.mockReset().mockResolvedValue({ items: [plan("a")], failed_workspaces: [] });
  });
  afterEach(() => cleanup());

  it("is a full-width row at least 44px tall that links to the page and closes the sheet", async () => {
    const onNavigate = vi.fn();
    mount(<PlansWaitingMobileRow onNavigate={onNavigate} />, true);
    const row = await screen.findByTestId("mobile-sidebar-plans-waiting");
    expect(row.getAttribute("href")).toBe("/plans/waiting");
    expect(row.className).toContain("h-11");
    expect(row.className).toContain("w-full");
    expect(row.textContent).toContain("Waiting for owner");
    await waitFor(() => expect(row.textContent).toContain("1"));
    row.click();
    expect(onNavigate).toHaveBeenCalledTimes(1);
  });

  it("is absent while the planFiles flag is off", async () => {
    mount(<PlansWaitingMobileRow onNavigate={vi.fn()} />, false);
    await settle();
    expect(screen.queryByTestId("mobile-sidebar-plans-waiting")).toBeNull();
    expect(getWaitingOwner).not.toHaveBeenCalled();
  });
});
