import { execSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
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

/** Local calendar day, `offsetDays` from today, as YYYY-MM-DD (the server's own time zone). */
export function localDay(offsetDays = 0): string {
  const day = new Date();
  day.setDate(day.getDate() + offsetDays);
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${day.getFullYear()}-${pad(day.getMonth() + 1)}-${pad(day.getDate())}`;
}

export function planFileText(options: {
  title: string;
  board: string;
  date?: string;
  body?: string;
}): string {
  const date = options.date ? `date: ${options.date}\n` : "";
  return `---\nboard: ${options.board}\ntitle: ${options.title}\n${date}---\n\n# ${options.title}\n\n${options.body ?? "Plan body."}\n`;
}

export type PlanFilesRepository = {
  repoDir: string;
  repoName: string;
  read: (relPath: string) => string;
  write: (relPath: string, content: string) => void;
  exists: (relPath: string) => boolean;
  /** Runs a git command in the repository and returns its trimmed stdout. */
  git: (command: string) => string;
  /** Unregisters the repository so later specs of the worker do not scan it. */
  cleanup: () => Promise<void>;
};

/**
 * Creates a git repository with the given files committed on `main` and
 * registers it as a local repository named `name` in the workspace.
 */
export async function createPlanFilesRepository(options: {
  backend: BackendContext;
  apiClient: ApiClient;
  workspaceId: string;
  name: string;
  files: Record<string, string>;
}): Promise<PlanFilesRepository> {
  const { backend, apiClient, workspaceId, name, files } = options;
  const repoDir = path.join(backend.tmpDir, "repos", name);
  const gitEnv = makeGitEnv(backend.tmpDir);
  const git = (command: string) =>
    execSync(`git ${command}`, { cwd: repoDir, env: gitEnv, encoding: "utf8" }).trim();
  const write = (relPath: string, content: string) => {
    const target = path.join(repoDir, relPath);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, content);
  };
  fs.mkdirSync(repoDir, { recursive: true });
  for (const [relPath, content] of Object.entries(files)) write(relPath, content);
  git("init -b main");
  git("add -A");
  git('commit -m "add plans"');
  const { id: repositoryId } = await apiClient.createRepository(workspaceId, repoDir, "main", {
    name,
  });
  return {
    repoDir,
    repoName: name,
    cleanup: () => apiClient.deleteRepository(repositoryId).catch(() => undefined),
    read: (relPath) => fs.readFileSync(path.join(repoDir, relPath), "utf8"),
    write,
    exists: (relPath) => fs.existsSync(path.join(repoDir, relPath)),
    git,
  };
}

export type ConfiguredPlanBoard = {
  boardId: string;
  stepId: (name: string) => string;
};

/**
 * Enables plan files for the workspace from the settings page, creates the
 * Plans board, saves, and runs the first sync pass. The pass runs through the
 * API: saving the configuration already starts a background pass, and the
 * settings button answers 409 while that one holds the workspace lock.
 */
export async function configurePlanBoard(
  page: Page,
  apiClient: ApiClient,
  workspaceId: string,
): Promise<ConfiguredPlanBoard> {
  await page.goto(`/settings/workspaces/${workspaceId}/workflows`);
  const save = await enablePlanFilesAndCreateBoard(page);
  await expect(save).toBeEnabled();
  await save.click();
  await expect(save).toBeDisabled();
  const boardId = await page.getByTestId("plan-files-board").inputValue();
  expect(await syncPlanFiles(apiClient, workspaceId)).toBe("ok");
  const steps = (await apiClient.listWorkflowSteps(boardId)).steps;
  return {
    boardId,
    stepId: (name) => {
      const step = steps.find((candidate) => candidate.name === name);
      if (!step) throw new Error(`Plans board has no ${name} step`);
      return step.id;
    },
  };
}

/**
 * Runs one sync pass through the API and returns its outcome. A pass already
 * running answers 409 and is retried.
 */
export async function syncPlanFiles(apiClient: ApiClient, workspaceId: string): Promise<string> {
  let outcome = "";
  await expect
    .poll(
      async () => {
        const response = await apiClient.rawRequest(
          "POST",
          `/api/v1/plan-files/sync?workspace_id=${encodeURIComponent(workspaceId)}`,
        );
        if (response.status === 200)
          outcome = ((await response.json()) as { outcome: string }).outcome;
        return response.status;
      },
      { message: "plan files sync pass" },
    )
    .toBe(200);
  return outcome;
}

/** The id of the plan task titled `title`, once the sync has created it. */
export async function planTaskId(
  apiClient: ApiClient,
  workspaceId: string,
  title: string,
): Promise<string> {
  let id = "";
  await expect
    .poll(
      async () => {
        const { tasks } = await apiClient.listTasks(workspaceId);
        id = tasks.find((task) => task.title.includes(title))?.id ?? "";
        return id;
      },
      { message: `plan task "${title}"` },
    )
    .not.toBe("");
  return id;
}

/** Full-page review screenshot, written only when PLAN_OPS_SHOTS=1 (to PLAN_OPS_SHOTS_DIR). */
export async function planOpsShot(page: Page, name: string) {
  if (process.env.PLAN_OPS_SHOTS !== "1") return;
  const dir = process.env.PLAN_OPS_SHOTS_DIR ?? path.join(os.tmpdir(), "plan-ops-shots");
  fs.mkdirSync(dir, { recursive: true });
  await page.screenshot({ path: path.join(dir, `${name}.png`), fullPage: true });
}
