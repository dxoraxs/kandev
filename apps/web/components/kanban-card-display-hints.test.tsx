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
