import { describe, expect, it } from "vitest";
import {
  compareTasksByDateThenBoardOrder,
  pickKanbanColumnComparator,
  sortIdsByDisplayOrder,
  sortTasksForPipelineView,
  type DisplayOrderTask,
} from "./task-order";

type Dated = {
  id: string;
  position?: number;
  workflowStepId?: string;
  cardDisplay?: { date?: { iso: string; kind: "waiting" | "due" | "deferred" } };
};

function task(id: string, position: number, iso?: string, workflowStepId = TODO): Dated {
  return {
    id,
    position,
    workflowStepId,
    cardDisplay: iso ? { date: { iso, kind: "due" } } : undefined,
  };
}

const SOON = "2026-10-06";
const LATE = "2026-12-01";
const REVIEW = "review";
const TODO = "todo";
const REVIEW_SOON = "review-soon";
const DATE_ASC = "date_asc" as const;

const ids = (tasks: { id: string }[]) => tasks.map((t) => t.id);

describe("compareTasksByDateThenBoardOrder", () => {
  it("puts dated tasks first in ascending date order", () => {
    const tasks = [
      task("late", 0, LATE),
      task("none", 1),
      task("soon", 2, SOON),
      task("mid", 3, "2026-11-15"),
    ];
    expect(ids([...tasks].sort(compareTasksByDateThenBoardOrder))).toEqual([
      "soon",
      "mid",
      "late",
      "none",
    ]);
  });

  it("keeps undated tasks in board order", () => {
    const tasks = [task("c", 2), task("dated", 9, SOON), task("a", 0), task("b", 1)];
    expect(ids([...tasks].sort(compareTasksByDateThenBoardOrder))).toEqual([
      "dated",
      "a",
      "b",
      "c",
    ]);
  });

  it("breaks a date tie by board order", () => {
    const tasks = [task("second", 5, SOON), task("first", 1, SOON)];
    expect(ids([...tasks].sort(compareTasksByDateThenBoardOrder))).toEqual(["first", "second"]);
  });

  it("treats an invalid date as undated", () => {
    const tasks = [
      task("bad-month", 0, "2026-13-01"),
      task("not-a-date", 1, "tomorrow"),
      task("ok", 2, SOON),
      task("none", 3),
    ];
    expect(ids([...tasks].sort(compareTasksByDateThenBoardOrder))).toEqual([
      "ok",
      "bad-month",
      "not-a-date",
      "none",
    ]);
  });
});

describe("date_asc selection", () => {
  const tasks = [task("none", 0), task("late", 1, LATE), task("soon", 2, SOON)];

  it("picks the date comparator for the column surfaces", () => {
    expect(pickKanbanColumnComparator(DATE_ASC)).toBe(compareTasksByDateThenBoardOrder);
    expect(ids([...tasks].sort(pickKanbanColumnComparator(DATE_ASC)))).toEqual([
      "soon",
      "late",
      "none",
    ]);
  });

  it("orders the pipeline view by step first, then date, then board order", () => {
    const pipeline = [
      task("review-none", 0, undefined, REVIEW),
      task("todo-none-b", 1, undefined, TODO),
      task("todo-late", 2, LATE, TODO),
      task("todo-none-a", 0, undefined, TODO),
      task(REVIEW_SOON, 3, SOON, REVIEW),
    ] as (Dated & { workflowStepId: string })[];
    const steps = [{ id: TODO }, { id: REVIEW }];
    expect(ids(sortTasksForPipelineView(pipeline, steps, DATE_ASC))).toEqual([
      "todo-late",
      "todo-none-a",
      "todo-none-b",
      REVIEW_SOON,
      "review-none",
    ]);
  });

  it("orders selected ids in the kanban and pipeline views", () => {
    const byId = new Map<string, DisplayOrderTask>(tasks.map((t) => [t.id, t]));
    expect(
      sortIdsByDisplayOrder(["none", "late", "soon"], byId, {
        sortToken: DATE_ASC,
        isPipelineView: false,
      }),
    ).toEqual(["soon", "late", "none"]);

    const stepIndexOf = (stepId: string | undefined) => (stepId === REVIEW ? 1 : 0);
    const piped = new Map<string, DisplayOrderTask>(
      [
        task(REVIEW_SOON, 0, SOON, REVIEW),
        task("todo-none", 0, undefined, TODO),
        task("todo-late", 1, LATE, TODO),
      ].map((t) => [t.id, t]),
    );
    expect(
      sortIdsByDisplayOrder([REVIEW_SOON, "todo-none", "todo-late"], piped, {
        sortToken: DATE_ASC,
        isPipelineView: true,
        stepIndexOf,
      }),
    ).toEqual(["todo-late", "todo-none", REVIEW_SOON]);
  });
});
