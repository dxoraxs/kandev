import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { expect, type Locator, type Page } from "@playwright/test";
import type { BackendContext } from "../fixtures/backend";
import type { ApiClient } from "./api-client";
import { makeGitEnv } from "./git-helper";
import { settledBoundingBox } from "./settled-box";

/** Env that turns the plan files runtime flag on for a spec's backend. */
export const PLAN_FILES_ENV = { KANDEV_FEATURES_PLAN_FILES: "true" };

export const PLAN_RELATIVE_PATH = "docs/plans/e2e-plan.md";

export type PlanRepository = {
  repoDir: string;
  planPath: string;
  writePlan: (title: string, board: string) => void;
  readPlan: () => string;
};

function planContent(title: string, board: string): string {
  return `---\nboard: ${board}\ntitle: ${title}\n---\n\n# ${title}\n\nPlan body.\n`;
}

/**
 * Creates a git repository inside the backend's temp dir with one plan file
 * (`board: queued`), registers it as a local repository in the workspace, and
 * returns helpers to rewrite or read the file on disk.
 */
export async function createPlanRepository(options: {
  backend: BackendContext;
  apiClient: ApiClient;
  workspaceId: string;
  title: string;
}): Promise<PlanRepository> {
  const { backend, apiClient, workspaceId, title } = options;
  const repoDir = path.join(backend.tmpDir, "repos", "e2e-plan-repo");
  const planPath = path.join(repoDir, PLAN_RELATIVE_PATH);
  fs.mkdirSync(path.dirname(planPath), { recursive: true });
  fs.writeFileSync(planPath, planContent(title, "queued"));
  const gitEnv = makeGitEnv(backend.tmpDir);
  execSync("git init -b main", { cwd: repoDir, env: gitEnv });
  execSync("git add -A", { cwd: repoDir, env: gitEnv });
  execSync('git commit -m "add plan"', { cwd: repoDir, env: gitEnv });
  await apiClient.createRepository(workspaceId, repoDir);
  return {
    repoDir,
    planPath,
    writePlan: (nextTitle, board) => fs.writeFileSync(planPath, planContent(nextTitle, board)),
    readPlan: () => fs.readFileSync(planPath, "utf8"),
  };
}

/**
 * Drags a kanban card onto a column with real pointer events. The board
 * reserves extra room while a drag is active, which shifts the columns, so the
 * target is re-measured while the drag is active.
 */
export async function dragCardToColumn(page: Page, card: Locator, column: Locator) {
  const from = await settledBoundingBox(card);
  const startX = from.x + from.width / 2;
  const startY = from.y + from.height / 2;
  await page.mouse.move(startX, startY);
  await page.mouse.down();
  // Exceed the 8px PointerSensor activation distance so the drag starts.
  await page.mouse.move(startX + 12, startY, { steps: 3 });
  await expect(page.getByTestId("desktop-kanban-drag-end-reserve")).toBeVisible();
  // The columns keep shifting for a few frames after the drag starts: re-aim at
  // the column's current position until dnd-kit reports it as the drop target.
  await expect
    .poll(async () => {
      const box = await column.boundingBox();
      if (!box) return false;
      await page.mouse.move(box.x + box.width / 2, box.y + Math.min(box.height / 2, 80));
      return ((await column.getAttribute("class")) ?? "").includes("bg-primary/5");
    })
    .toBe(true);
  await page.mouse.up();
}

/** Turns the section on and creates the Plans board; returns the Save button. */
export async function enablePlanFilesAndCreateBoard(page: Page) {
  await page.getByTestId("plan-files-section").waitFor({ state: "visible" });
  await page.getByTestId("plan-files-enabled").click();
  await page.getByTestId("plan-files-create-board").click();
  return page.getByTestId("plan-files-save");
}

export const UNADAPTED_FILE_COUNT = 2;

export type UnadaptedRepository = {
  repoDir: string;
  repoName: string;
  /** Present only when the repository was registered through the API. */
  repositoryId?: string;
  cleanup: () => Promise<void>;
};

/** Ids of workspace repositories saved from the UI for `repoDir` (its resolved path). */
async function savedRepositoryIds(
  apiClient: ApiClient,
  workspaceId: string,
  repoDir: string,
): Promise<string[]> {
  const response = await apiClient.rawRequest(
    "GET",
    `/api/v1/workspaces/${workspaceId}/repositories`,
  );
  if (!response.ok) return [];
  const body = (await response.json()) as {
    repositories: Array<{ id: string; local_path: string }>;
  };
  const resolved = fs.existsSync(repoDir) ? fs.realpathSync(repoDir) : repoDir;
  return body.repositories
    .filter((repo) => repo.local_path === resolved || repo.local_path === repoDir)
    .map((repo) => repo.id);
}

/**
 * Creates a git repository whose `docs/plans` holds plan files without a board
 * header (the shape adaptation exists for). With `register`, it is added to the
 * workspace through the API under the backend's temp dir; otherwise it lives
 * outside the backend home so the settings "Add Local Repository" flow accepts it.
 */
export async function createUnadaptedRepository(options: {
  backend: BackendContext;
  apiClient: ApiClient;
  workspaceId: string;
  name: string;
  register: boolean;
}): Promise<UnadaptedRepository> {
  const { backend, apiClient, workspaceId, name, register } = options;
  const parent = register
    ? path.join(backend.tmpDir, "repos")
    : fs.mkdtempSync(path.join(path.dirname(backend.tmpDir), "kandev-e2e-adapt-"));
  const repoDir = path.join(parent, name);
  const plansDir = path.join(repoDir, "docs", "plans");
  fs.mkdirSync(plansDir, { recursive: true });
  for (let index = 1; index <= UNADAPTED_FILE_COUNT; index += 1) {
    fs.writeFileSync(
      path.join(plansDir, `legacy-${index}.md`),
      `# Legacy plan ${index}\n\nNo header.\n`,
    );
  }
  const gitEnv = makeGitEnv(backend.tmpDir);
  execSync("git init -b main", { cwd: repoDir, env: gitEnv });
  execSync("git add -A", { cwd: repoDir, env: gitEnv });
  execSync('git commit -m "add legacy plans"', { cwd: repoDir, env: gitEnv });
  const repositoryId = register
    ? (await apiClient.createRepository(workspaceId, repoDir, "main", { name })).id
    : undefined;
  return {
    repoDir,
    repoName: name,
    repositoryId,
    cleanup: async () => {
      const ids = repositoryId
        ? [repositoryId]
        : await savedRepositoryIds(apiClient, workspaceId, repoDir);
      for (const id of ids) {
        await apiClient.rawRequest("DELETE", `/api/v1/repositories/${id}`).catch(() => undefined);
      }
      fs.rmSync(register ? repoDir : parent, { recursive: true, force: true });
    },
  };
}

/**
 * Adds `repoPath` as a local repository through the settings Repositories tab
 * and saves it, which is the moment the plan adaptation offer can appear.
 */
export async function addRepositoryThroughSettings(
  page: Page,
  workspaceId: string,
  repoPath: string,
) {
  await page.goto(`/settings/workspaces/${workspaceId}/repositories`);
  await page.getByRole("button", { name: "Add Local Repository" }).click();
  const dialog = page.getByRole("dialog", { name: "Add Local Repository" });
  await expect(dialog).toBeVisible();
  await dialog.getByPlaceholder("/absolute/path/to/repository").fill(repoPath);
  await dialog.getByRole("button", { name: "Validate", exact: true }).click();
  await expect(dialog.getByText("Valid git repository", { exact: true })).toBeVisible();
  await dialog.getByRole("button", { name: "Use Repository" }).click();
  await expect(dialog).toBeHidden();
  await page
    .getByTestId("settings-floating-save")
    .getByRole("button", { name: "Save changes" })
    .click();
}

/** Plan adaptation launches a session, which needs a workspace default agent profile. */
export async function useDefaultAgentProfile(
  apiClient: ApiClient,
  workspaceId: string,
  agentProfileId: string,
) {
  await apiClient.updateWorkspace(workspaceId, { default_agent_profile_id: agentProfileId });
}
