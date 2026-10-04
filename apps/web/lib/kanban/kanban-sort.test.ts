import { describe, expect, it } from "vitest";
import { DEFAULT_KANBAN_SORT, KANBAN_SORT_OPTIONS, parseKanbanSort } from "./kanban-sort";

describe("parseKanbanSort", () => {
  it("accepts the defined tokens verbatim", () => {
    expect(parseKanbanSort("created_desc")).toBe("created_desc");
    expect(parseKanbanSort("priority_desc")).toBe("priority_desc");
    expect(parseKanbanSort("position_asc")).toBe("position_asc");
    expect(parseKanbanSort(" position_asc ")).toBe("position_asc");
  });

  it("resolves undefined, null and empty to the default", () => {
    expect(parseKanbanSort(undefined)).toBe(DEFAULT_KANBAN_SORT);
    expect(parseKanbanSort(null)).toBe(DEFAULT_KANBAN_SORT);
    expect(parseKanbanSort("")).toBe(DEFAULT_KANBAN_SORT);
  });

  it("resolves an unrecognized value to the default rather than failing", () => {
    expect(parseKanbanSort("priority_asc")).toBe(DEFAULT_KANBAN_SORT);
    expect(parseKanbanSort("garbage")).toBe(DEFAULT_KANBAN_SORT);
  });

  it("trims surrounding whitespace before matching", () => {
    expect(parseKanbanSort(" priority_desc")).toBe("priority_desc");
    expect(parseKanbanSort("created_desc ")).toBe("created_desc");
    expect(parseKanbanSort("  priority_desc  ")).toBe("priority_desc");
  });

  it("does not resolve a value that only matches after case-folding", () => {
    expect(parseKanbanSort("CREATED_DESC")).toBe(DEFAULT_KANBAN_SORT);
  });
});

describe("KANBAN_SORT_OPTIONS", () => {
  it("presents exactly the defined tokens", () => {
    expect(KANBAN_SORT_OPTIONS.map((option) => option.value)).toEqual([
      "created_desc",
      "priority_desc",
      "position_asc",
    ]);
  });
});
