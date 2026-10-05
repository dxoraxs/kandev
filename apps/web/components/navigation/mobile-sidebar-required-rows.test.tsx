import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";

const getWaitingOwner = vi.fn();

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getWaitingOwner: (...a: unknown[]) => getWaitingOwner(...a),
}));
vi.mock("@/lib/ws/connection", () => ({ getWebSocketClient: () => null }));
vi.mock("@/hooks/use-app-destinations", () => ({ useStaticDestinations: () => [] }));
vi.mock("@/hooks/use-in-office", () => ({ useOfficeModeState: () => "kanban" }));
vi.mock("@/hooks/use-quick-chat-launcher", () => ({ useQuickChatLauncher: () => vi.fn() }));
vi.mock("@/hooks/use-quick-terminal-launcher", () => ({ useQuickTerminalLauncher: () => vi.fn() }));
vi.mock("@/hooks/domains/sidebar/use-sidebar-layout-navigation", () => ({
  useSidebarLayoutNavigation: () => ({}),
}));
vi.mock("@/components/integrations/integrations-menu", () => ({
  MobileIntegrationsSection: () => null,
}));

import { MobileRequiredRows } from "./mobile-sidebar-layout-navigation";

function mount(planFiles: boolean) {
  return render(
    <StateProvider initialState={{ features: { ...defaultState.features, planFiles } }}>
      <MobileRequiredRows onNavigate={vi.fn()} omitSections={new Set()} omitDestinations={[]} />
    </StateProvider>,
  );
}

describe("MobileRequiredRows Waiting for owner entry", () => {
  beforeEach(() => {
    getWaitingOwner.mockReset().mockResolvedValue({ items: [], failed_workspaces: [] });
  });
  afterEach(() => cleanup());

  it("offers the page in the phone navigation sheet while planFiles is on", async () => {
    mount(true);
    const row = await screen.findByTestId("mobile-sidebar-plans-waiting");
    expect(row.getAttribute("href")).toBe("/plans/waiting");
    await waitFor(() => expect(getWaitingOwner).toHaveBeenCalled());
  });

  it("omits the whole fixed block when nothing else is shown and planFiles is off", () => {
    mount(false);
    expect(screen.queryByTestId("mobile-sidebar-plans-waiting")).toBeNull();
    expect(screen.queryByTestId("mobile-sidebar-fixed-navigation")).toBeNull();
  });
});
