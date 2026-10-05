import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";
import { ApiError } from "@/lib/api/client";
import { invalidatePlanBoardConfig } from "@/hooks/domains/plans/use-plan-board";

const getPlanFilesConfig = vi.fn();
const createPlan = vi.fn();
const toastSuccess = vi.fn();
const breakpoint = { isMobile: false };
const repositories = vi.hoisted(() => ({ items: [] as Record<string, unknown>[] }));

vi.mock("@/lib/api/domains/plan-files-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/plan-files-api")>()),
  getPlanFilesConfig: (...a: unknown[]) => getPlanFilesConfig(...a),
  createPlan: (...a: unknown[]) => createPlan(...a),
}));
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile: breakpoint.isMobile }),
}));
vi.mock("@/hooks/domains/workspace/use-repositories", () => ({
  useRepositories: () => ({ repositories: repositories.items, isLoading: false }),
}));
vi.mock("sonner", () => ({ toast: { success: (...a: unknown[]) => toastSuccess(...a) } }));

import { NewPlanDialog } from "./new-plan-dialog";

const WS = "ws-1";
const PLANS_DIR = "docs/plans";
const OTHER_DIR = "docs/superpowers/plans";
const TITLE = "new-plan-title";
const FILE_NAME = "new-plan-file-name";
const CREATE = "new-plan-create";
const ERROR = "new-plan-error";
const EXECUTOR = "new-plan-executor";
const DIRECTORY = "new-plan-directory";

function config(overrides: Record<string, unknown> = {}) {
  return {
    workspace_id: WS,
    enabled: true,
    workflow_id: "wf-plans",
    status_steps: { queued: "step-queued" },
    directories: [PLANS_DIR, OTHER_DIR],
    executor_steps: {},
    ...overrides,
  };
}

function repo(id: string, name: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    workspace_id: WS,
    name,
    source_type: "local",
    local_path: `/work/${name}`,
    ...overrides,
  };
}

function mount(onOpenChange = vi.fn()) {
  render(
    <StateProvider initialState={{ features: { ...defaultState.features, planFiles: true } }}>
      <NewPlanDialog workspaceId={WS} open onOpenChange={onOpenChange} />
    </StateProvider>,
  );
  return onOpenChange;
}

async function typeTitle(value: string) {
  fireEvent.change(await screen.findByTestId(TITLE), { target: { value } });
}

async function submit() {
  const button = screen.getByTestId(CREATE) as HTMLButtonElement;
  await waitFor(() => expect(button.disabled).toBe(false));
  fireEvent.click(button);
}

beforeEach(() => {
  breakpoint.isMobile = false;
  invalidatePlanBoardConfig(WS);
  getPlanFilesConfig.mockReset().mockResolvedValue(config());
  createPlan
    .mockReset()
    .mockResolvedValue({ task_id: "t1", repository_id: "r1", rel_path: "docs/plans/x.md" });
  toastSuccess.mockReset();
  repositories.items = [
    repo("r-remote", "aaa-remote", { source_type: "github", local_path: "" }),
    repo("r1", "dmhive"),
  ];
});
afterEach(() => cleanup());

describe("NewPlanDialog desktop", () => {
  it("shows the fields of the form and hides the executor without executor steps", async () => {
    mount();
    expect(await screen.findByTestId("new-plan-dialog")).toBeTruthy();
    for (const id of [
      "new-plan-repository",
      DIRECTORY,
      TITLE,
      FILE_NAME,
      "new-plan-priority",
      "new-plan-body",
    ]) {
      expect(screen.getByTestId(id)).toBeTruthy();
    }
    expect(screen.queryByTestId(EXECUTOR)).toBeNull();
  });

  it("lists only local repositories and the configured directories", async () => {
    mount();
    const repoSelect = (await screen.findByTestId("new-plan-repository")) as HTMLSelectElement;
    expect(Array.from(repoSelect.options).map((o) => o.textContent)).toEqual(["dmhive"]);
    await waitFor(() =>
      expect(
        Array.from((screen.getByTestId(DIRECTORY) as HTMLSelectElement).options).map(
          (o) => o.value,
        ),
      ).toEqual([PLANS_DIR, OTHER_DIR]),
    );
  });

  it("keeps Create disabled until the title has text", async () => {
    mount();
    const create = (await screen.findByTestId(CREATE)) as HTMLButtonElement;
    expect(create.disabled).toBe(true);
    await typeTitle("   ");
    expect(create.disabled).toBe(true);
    await typeTitle("Fix totals");
    await waitFor(() => expect(create.disabled).toBe(false));
  });
});

describe("NewPlanDialog desktop submission", () => {
  it("creates with the defaults and closes with a confirmation", async () => {
    const onOpenChange = mount();
    await typeTitle("Fix: totals #2");
    await submit();
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(createPlan).toHaveBeenCalledWith(
      { repository_id: "r1", directory: PLANS_DIR, title: "Fix: totals #2", priority: "medium" },
      { workspaceId: WS },
    );
    expect(toastSuccess).toHaveBeenCalledWith("Plan created");
  });

  it("sends the chosen directory, file name, priority, executor, and body", async () => {
    getPlanFilesConfig.mockResolvedValue(
      config({ executor_steps: { s1: "claude", s2: "codex", s3: "claude" } }),
    );
    mount();
    await typeTitle("Plan");
    await waitFor(() => expect(screen.getByTestId(EXECUTOR)).toBeTruthy());
    const executor = screen.getByTestId(EXECUTOR) as HTMLSelectElement;
    expect(Array.from(executor.options).map((o) => o.textContent)).toEqual([
      "None",
      "claude",
      "codex",
    ]);
    fireEvent.change(executor, { target: { value: "name:codex" } });
    fireEvent.change(screen.getByTestId(DIRECTORY), {
      target: { value: OTHER_DIR },
    });
    fireEvent.change(screen.getByTestId(FILE_NAME), { target: { value: " my-plan.md " } });
    fireEvent.change(screen.getByTestId("new-plan-priority"), { target: { value: "high" } });
    fireEvent.change(screen.getByTestId("new-plan-body"), { target: { value: "Details" } });
    await submit();
    await waitFor(() =>
      expect(createPlan).toHaveBeenCalledWith(
        {
          repository_id: "r1",
          directory: OTHER_DIR,
          title: "Plan",
          file_name: "my-plan.md",
          priority: "high",
          executor: "codex",
          body: "Details",
        },
        { workspaceId: WS },
      ),
    );
  });

  it("refuses an invalid file name before sending", async () => {
    mount();
    await typeTitle("Plan");
    fireEvent.change(screen.getByTestId(FILE_NAME), { target: { value: "../x" } });
    expect((screen.getByTestId(CREATE) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByTestId("new-plan-file-name-error")).toBeTruthy();
    fireEvent.change(screen.getByTestId(FILE_NAME), { target: { value: "x.md" } });
    await waitFor(() =>
      expect((screen.getByTestId(CREATE) as HTMLButtonElement).disabled).toBe(false),
    );
    expect(screen.queryByTestId("new-plan-file-name-error")).toBeNull();
  });
});

describe("NewPlanDialog desktop errors", () => {
  it("keeps the dialog open and names the collision when the file exists", async () => {
    createPlan.mockRejectedValue(new ApiError("exists", 409, { code: "file_exists" }));
    const onOpenChange = mount();
    await typeTitle("Same");
    await submit();
    expect((await screen.findByTestId(ERROR)).textContent).toBe(
      "A file with this name already exists. Choose another name.",
    );
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(toastSuccess).not.toHaveBeenCalled();
  });

  it("shows the refusal and the generic failure copy", async () => {
    createPlan.mockRejectedValueOnce(new ApiError("bad", 400, { code: "invalid_plan" }));
    createPlan.mockRejectedValueOnce(new ApiError("gone", 404, { code: "repository_not_found" }));
    createPlan.mockRejectedValueOnce(new ApiError("boom", 500, { error: "boom" }));
    mount();
    await typeTitle("Plan");
    await submit();
    expect((await screen.findByTestId(ERROR)).textContent).toContain("refused");
    fireEvent.click(screen.getByTestId(CREATE));
    await waitFor(() =>
      expect(screen.getByTestId(ERROR).textContent).toContain("no longer available"),
    );
    fireEvent.click(screen.getByTestId(CREATE));
    await waitFor(() =>
      expect(screen.getByTestId(ERROR).textContent).toContain("could not be created"),
    );
  });

  it("closes without creating on Cancel", async () => {
    const onOpenChange = mount();
    fireEvent.click(await screen.findByTestId("new-plan-cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(createPlan).not.toHaveBeenCalled();
  });

  it("explains a workspace without a local repository and cannot create", async () => {
    repositories.items = [repo("r-remote", "remote", { source_type: "github", local_path: "" })];
    mount();
    expect(await screen.findByTestId("new-plan-no-repository")).toBeTruthy();
    await typeTitle("Plan");
    expect((screen.getByTestId(CREATE) as HTMLButtonElement).disabled).toBe(true);
  });
});

describe("NewPlanDialog phone", () => {
  beforeEach(() => {
    breakpoint.isMobile = true;
  });

  it("is a full-height surface with a scroll region and a sticky Create button", async () => {
    mount();
    const drawer = await screen.findByTestId("new-plan-drawer");
    expect(drawer.className).toContain("h-[calc(100dvh");
    expect(screen.queryByTestId("new-plan-dialog")).toBeNull();
    expect(screen.getByTestId("new-plan-scroll").className).toContain("overflow-y-auto");
    const footer = screen.getByTestId("new-plan-footer");
    expect(footer.className).toContain("sticky");
    expect(footer.contains(screen.getByTestId(CREATE))).toBe(true);
  });

  it("uses full-width touch controls", async () => {
    mount();
    await screen.findByTestId("new-plan-drawer");
    for (const id of [TITLE, FILE_NAME, "new-plan-repository", DIRECTORY, "new-plan-priority"]) {
      expect(screen.getByTestId(id).className).toContain("w-full");
    }
    const create = screen.getByTestId(CREATE);
    expect(create.className).toContain("min-h-11");
    expect(create.className).toContain("w-full");
  });

  it("creates and closes like the desktop form", async () => {
    const onOpenChange = mount();
    await typeTitle("From my phone");
    await submit();
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(createPlan).toHaveBeenCalledWith(
      expect.objectContaining({ title: "From my phone", repository_id: "r1" }),
      { workspaceId: WS },
    );
  });
});
