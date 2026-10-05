import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { MobileDisplayOptions, type MobileDisplayOptionsProps } from "./mobile-display-options";

const BASE: MobileDisplayOptionsProps = {
  activeWorkflowId: null,
  workflows: [],
  onWorkflowChange: vi.fn(),
  repositoryValue: "all",
  repositories: [],
  repositoriesLoading: false,
  onRepositoryChange: vi.fn(),
  enablePreviewOnClick: undefined,
  onTogglePreviewOnClick: undefined,
  tasksListShowDetails: false,
  onToggleTasksListShowDetails: vi.fn(),
  showTaskDetails: false,
  showWorkflow: false,
  showRepository: false,
  showPreviewPanel: false,
  columnsSection: null,
  showBoardControls: false,
  boardSort: "created_desc",
  onBoardSortChange: vi.fn(),
  priorityFilterTokens: [],
  onPriorityFilterChange: vi.fn(),
  pluginFilters: [],
  pluginFilterSelections: {},
  onPluginFilterChange: vi.fn(),
};

function mount(onNewPlan?: () => void) {
  render(
    <StateProvider>
      <MobileDisplayOptions open {...BASE} onNewPlan={onNewPlan} />
    </StateProvider>,
  );
}

afterEach(() => cleanup());

describe("MobileDisplayOptions New plan entry", () => {
  it("shows a full-width touch entry that starts the New plan form", () => {
    const onNewPlan = vi.fn();
    mount(onNewPlan);
    const entry = screen.getByTestId("mobile-display-new-plan");
    expect(entry.textContent).toContain("New plan");
    expect(entry.className).toContain("min-h-11");
    expect(entry.className).toContain("w-full");
    fireEvent.click(entry);
    expect(onNewPlan).toHaveBeenCalledTimes(1);
  });

  it("has no entry when the board is not the plan board", () => {
    mount();
    expect(screen.queryByTestId("mobile-display-new-plan")).toBeNull();
  });
});
