import { render, screen, cleanup } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KanbanCardHintRow } from "./kanban-card-display-hints";
import type { Task } from "./kanban-card";
import type { CardDisplayHints } from "@/lib/kanban/card-display";

const DATE_TAG = "kanban-card-date-tag";
const PROGRESS_CHIP = "kanban-card-progress-chip";
const EXECUTOR_BADGE = "kanban-card-executor-badge";
const ARIA = "aria-label";
const TONE_ATTR = "data-tone";
const BASE: Task = { id: "task-1", title: "Fix the bug", workflowStepId: "step-1" };

function row(cardDisplay?: CardDisplayHints) {
  return render(<KanbanCardHintRow task={{ ...BASE, cardDisplay }} />);
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(new Date(2026, 9, 5, 12, 0, 0));
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("KanbanCardHintRow", () => {
  it("renders nothing without hints", () => {
    const { container } = row(undefined);
    expect(container.firstChild).toBeNull();
  });

  it("renders the date tag with short text, full accessible name and tone", () => {
    row({ date: { iso: "2026-10-05", kind: "waiting" } });
    const tag = screen.getByTestId(DATE_TAG);
    expect(tag.textContent).toContain("Oct 5");
    expect(tag.textContent).not.toContain("2026");
    expect(tag.getAttribute(ARIA)).toBe("Waiting until October 5, 2026");
    expect(tag.getAttribute("title")).toBe("Waiting until October 5, 2026");
    expect(tag.getAttribute(TONE_ATTR)).toBe("warning");
    expect(tag.className).toContain("bg-amber-500/15");
  });

  it("uses the neutral, danger and deferred copy and tones", () => {
    const { unmount } = row({ date: { iso: "2026-10-20", kind: "due" } });
    let tag = screen.getByTestId(DATE_TAG);
    expect(tag.getAttribute(TONE_ATTR)).toBe("neutral");
    expect(tag.className).toContain("bg-muted");
    expect(tag.getAttribute(ARIA)).toBe("Due October 20, 2026");
    unmount();

    const overdue = row({ date: { iso: "2026-10-01", kind: "due" } });
    tag = screen.getByTestId(DATE_TAG);
    expect(tag.getAttribute(TONE_ATTR)).toBe("danger");
    expect(tag.className).toContain("bg-red-500/15");
    overdue.unmount();

    row({ date: { iso: "2026-10-01", kind: "deferred" } });
    tag = screen.getByTestId(DATE_TAG);
    expect(tag.getAttribute(TONE_ATTR)).toBe("warning");
    expect(tag.getAttribute(ARIA)).toBe("Deferred until October 1, 2026");
  });

  it("adds the year when it differs from the current year", () => {
    row({ date: { iso: "2027-01-12", kind: "due" } });
    expect(screen.getByTestId(DATE_TAG).textContent).toContain("2027");
  });

  it("renders the progress chip with plural-aware accessible name", () => {
    const { unmount } = row({ progress: { done: 1, total: 3 } });
    let chip = screen.getByTestId(PROGRESS_CHIP);
    expect(chip.textContent).toContain("1/3");
    expect(chip.getAttribute(ARIA)).toBe("1 of 3 steps done");
    expect(chip.getAttribute("data-complete")).toBe("false");
    unmount();

    row({ progress: { done: 1, total: 1 } });
    chip = screen.getByTestId(PROGRESS_CHIP);
    expect(chip.getAttribute(ARIA)).toBe("1 of 1 step done");
    expect(chip.getAttribute("data-complete")).toBe("true");
    expect(chip.className).toContain("bg-emerald-500/15");
  });

  it("renders an agent executor as an uppercased initial", () => {
    row({ executor: { name: "claude", kind: "agent" } });
    const badge = screen.getByTestId(EXECUTOR_BADGE);
    expect(badge.textContent).toBe("C");
    expect(badge.getAttribute("data-kind")).toBe("agent");
    expect(badge.getAttribute(ARIA)).toBe("Executor: claude");
    expect(badge.getAttribute("title")).toBe("Executor: claude");
  });

  it("renders a person executor as an icon without initial text", () => {
    row({ executor: { name: "Valentin", kind: "person" } });
    const badge = screen.getByTestId(EXECUTOR_BADGE);
    expect(badge.textContent).toBe("");
    expect(badge.querySelector("svg")).not.toBeNull();
    expect(badge.getAttribute("data-kind")).toBe("person");
  });

  it("renders only the fields present, executor pushed right", () => {
    row({ executor: { name: "Claude", kind: "agent" } });
    expect(screen.getByTestId("kanban-card-hint-row")).not.toBeNull();
    expect(screen.queryByTestId(DATE_TAG)).toBeNull();
    expect(screen.queryByTestId(PROGRESS_CHIP)).toBeNull();
    expect(screen.getByTestId(EXECUTOR_BADGE).className).toContain("ml-auto");
  });
});

describe("KanbanCardHintRow flags", () => {
  it("renders each flag as a warning tag with a visible label and an accessible name", () => {
    row({ flags: ["stale", "open_items", "uncommitted"] });
    const stale = screen.getByTestId("kanban-card-flag-stale");
    expect(stale.textContent).toBe("Stale");
    expect(stale.getAttribute(ARIA)).toBe("Stale: no recent activity on this plan");
    expect(stale.className).toContain("bg-amber-500/15");
    const open = screen.getByTestId("kanban-card-flag-open_items");
    expect(open.textContent).toBe("Open items");
    expect(open.getAttribute(ARIA)).toBe("Open items: this plan is done but items are still open");
    const uncommitted = screen.getByTestId("kanban-card-flag-uncommitted");
    expect(uncommitted.textContent).toBe("Uncommitted");
    expect(uncommitted.getAttribute("title")).toBe(
      "Uncommitted: this plan file has uncommitted changes",
    );
    expect(screen.getAllByRole("img")).toHaveLength(3);
  });

  it("places the flags after the date and progress and keeps the executor on the right", () => {
    row({
      date: { iso: "2026-10-05", kind: "due" },
      progress: { done: 3, total: 8 },
      flags: ["stale"],
      executor: { name: "Claude", kind: "agent" },
    });
    const order = Array.from(screen.getByTestId("kanban-card-hint-row").children).map((el) =>
      el.getAttribute("data-testid"),
    );
    expect(order).toEqual([DATE_TAG, PROGRESS_CHIP, "kanban-card-flag-stale", EXECUTOR_BADGE]);
    expect(screen.getByTestId(EXECUTOR_BADGE).className).toContain("ml-auto");
  });

  it("renders the hint row for flags alone", () => {
    row({ flags: ["uncommitted"] });
    expect(screen.getByTestId("kanban-card-hint-row")).not.toBeNull();
    expect(screen.queryByTestId("kanban-card-flag-stale")).toBeNull();
  });

  it("exposes each hint as an image with an accessible name", () => {
    row({
      date: { iso: "2026-10-05", kind: "waiting" },
      progress: { done: 1, total: 3 },
      executor: { name: "claude", kind: "agent" },
    });
    expect(screen.getByRole("img", { name: "Waiting until October 5, 2026" })).not.toBeNull();
    expect(screen.getByRole("img", { name: "1 of 3 steps done" })).not.toBeNull();
    expect(screen.getByRole("img", { name: "Executor: claude" })).not.toBeNull();
  });
});
