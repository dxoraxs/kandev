import { describe, expect, it } from "vitest";
import { resolveSessionlessTaskView } from "./sessionless-task-view-model";

// @covers AC-TASKS-TASK-OPEN-WITHOUT-AGENT-002.2, 003.3, 003.6
// @covers AC-TASKS-TASK-DESCRIPTION-VIEW-003.1, 003.3
describe("resolveSessionlessTaskView", () => {
  it("shows the description and no status line while idle", () => {
    expect(
      resolveSessionlessTaskView({ description: "# Plan", status: "idle", workspaceId: "ws" }),
    ).toEqual({ description: "# Plan", statusLine: "none", settingsWorkspaceId: null });
  });

  it("treats a blank description as missing", () => {
    expect(
      resolveSessionlessTaskView({ description: "  \n ", status: "idle", workspaceId: null })
        .description,
    ).toBeNull();
    expect(
      resolveSessionlessTaskView({ description: undefined, status: "idle", workspaceId: null })
        .description,
    ).toBeNull();
  });

  it("shows the preparing line while ensure runs", () => {
    expect(
      resolveSessionlessTaskView({ description: "x", status: "preparing", workspaceId: "ws" })
        .statusLine,
    ).toBe("preparing");
  });

  it("shows the unassigned notice with the workspace settings target", () => {
    expect(
      resolveSessionlessTaskView({ description: "x", status: "unassigned", workspaceId: "ws-1" }),
    ).toEqual({ description: "x", statusLine: "unassigned", settingsWorkspaceId: "ws-1" });
  });

  it("omits the settings target when the workspace is unknown", () => {
    expect(
      resolveSessionlessTaskView({ description: "x", status: "unassigned", workspaceId: null })
        .settingsWorkspaceId,
    ).toBeNull();
  });

  it("leaves real ensure errors to the page-level banner", () => {
    expect(
      resolveSessionlessTaskView({ description: "x", status: "error", workspaceId: "ws" })
        .statusLine,
    ).toBe("none");
  });
});
