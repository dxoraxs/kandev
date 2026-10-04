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
