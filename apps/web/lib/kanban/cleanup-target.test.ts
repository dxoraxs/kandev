import { describe, expect, it } from "vitest";
import { resolveCleanupTarget } from "./cleanup-target";

type Repo = { id: string; name: string; source_type: string };

const local = (id: string): Repo => ({ id, name: id, source_type: "local" });
const remote = (id: string): Repo => ({ id, name: id, source_type: "github" });

describe("resolveCleanupTarget", () => {
  it("is hidden when the workspace has no local repository", () => {
    expect(resolveCleanupTarget([remote("r")], null)).toEqual({ kind: "hidden" });
    expect(resolveCleanupTarget([], null)).toEqual({ kind: "hidden" });
  });

  it("targets the selected local repository", () => {
    expect(resolveCleanupTarget([local("a"), local("b")], "b")).toEqual({
      kind: "ready",
      repository: local("b"),
    });
  });

  it("targets the only local repository when nothing is selected", () => {
    expect(resolveCleanupTarget([local("a"), remote("r")], null)).toEqual({
      kind: "ready",
      repository: local("a"),
    });
  });

  it("needs a selection when several local repositories exist and none is selected", () => {
    expect(resolveCleanupTarget([local("a"), local("b")], null)).toEqual({
      kind: "needs_selection",
    });
  });

  it("needs a selection when the selected repository is not local", () => {
    expect(resolveCleanupTarget([local("a"), remote("r")], "r")).toEqual({
      kind: "needs_selection",
    });
  });

  it("falls back to the only local repository when the selection is unknown", () => {
    expect(resolveCleanupTarget([local("a")], "gone")).toEqual({
      kind: "ready",
      repository: local("a"),
    });
  });
});
