import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SwimlaneHeader } from "./swimlane-header";

vi.mock("@/hooks/use-kanban-display-settings", () => ({
  useKanbanDisplaySettings: () => ({ boardSort: "position_asc", onBoardSortChange: vi.fn() }),
}));

afterEach(cleanup);

describe("SwimlaneHeader", () => {
  it("shows the board sort picker on the lane panel with the current sort", () => {
    render(
      <SwimlaneHeader
        workflowName="Flow"
        taskCount={3}
        isCollapsed={false}
        onToggleCollapse={vi.fn()}
      />,
    );
    const sort = screen.getByTestId("header-board-sort");
    expect(sort.textContent).toContain("Board order");
  });
});
