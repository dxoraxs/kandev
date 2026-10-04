import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ApiError } from "@/lib/api/client";

const getPlanGitStatus = vi.fn();
const commitPlanFiles = vi.fn();
const breakpoint = { isMobile: false };

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getPlanGitStatus: (...a: unknown[]) => getPlanGitStatus(...a),
  commitPlanFiles: (...a: unknown[]) => commitPlanFiles(...a),
}));
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile: breakpoint.isMobile }),
}));

import { PlanFilesGit } from "./plan-files-git";

const DOCS_A = "docs/plans/a.md";
const COMMIT_DIALOG = "plan-files-commit-dialog";
const REPOS = [
  {
    repository_id: "r1",
    repository_name: "dmhive",
    files: [DOCS_A, "docs/plans/INDEX.md", "docs/plans/my plan.md"],
  },
  { repository_id: "r2", repository_name: "kandev", files: [] },
];
const COMMIT_BUTTON = "plan-files-git-commit";
const CONFIRM = "plan-files-commit-confirm";
const MESSAGE = "plan-files-commit-message";
const ERROR = "plan-files-commit-error";
const DEFAULT_MESSAGE = "docs(plans): update plan files";

function rejection(status: number, body: Record<string, unknown>) {
  return new ApiError(String(body.error ?? "failed"), status, body);
}

async function openDialog() {
  render(<PlanFilesGit workspaceId="ws-1" />);
  fireEvent.click(await screen.findByTestId(COMMIT_BUTTON));
  return screen.findByTestId(breakpoint.isMobile ? "plan-files-commit-drawer" : COMMIT_DIALOG);
}

beforeEach(() => {
  breakpoint.isMobile = false;
  getPlanGitStatus.mockReset();
  commitPlanFiles.mockReset();
  getPlanGitStatus.mockResolvedValue(REPOS);
});

afterEach(cleanup);

describe("PlanFilesGit rows", () => {
  it("lists each repository with its count and offers Commit only when files changed", async () => {
    render(<PlanFilesGit workspaceId="ws-1" />);
    const rows = await screen.findAllByTestId("plan-files-git-row");
    expect(rows).toHaveLength(2);
    expect(rows[0]!.textContent).toContain("dmhive");
    expect(rows[0]!.querySelector('[data-testid="plan-files-git-count"]')!.textContent).toBe(
      "3 files",
    );
    expect(rows[1]!.querySelector('[data-testid="plan-files-git-count"]')!.textContent).toBe(
      "0 files",
    );
    expect(screen.getAllByTestId(COMMIT_BUTTON)).toHaveLength(1);
    expect(screen.getByTestId(COMMIT_BUTTON).textContent).toBe("Commit plan files");
    expect(getPlanGitStatus).toHaveBeenCalledWith({ workspaceId: "ws-1" });
  });

  it("uses the singular for one file", async () => {
    getPlanGitStatus.mockResolvedValue([{ ...REPOS[0], files: [DOCS_A] }]);
    render(<PlanFilesGit workspaceId="ws-1" />);
    expect((await screen.findByTestId("plan-files-git-count")).textContent).toBe("1 file");
  });

  it("renders nothing when there are no repositories or the read fails", async () => {
    getPlanGitStatus.mockResolvedValue([]);
    const { container, rerender } = render(<PlanFilesGit workspaceId="ws-1" />);
    await waitFor(() => expect(getPlanGitStatus).toHaveBeenCalledTimes(1));
    expect(container.firstChild).toBeNull();
    getPlanGitStatus.mockRejectedValue(new Error("boom"));
    rerender(<PlanFilesGit workspaceId="ws-1" refreshKey="later" />);
    await waitFor(() => expect(getPlanGitStatus).toHaveBeenCalledTimes(2));
    expect(container.firstChild).toBeNull();
  });
});

describe("PlanFilesGit commit dialog", () => {
  it("shows the files and the default message, and commits the repository", async () => {
    commitPlanFiles.mockResolvedValue({ commit: "abc", files: REPOS[0]!.files });
    const dialog = await openDialog();
    expect(dialog.textContent).toContain("docs/plans/my plan.md");
    const input = screen.getByTestId(MESSAGE) as HTMLInputElement;
    expect(input.value).toBe(DEFAULT_MESSAGE);
    fireEvent.change(input, { target: { value: "  plans: tidy up  " } });
    getPlanGitStatus.mockResolvedValue([{ ...REPOS[0], files: [] }, REPOS[1]]);
    fireEvent.click(screen.getByTestId(CONFIRM));

    await waitFor(() =>
      expect(commitPlanFiles).toHaveBeenCalledWith(
        { repository_id: "r1", message: "plans: tidy up" },
        { workspaceId: "ws-1" },
      ),
    );
    await waitFor(() => expect(screen.queryByTestId(COMMIT_DIALOG)).toBeNull());
    await waitFor(() => expect(screen.queryByTestId(COMMIT_BUTTON)).toBeNull());
    expect(getPlanGitStatus).toHaveBeenCalledTimes(2);
  });

  it("keeps the dialog open and shows the reason with the git output when the commit fails", async () => {
    commitPlanFiles.mockRejectedValue(
      rejection(422, { error: "failed", code: "commit_failed", output: "plan lint failed" }),
    );
    await openDialog();
    fireEvent.click(screen.getByTestId(CONFIRM));

    const error = await screen.findByTestId(ERROR);
    expect(error.textContent).toContain("Git could not create the commit. Nothing was changed.");
    expect(screen.getByTestId("plan-files-commit-output").textContent).toBe("plan lint failed");
    expect(screen.getByTestId(COMMIT_DIALOG)).toBeTruthy();
  });

  it("explains a repository in the middle of a merge", async () => {
    commitPlanFiles.mockRejectedValue(rejection(409, { error: "busy", code: "repository_busy" }));
    await openDialog();
    fireEvent.click(screen.getByTestId(CONFIRM));

    const error = await screen.findByTestId(ERROR);
    expect(error.textContent).toContain("in the middle of a merge, rebase, cherry-pick, or revert");
    expect(screen.queryByTestId("plan-files-commit-output")).toBeNull();
  });

  it("re-reads the state and says so when nothing is left to commit", async () => {
    commitPlanFiles.mockRejectedValue(rejection(409, { error: "none", code: "nothing_to_commit" }));
    await openDialog();
    fireEvent.click(screen.getByTestId(CONFIRM));

    expect((await screen.findByTestId(ERROR)).textContent).toContain("any more");
    await waitFor(() => expect(getPlanGitStatus).toHaveBeenCalledTimes(2));
  });

  it("shows a generic message for an unexpected failure and clears it when reopened", async () => {
    commitPlanFiles.mockRejectedValue(new Error("network down"));
    await openDialog();
    fireEvent.click(screen.getByTestId(CONFIRM));
    expect((await screen.findByTestId(ERROR)).textContent).toContain(
      "The commit could not be created. Try again.",
    );

    fireEvent.click(screen.getByTestId("plan-files-commit-cancel"));
    await waitFor(() => expect(screen.queryByTestId(COMMIT_DIALOG)).toBeNull());
    fireEvent.click(screen.getByTestId(COMMIT_BUTTON));
    await screen.findByTestId(COMMIT_DIALOG);
    expect(screen.queryByTestId(ERROR)).toBeNull();
  });

  it("disables the confirm button while the commit runs", async () => {
    let resolveCommit: (value: unknown) => void = () => {};
    commitPlanFiles.mockReturnValue(new Promise((resolve) => (resolveCommit = resolve)));
    await openDialog();
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() =>
      expect((screen.getByTestId(CONFIRM) as HTMLButtonElement).disabled).toBe(true),
    );
    resolveCommit({ commit: "abc", files: [] });
    await waitFor(() => expect(screen.queryByTestId(COMMIT_DIALOG)).toBeNull());
  });
});

describe("PlanFilesGit on a phone", () => {
  it("opens the commit form in a bottom drawer with full-width touch buttons", async () => {
    breakpoint.isMobile = true;
    commitPlanFiles.mockResolvedValue({ commit: "abc", files: [] });
    const drawer = await openDialog();
    expect(drawer.textContent).toContain(DOCS_A);
    expect(screen.getByTestId(CONFIRM).className).toContain("min-h-11");
    expect(screen.getByTestId(CONFIRM).className).toContain("w-full");
    fireEvent.click(screen.getByTestId(CONFIRM));
    await waitFor(() => expect(commitPlanFiles).toHaveBeenCalledTimes(1));
  });
});
