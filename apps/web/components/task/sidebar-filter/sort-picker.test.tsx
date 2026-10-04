import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SortPicker } from "./sort-picker";

afterEach(cleanup);

describe("SortPicker", () => {
  it("offers the board-order sort and reports it when chosen", () => {
    const onChange = vi.fn();
    render(<SortPicker value={{ key: "state", direction: "asc" }} onChange={onChange} />);
    const trigger = screen.getByTestId("sort-key-select");
    fireEvent.pointerDown(trigger, { button: 0, ctrlKey: false, pointerType: "mouse" });
    fireEvent.click(trigger);
    fireEvent.click(screen.getByRole("option", { name: /Board order/ }));
    expect(onChange).toHaveBeenCalledWith({ key: "position", direction: "asc" });
  });
});
