import { describe, expect, it } from "vitest";
import type { Task } from "@/lib/types/http";
import {
  GROUP_OPTION_LABEL_KEYS,
  SORT_OPTION_LABEL_KEYS,
  TASKS_LIST_GROUP_OPTIONS,
  TASKS_LIST_SORT_OPTIONS,
  compareTasksForList,
  sortTasksByFacet,
  sortTasksForList,
  parseTasksListGroup,
  DEFAULT_TASKS_LIST_GROUP,
} from "./tasks-list-options";

// @covers AC-UI-LIST-STEP-GROUPING-001.1 and AC-UI-LIST-STEP-GROUPING-001.5
describe("workflow step grouping preferences", () => {
  it.each(["state", "workflow_step", "invalid", null, undefined, ""])(
    "resolves %s to workflow step grouping",
    (value) => expect(parseTasksListGroup(value)).toBe("workflow_step"),
  );
  it("offers workflow step as the default instead of runtime state", () => {
    expect(DEFAULT_TASKS_LIST_GROUP).toBe("workflow_step");
    expect(TASKS_LIST_GROUP_OPTIONS.map((option) => option.value)).toEqual([
      "workflow_step",
      "workflow",
      "repository",
      "none",
    ]);
  });
});

describe("SORT_OPTION_LABEL_KEYS", () => {
  it("maps every sort option to a tasks: translation key", () => {
    for (const option of TASKS_LIST_SORT_OPTIONS) {
      const key = SORT_OPTION_LABEL_KEYS[option.value];
      expect(key).toBeDefined();
      expect(key).toMatch(/^tasks:/);
    }
  });

  it("has no stray keys beyond the configured sort options", () => {
    const optionValues = new Set(TASKS_LIST_SORT_OPTIONS.map((option) => option.value));
    expect(Object.keys(SORT_OPTION_LABEL_KEYS).sort()).toEqual([...optionValues].sort());
  });
});

describe("sortTasksByFacet", () => {
  it("sorts by the first label, keeps ties stable, and leaves untagged tasks last", () => {
    const tasks = [
      { id: "one", title: "one" },
      { id: "two", title: "two" },
      { id: "three", title: "three" },
    ] as never[];
    expect(
      sortTasksByFacet(tasks, "facet:plugin:tags", {
        "facet:plugin:tags:one": [{ value: "z", label: "Zulu" }],
        "facet:plugin:tags:two": [
          { value: "a", label: "alpha" },
          { value: "z", label: "Zulu" },
        ],
      }).map((task) => task.id),
    ).toEqual(["two", "one", "three"]);
  });
});

describe("GROUP_OPTION_LABEL_KEYS", () => {
  it("maps every group option to a tasks: translation key", () => {
    for (const option of TASKS_LIST_GROUP_OPTIONS) {
      const key = GROUP_OPTION_LABEL_KEYS[option.value];
      expect(key).toBeDefined();
      expect(key).toMatch(/^tasks:/);
    }
  });

  it("has no stray keys beyond the configured group options", () => {
    const optionValues = new Set(TASKS_LIST_GROUP_OPTIONS.map((option) => option.value));
    expect(Object.keys(GROUP_OPTION_LABEL_KEYS).sort()).toEqual([...optionValues].sort());
  });
});

describe("position_asc list sort", () => {
  const task = (id: string, title: string, position: number) =>
    ({
      id,
      title,
      position,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    }) as Task;

  it("is offered with a tasks: label key", () => {
    expect(TASKS_LIST_SORT_OPTIONS.map((option) => option.value)).toContain("position_asc");
    expect(SORT_OPTION_LABEL_KEYS.position_asc).toBe("tasks:sortPositionAsc");
  });

  it("orders by position ascending and breaks ties by title", () => {
    const sorted = sortTasksForList(
      [task("c", "Gamma", 2), task("b2", "Beta", 1), task("b1", "Alpha", 1), task("a", "Zed", 0)],
      "position_asc",
    );
    expect(sorted.map((t) => t.id)).toEqual(["a", "b1", "b2", "c"]);
    expect(compareTasksForList(task("x", "A", 1), task("y", "A", 2), "position_asc")).toBeLessThan(
      0,
    );
  });
});
