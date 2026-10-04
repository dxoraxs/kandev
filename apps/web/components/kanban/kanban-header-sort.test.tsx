import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KanbanHeaderSort } from "./kanban-header-sort";

const { useKanbanDisplaySettingsMock } = vi.hoisted(() => ({
  useKanbanDisplaySettingsMock: vi.fn(),
}));

vi.mock("@/hooks/use-kanban-display-settings", () => ({
  useKanbanDisplaySettings: useKanbanDisplaySettingsMock,
}));

const onBoardSortChange = vi.fn();

beforeEach(() => {
  onBoardSortChange.mockReset();
  useKanbanDisplaySettingsMock.mockReturnValue({
    boardSort: "created_desc",
    onBoardSortChange,
  });
});

afterEach(cleanup);

function openSelect() {
  const trigger = screen.getByTestId("header-board-sort");
  fireEvent.pointerDown(trigger, { button: 0, ctrlKey: false, pointerType: "mouse" });
  fireEvent.click(trigger);
}

describe("KanbanHeaderSort", () => {
  it("is visible, named by what it selects, and shows the current sort", () => {
    render(<KanbanHeaderSort />);
    const trigger = screen.getByRole("combobox", { name: "Board sort" });
    expect(trigger.textContent).toContain("Newest first");
  });

  it("offers the three sort variants", () => {
    render(<KanbanHeaderSort />);
    openSelect();
    expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual([
      "Newest first",
      "Priority",
      "Board order",
    ]);
  });

  it("changes the sort through the shared display-settings handler", () => {
    render(<KanbanHeaderSort />);
    openSelect();
    fireEvent.click(screen.getByRole("option", { name: "Board order" }));
    expect(onBoardSortChange).toHaveBeenCalledWith("position_asc");
  });
});
