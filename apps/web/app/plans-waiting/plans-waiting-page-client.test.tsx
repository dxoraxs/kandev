import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import type { WaitingOwnerState } from "@/hooks/domains/plans/use-waiting-owner";
import type { WaitingOwnerItem } from "@/lib/api/domains/plan-files-api";

const push = vi.fn();
const reload = vi.fn();
let waiting: WaitingOwnerState;
let isMobile = false;

vi.mock("@/hooks/domains/plans/use-waiting-owner", () => ({ useWaitingOwner: () => waiting }));
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile }),
}));
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ push }) }));
vi.mock("@/components/routing/app-link", () => ({
  default: ({ href, children, ...rest }: { href: string; children: ReactNode }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));
vi.mock("@/components/page-shell", () => ({
  PageShell: ({ title, children }: { title: string; children: ReactNode }) => (
    <div data-testid="stub-page-shell" data-title={title}>
      {children}
    </div>
  ),
}));

import { PlansWaitingPageClient } from "./plans-waiting-page-client";

function item(overrides: Partial<WaitingOwnerItem> = {}): WaitingOwnerItem {
  return {
    workspace_id: "ws-1",
    workspace_name: "dmhive",
    task_id: "t-1",
    title: "06a Instagram data deletion",
    repository_name: "dmhive",
    rel_path: "docs/plans/06a.md",
    date: "2026-10-05",
    executor: "Claude",
    priority: "medium",
    ...overrides,
  };
}

function state(overrides: Partial<WaitingOwnerState> = {}): WaitingOwnerState {
  return { status: "ready", items: [], failedWorkspaces: [], reload, ...overrides };
}

const TWO = [
  item(),
  item({
    task_id: "t-2",
    workspace_name: "kandev",
    title: "Plan board operations",
    repository_name: "kandev",
    date: "",
    executor: "",
  }),
];

describe("PlansWaitingPageClient desktop", () => {
  beforeEach(() => {
    isMobile = false;
    push.mockReset();
    reload.mockReset();
    waiting = state({ items: TWO });
  });
  afterEach(() => cleanup());

  it("titles the page with the count and lists one table row per plan in order", () => {
    render(<PlansWaitingPageClient />);
    expect(screen.getByTestId("stub-page-shell").dataset.title).toBe("Waiting for owner (2)");
    const table = screen.getByTestId("plans-waiting-table");
    const headers = within(table)
      .getAllByRole("columnheader")
      .map((h) => h.textContent);
    expect(headers).toEqual(["Workspace", "Plan", "Repository", "Date", "Executor"]);
    const rows = within(table).getAllByTestId(/^plans-waiting-row-/);
    expect(rows.map((r) => r.dataset.testid)).toEqual([
      "plans-waiting-row-t-1",
      "plans-waiting-row-t-2",
    ]);
    const first = within(rows[0]!);
    expect(first.getByText("dmhive", { selector: "td:first-child" })).toBeTruthy();
    expect(first.getByText("06a Instagram data deletion")).toBeTruthy();
    expect(first.getByText("Claude")).toBeTruthy();
    expect(first.getByTestId("plans-waiting-date").textContent).toMatch(/Oct/);
    expect(within(rows[1]!).queryByTestId("plans-waiting-date")).toBeNull();
    expect(screen.queryByTestId("plans-waiting-list")).toBeNull();
  });

  it("opens the task from the title link and from a click on the row", () => {
    render(<PlansWaitingPageClient />);
    const link = screen.getByRole("link", { name: "06a Instagram data deletion" });
    expect(link.getAttribute("href")).toBe("/t/t-1");
    fireEvent.click(screen.getByTestId("plans-waiting-row-t-2"));
    expect(push).toHaveBeenCalledWith("/t/t-2");
  });

  it("names the failed workspaces while still listing the others", () => {
    waiting = state({
      items: TWO,
      failedWorkspaces: [
        { workspace_id: "ws-9", workspace_name: "tg-hub" },
        { workspace_id: "ws-8", workspace_name: "other" },
      ],
    });
    render(<PlansWaitingPageClient />);
    expect(screen.getByTestId("plans-waiting-failed").textContent).toBe(
      "Could not load: tg-hub, other",
    );
    expect(screen.getAllByTestId(/^plans-waiting-row-/)).toHaveLength(2);
  });

  it("shows the empty state when nothing waits", () => {
    waiting = state();
    render(<PlansWaitingPageClient />);
    expect(screen.getByTestId("plans-waiting-empty").textContent).toBe(
      "No plans are waiting for you.",
    );
    expect(screen.queryByTestId("plans-waiting-table")).toBeNull();
  });

  it("shows the empty state next to the failed workspaces when only failures exist", () => {
    waiting = state({ failedWorkspaces: [{ workspace_id: "w", workspace_name: "tg-hub" }] });
    render(<PlansWaitingPageClient />);
    expect(screen.getByTestId("plans-waiting-failed")).toBeTruthy();
    expect(screen.getByTestId("plans-waiting-empty")).toBeTruthy();
  });

  it("shows a loading status before the first read", () => {
    waiting = state({ status: "loading" });
    render(<PlansWaitingPageClient />);
    expect(screen.getByRole("status").textContent).toBe("Loading…");
    expect(screen.getByTestId("stub-page-shell").dataset.title).toBe("Waiting for owner");
  });

  it("offers a retry when the first read failed", () => {
    waiting = state({ status: "error" });
    render(<PlansWaitingPageClient />);
    expect(screen.getByTestId("plans-waiting-error")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("keeps the last list and warns when a later read failed", () => {
    waiting = state({ status: "error", items: TWO });
    render(<PlansWaitingPageClient />);
    expect(screen.getAllByTestId(/^plans-waiting-row-/)).toHaveLength(2);
    expect(screen.getByTestId("plans-waiting-error")).toBeTruthy();
  });
});

describe("PlansWaitingPageClient phone", () => {
  beforeEach(() => {
    isMobile = true;
    push.mockReset();
    waiting = state({ items: TWO });
  });
  afterEach(() => cleanup());

  it("renders one full-width link row per plan, at least 44px tall, with details on a second line", () => {
    render(<PlansWaitingPageClient />);
    expect(screen.queryByTestId("plans-waiting-table")).toBeNull();
    const list = screen.getByTestId("plans-waiting-list");
    const rows = within(list).getAllByTestId(/^plans-waiting-row-/);
    expect(rows).toHaveLength(2);
    for (const row of rows) {
      expect(row.tagName).toBe("A");
      expect(row.className).toContain("min-h-11");
      expect(row.className).toContain("w-full");
    }
    expect(rows[0]!.getAttribute("href")).toBe("/t/t-1");
    const meta = within(rows[0]!).getByTestId("plans-waiting-meta").textContent ?? "";
    expect(meta).toContain("dmhive");
    expect(meta).toContain("Claude");
    expect(meta).toMatch(/Oct/);
    const bare = within(rows[1]!).getByTestId("plans-waiting-meta").textContent ?? "";
    expect(bare).toBe("kandev");
  });

  it("names failed workspaces and keeps the retry control at least 44px tall", () => {
    waiting = state({
      status: "error",
      failedWorkspaces: [{ workspace_id: "ws-9", workspace_name: "tg-hub" }],
    });
    render(<PlansWaitingPageClient />);
    expect(screen.getByTestId("plans-waiting-failed").textContent).toContain("tg-hub");
    expect(screen.getByRole("button", { name: "Try again" }).className).toContain("min-h-11");
  });
});
