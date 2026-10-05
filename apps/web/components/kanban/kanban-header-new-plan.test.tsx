import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";
import { invalidatePlanBoardConfig } from "@/hooks/domains/plans/use-plan-board";

const getPlanFilesConfig = vi.fn();
const breakpoint = { isMobile: false, isTablet: false };

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getPlanFilesConfig: (...a: unknown[]) => getPlanFilesConfig(...a),
}));
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => breakpoint,
}));
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock("@/hooks/use-kanban-display-settings", () => ({
  useKanbanDisplaySettings: () => ({
    effectiveTaskListingView: "kanban",
    onViewModeChange: vi.fn(),
    workspaces: [],
    activeWorkspaceId: "ws-1",
    repositories: [],
    selectedRepositoryId: null,
  }),
}));
vi.mock("@/hooks/use-release-notes", () => ({ useReleaseNotes: () => ({ hasNotes: false }) }));
vi.mock("@/hooks/use-system-health-indicator", () => ({
  useSystemHealthIndicator: () => ({
    hasIssues: false,
    issues: [],
    dialogOpen: false,
    openDialog: vi.fn(),
    closeDialog: vi.fn(),
  }),
}));
vi.mock("@/hooks/use-plugin-task-filters", () => ({
  usePluginTaskFilters: () => ({ filters: [], selections: {}, setFilterSelection: vi.fn() }),
}));
vi.mock("@/hooks/use-quick-chat-launcher", () => ({ useQuickChatLauncher: () => vi.fn() }));
vi.mock("@/hooks/use-quick-terminal-launcher", () => ({
  useQuickTerminalLauncher: () => vi.fn(),
}));
vi.mock("@/components/quick-chat/use-quick-chat-activity", () => ({
  useQuickChatActivity: () => ({ activity: null, label: "Quick Chat" }),
}));
vi.mock("@/components/quick-chat/quick-chat-activity-indicator", () => ({
  QuickChatActivityIndicator: () => null,
}));
vi.mock("@/components/page-topbar", () => ({
  PageTopbar: ({ center, actions }: { center?: ReactNode; actions?: ReactNode }) => (
    <header>
      {center}
      {actions}
    </header>
  ),
}));
vi.mock("@/components/actions/surface-action", () => ({ SurfaceAction: () => null }));
vi.mock("@/components/system-metrics/topbar-metrics", () => ({ TopbarMetrics: () => null }));
vi.mock("../kanban-display-dropdown", () => ({ KanbanDisplayDropdown: () => null }));
vi.mock("./kanban-header-sort", () => ({ KanbanHeaderSort: () => null }));
vi.mock("./main-top-bar-plugin-actions", () => ({ MainTopBarPluginActions: () => null }));
vi.mock("./task-search-input", () => ({ TaskSearchInput: () => null }));
vi.mock("../system-health/health-indicator", () => ({
  HealthIndicatorButton: () => null,
  HealthIssuesDialog: () => null,
}));
vi.mock("../release-notes/release-notes-dialog", () => ({ ReleaseNotesDialog: () => null }));
vi.mock("./kanban-header-mobile", () => ({
  KanbanHeaderMobile: ({ onNewPlan }: { onNewPlan?: () => void }) => (
    <div data-testid="phone-header">
      {onNewPlan && <button data-testid="phone-new-plan" onClick={onNewPlan} />}
    </div>
  ),
}));
vi.mock("./mobile-menu-sheet", () => ({
  MobileMenuSheet: ({ onNewPlan }: { onNewPlan?: () => void }) => (
    <div data-testid="tablet-menu">
      {onNewPlan && <button data-testid="tablet-menu-new-plan" onClick={onNewPlan} />}
    </div>
  ),
}));
vi.mock("./new-plan-dialog", () => ({
  NewPlanDialog: ({ open, workspaceId }: { open: boolean; workspaceId: string }) =>
    open ? <div data-testid="new-plan-dialog-open">{workspaceId}</div> : null,
}));

import { KanbanHeader } from "./kanban-header";

const WS = "ws-1";
const BOARD = "wf-plans";
const BUTTON = "new-plan-button";
const DIALOG_OPEN = "new-plan-dialog-open";

async function mount(workflowId: string | null, currentPage: "kanban" | "tasks" = "kanban") {
  render(
    <StateProvider
      initialState={{
        features: { ...defaultState.features, planFiles: true },
        workflows: { ...defaultState.workflows, activeId: workflowId },
      }}
    >
      <KanbanHeader workspaceId={WS} currentPage={currentPage} />
    </StateProvider>,
  );
  await act(async () => {
    await Promise.resolve();
  });
}

beforeEach(() => {
  breakpoint.isMobile = false;
  breakpoint.isTablet = false;
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

describe("KanbanHeader New plan action", () => {
  it("shows a labelled button on desktop only on the plan board and opens the form", async () => {
    await mount(BOARD);
    const button = screen.getByTestId(BUTTON);
    expect(button.textContent).toContain("New plan");
    expect(screen.queryByTestId(DIALOG_OPEN)).toBeNull();
    fireEvent.click(button);
    expect(screen.getByTestId(DIALOG_OPEN).textContent).toBe(WS);
  });

  it("is absent on desktop for another workflow and on the list page", async () => {
    await mount("wf-other");
    expect(screen.queryByTestId(BUTTON)).toBeNull();
    cleanup();
    await mount(BOARD, "tasks");
    expect(screen.queryByTestId(BUTTON)).toBeNull();
  });

  it("shows the compact button on tablet and the menu entry", async () => {
    breakpoint.isTablet = true;
    await mount(BOARD);
    const button = screen.getByTestId(BUTTON);
    expect(button.getAttribute("data-compact")).toBe("true");
    fireEvent.click(screen.getByTestId("tablet-menu-new-plan"));
    expect(screen.getByTestId(DIALOG_OPEN)).toBeTruthy();
  });

  it("has no tablet button for another workflow", async () => {
    breakpoint.isTablet = true;
    await mount("wf-other");
    expect(screen.queryByTestId(BUTTON)).toBeNull();
    expect(screen.queryByTestId("tablet-menu-new-plan")).toBeNull();
  });

  it("hands the action to the phone header and opens the form from there", async () => {
    breakpoint.isMobile = true;
    await mount(BOARD);
    expect(screen.queryByTestId(BUTTON)).toBeNull();
    fireEvent.click(screen.getByTestId("phone-new-plan"));
    expect(screen.getByTestId(DIALOG_OPEN)).toBeTruthy();
  });

  it("gives the phone header no action off the plan board", async () => {
    breakpoint.isMobile = true;
    await mount("wf-other");
    expect(screen.queryByTestId("phone-new-plan")).toBeNull();
  });
});
